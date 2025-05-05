package discord

import (
	"context"
	"errors"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/utils"
)

var limiterLog = logging.GlobalLogger().With().
	Str("module", "discord").
	Str("discord actor", "rate limiter").
	Logger()

var buckets sync.Map      // map[route name]bucket name
var rateLimiters sync.Map // map[bucket name]*restRateLimiter
var limiterInitMutex sync.Mutex

type restRateLimiter struct {
	requests chan struct{}
	refills  chan rateLimiterRefill
}

type rateLimiterRefill struct {
	resetAfter  time.Duration
	maxRequests int
}

/*
Whenever we send a request, we must sleep until this time
(if it is in the future, of course). This is a quick and
dirty way to pause all sending in case of a global rate
limit.

I could put a mutex on this but I don't think it's actually
a problem to have race conditions here. Just set it when
you get throttled. EZ.
*/
var globalRateLimitTime time.Time

type rateLimitHeaders struct {
	Bucket     string
	Limit      int
	Remaining  int
	ResetAfter time.Duration
}

func parseRateLimitHeaders(header http.Header) (rateLimitHeaders, bool) {
	var err error

	bucket := header.Get("X-RateLimit-Bucket")
	var limit int
	var remaining int
	var resetAfter time.Duration

	limitStr := header.Get("X-RateLimit-Limit")
	if limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			limiterLog.Error().
				Err(err).
				Str("value", limitStr).
				Msg("failed to parse X-RateLimit-Limit header")
			return rateLimitHeaders{}, false
		}
	}

	remainingStr := header.Get("X-RateLimit-Remaining")
	if remainingStr != "" {
		remaining, err = strconv.Atoi(remainingStr)
		if err != nil {
			limiterLog.Error().
				Err(err).
				Str("value", remainingStr).
				Msg("failed to parse X-RateLimit-Remaining header")
			return rateLimitHeaders{}, false
		}
	}

	resetAfterStr := header.Get("X-RateLimit-Reset-After")
	if resetAfterStr != "" {
		resetAfterSeconds, err := strconv.ParseFloat(resetAfterStr, 64)
		if err != nil {
			limiterLog.Error().
				Err(err).
				Str("value", resetAfterStr).
				Msg("failed to parse X-RateLimit-Reset-After header")
			return rateLimitHeaders{}, false
		}
		resetAfter = time.Duration(math.Ceil(resetAfterSeconds)) * time.Second
	}

	return rateLimitHeaders{
		Bucket:     bucket,
		Limit:      limit,
		Remaining:  remaining,
		ResetAfter: resetAfter,
	}, true
}

func createLimiter(headers rateLimitHeaders, routeName string) {
	limiterInitMutex.Lock()
	defer limiterInitMutex.Unlock()

	buckets.Store(routeName, headers.Bucket)
	ilimiter, loaded := rateLimiters.LoadOrStore(headers.Bucket, &restRateLimiter{
		requests: make(chan struct{}, 1000), // presumably this is big enough to handle bursts
		refills:  make(chan rateLimiterRefill),
	})
	if !loaded {
		limiter := ilimiter.(*restRateLimiter)

		log := limiterLog.With().Str("bucket", headers.Bucket).Logger()

	prefillloop:
		// Pre-fill the limiter with remaining requests
		for i := 0; i < headers.Remaining; i++ {
			select {
			case limiter.requests <- struct{}{}:
			default:
				log.Warn().
					Int("remaining", headers.Remaining).
					Msg("rate limiting channel was too small; you should increase the default capacity")
				break prefillloop
			}
		}

		/*
			Start the refiller for this bucket. It waits for a request to tell
			it when to next	reset the rate limit, and how full to fill the bucket.
			It then sleeps and refills the bucket, just like it should :)
		*/
		go func() {
			for {
				// Wake up on the first request after refilling
				refill := <-limiter.refills

				// Sleep for the remainder of the bucket's time
				time.Sleep(refill.resetAfter)

			drainloop:
				// drain the bucket
				for {
					select {
					case <-limiter.requests:
					default:
						break drainloop
					}
				}

			refillloop:
				// refill it with the max number of requests
				for i := 0; i < refill.maxRequests; i++ {
					select {
					case limiter.requests <- struct{}{}:
					default:
						log.Warn().
							Int("maxRequests", refill.maxRequests).
							Msg("rate limiting channel was too small; you should increase the default capacity")
						break refillloop
					}
				}

				// And then we wait again to hear about our next
				// bucket's worth of requests.
			}
		}()

		// Tell the refiller about its first refill
		limiter.refills <- rateLimiterRefill{
			resetAfter:  headers.ResetAfter,
			maxRequests: headers.Limit,
		}
	}
}

func (l *restRateLimiter) update(headers rateLimitHeaders) {
	refill := rateLimiterRefill{
		resetAfter:  headers.ResetAfter,
		maxRequests: headers.Limit,
	}

	/*
		Tell the refiller about this request. If the refiller is already
		busy sleeping, this will have no effect, which is what we want.
		(It's already sleeping for as long as it needs to.)
	*/
	select {
	case l.refills <- refill:
	default:
	}
}

func doWithRateLimiting(ctx context.Context, routeName string, getReq func(ctx context.Context) *http.Request) (*http.Response, error) {
	var bucket string
	ibucket, ok := buckets.Load(routeName)
	if ok {
		bucket = ibucket.(string)
	}

	for {
		var limiter *restRateLimiter
		if bucket != "" {
			ilimiter, ok := rateLimiters.Load(bucket)
			if ok {
				limiter = ilimiter.(*restRateLimiter)
			}
		}

		if globalRateLimitTime.After(time.Now()) {
			// oh boy, global rate limit, pause until the coast is clear
			err := utils.SleepContext(ctx, globalRateLimitTime.Sub(time.Now())+1*time.Second)
			if err != nil {
				return nil, err
			}
		}

		if limiter != nil {
			select {
			case <-limiter.requests:
			case <-ctx.Done():
				return nil, errors.New("request interrupted during rate limiting")
			}
		}

		res, err := httpClient.Do(getReq(ctx))
		if err != nil {
			return nil, err
		}

		headers, headersOk := parseRateLimitHeaders(res.Header)
		if headersOk {
			if limiter == nil || headers.Bucket != bucket {
				createLimiter(headers, routeName)
			} else {
				limiter.update(headers)
			}
		}

		if res.StatusCode == 429 {
			if res.Header.Get("X-RateLimit-Global") != "" {
				// globally rate limited
				logging.ExtractLogger(ctx).Warn().Msg("got globally rate limited by Discord")
				retryAfter, err := strconv.Atoi(res.Header.Get("Retry-After"))
				if err == nil {
					globalRateLimitTime = time.Now().Add(time.Duration(retryAfter) * time.Second)
				} else {
					// well this is bad, just sleep for 60 seconds and pray that it's long enough
					logging.ExtractLogger(ctx).Warn().
						Err(err).
						Msg("got globally rate limited but couldn't determine how long to wait")
					globalRateLimitTime = time.Now().Add(60 * time.Second)
				}
			} else {
				// locally rate limited

				/*
					Despite our best efforts, we ended up rate limited anyway.
					Simply wait the amount of time Discord asks, and then try
					again. On the next go-around, hopefully we'll either succeed
					or have a rate limiter initialized and ready to go.
				*/
				logging.ExtractLogger(ctx).Warn().Msg("got rate limited by Discord")
				if headersOk {
					err := utils.SleepContext(ctx, headers.ResetAfter)
					if err != nil {
						return nil, err
					}
				} else {
					logging.ExtractLogger(ctx).Warn().Msg("got rate limited, but didn't have the headers??")
					err := utils.SleepContext(ctx, 1*time.Second)
					if err != nil {
						return nil, err
					}
				}
			}
			continue
		}

		return res, nil
	}
}
