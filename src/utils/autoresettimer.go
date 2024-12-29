package utils

import (
	"context"
	"time"
)

type AutoResetTimer struct {
	C chan struct{}
}

func MakeAutoResetTimer(ctx context.Context, dur time.Duration, triggerImmediately bool) *AutoResetTimer {
	res := &AutoResetTimer{
		C: make(chan struct{}, 0),
	}

	go func() {
		var timer *time.Timer

		defer func() {
			if timer != nil {
				stopped := timer.Stop()
				if !stopped {
					select {
					case <-timer.C:
					default:
					}
				}
			}
			close(res.C)
		}()

		if triggerImmediately {
			select {
			case res.C <- struct{}{}:
			case <-ctx.Done():
				return
			}
		}
		for {
			if timer == nil {
				timer = time.NewTimer(dur)
			} else {
				timer.Reset(dur)
			}

			select {
			case <-timer.C:
				select {
				case res.C <- struct{}{}:
				case <-ctx.Done():
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return res
}
