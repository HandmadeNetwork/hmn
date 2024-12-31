package utils

import (
	"time"
)

// An equivalent to [time.Ticker] that also ticks immediately upon creation.
type InstaTicker struct {
	C <-chan time.Time

	done   chan struct{}
	ticker *time.Ticker
}

func NewInstaTicker(d time.Duration) *InstaTicker {
	ticker := time.NewTicker(d)
	c := make(chan time.Time)
	done := make(chan struct{})
	go func() {
		// Send initial tick
		select {
		case <-done:
			return
		case c <- time.Now():
		}

		// Proxy the ticker's channel
		for {
			select {
			case <-done:
				return
			case t := <-ticker.C:
				c <- t
			}
		}
	}()
	return &InstaTicker{
		C:      c,
		done:   done,
		ticker: ticker,
	}
}

// Stops the ticker from ticking. Calling Stop more than once will panic.
func (it *InstaTicker) Stop() {
	it.ticker.Stop()
	close(it.done)
}
