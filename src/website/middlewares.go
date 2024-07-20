package website

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"git.handmade.network/hmn/hmn/src/auth"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/perf"
	"git.handmade.network/hmn/hmn/src/utils"
)

func panicCatcherMiddleware(h Handler) Handler {
	return func(c *RequestContext) (res ResponseData) {
		defer func() {
			if recovered := recover(); recovered != nil {
				maybeError, ok := recovered.(*error)
				var err error
				if ok {
					err = *maybeError
				} else {
					err = oops.New(nil, fmt.Sprintf("Recovered from panic with value: %v", recovered))
				}
				res = c.ErrorResponse(http.StatusInternalServerError, err)
			}
		}()

		return h(c)
	}
}

func trackRequestPerf(perfCollector *perf.PerfCollector) func(Handler) Handler {
	return func(h Handler) Handler {
		return func(c *RequestContext) ResponseData {
			c.Perf = perf.MakeNewRequestPerf(c.Route, c.Req.Method, c.Req.URL.Path)
			c.PerfCollector = perfCollector
			defer func() {
				c.Perf.EndRequest()
				log := logging.Info()
				blockStack := make([]time.Time, 0)
				for i, block := range c.Perf.Blocks {
					for len(blockStack) > 0 && block.End.After(blockStack[len(blockStack)-1]) {
						blockStack = blockStack[:len(blockStack)-1]
					}
					log.Str(fmt.Sprintf("[%4.d] At %9.2fms", i, c.Perf.MsFromStart(&block)), fmt.Sprintf("%*.s[%s] %s (%.4fms)", len(blockStack)*2, "", block.Category, block.Description, block.DurationMs()))
					blockStack = append(blockStack, block.End)
				}
				log.Msg(fmt.Sprintf("Served [%s] %s in %.4fms", c.Perf.Method, c.Perf.Path, float64(c.Perf.End.Sub(c.Perf.Start).Nanoseconds())/1000/1000))
				perfCollector.SubmitRun(c.Perf)
			}()

			return h(c)
		}
	}
}

func needsAuth(h Handler) Handler {
	return func(c *RequestContext) ResponseData {
		if c.CurrentUser == nil {
			return c.Redirect(hmnurl.BuildLoginPage(c.FullUrl()), http.StatusSeeOther)
		}

		return h(c)
	}
}

func adminsOnly(h Handler) Handler {
	return func(c *RequestContext) ResponseData {
		if c.CurrentUser == nil || !c.CurrentUser.IsStaff {
			return FourOhFour(c)
		}

		return h(c)
	}
}

func educationBetaTestersOnly(h Handler) Handler {
	return func(c *RequestContext) ResponseData {
		if c.CurrentUser == nil || !c.CurrentUser.CanSeeUnpublishedEducationContent() {
			return FourOhFour(c)
		}

		return h(c)
	}
}

func educationAuthorsOnly(h Handler) Handler {
	return func(c *RequestContext) ResponseData {
		if c.CurrentUser == nil || !c.CurrentUser.CanAuthorEducation() {
			return FourOhFour(c)
		}

		return h(c)
	}
}

func csrfMiddleware(h Handler) Handler {
	// CSRF mitigation actions per the OWASP cheat sheet:
	// https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html
	return func(c *RequestContext) ResponseData {
		c.Req.ParseMultipartForm(100 * 1024 * 1024)
		csrfToken := c.Req.Form.Get(auth.CSRFFieldName)
		if csrfToken != c.CurrentSession.CSRFToken {
			c.Logger.Warn().Str("userId", c.CurrentUser.Username).Msg("user failed CSRF validation - potential attack?")

			res := c.Redirect("/", http.StatusSeeOther)
			logoutUser(c, &res)

			return res
		}

		return h(c)
	}
}

func securityTimerMiddleware(duration time.Duration, h Handler) Handler {
	// NOTE(asaf): Will make sure that the request takes at least `duration` to finish. Adds a 10% random duration.
	return func(c *RequestContext) ResponseData {
		additionalDuration := time.Duration(rand.Int63n(utils.Max(1, int64(duration)/10)))
		timer := time.NewTimer(duration + additionalDuration)
		res := h(c)
		select {
		case <-c.Done():
		case <-timer.C:
		}
		return res
	}
}

func logContextErrors(c *RequestContext, errs ...error) {
	for _, err := range errs {
		c.Logger.Error().Timestamp().Stack().Str("Requested", c.FullUrl()).Err(err).Msg("error occurred during request")
	}
}

func logContextErrorsMiddleware(h Handler) Handler {
	return func(c *RequestContext) ResponseData {
		res := h(c)
		logContextErrors(c, res.Errors...)
		return res
	}
}
