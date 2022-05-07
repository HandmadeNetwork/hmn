package utils

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"git.handmade.network/hmn/hmn/src/oops"
)

// Returns the provided value, or a default value if the input was zero.
func OrDefault[T comparable](v T, def T) T {
	var zero T
	if v == zero {
		return def
	} else {
		return v
	}
}

func IntMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func IntMax(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func IntClamp(min, t, max int) int {
	return IntMax(min, IntMin(t, max))
}

func Int64Max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func NumPages(numThings, thingsPerPage int) int {
	return IntMax(int(math.Ceil(float64(numThings)/float64(thingsPerPage))), 1)
}

/*
Recover a panic and convert it to a returned error. Call it like so:

	func MyFunc() (err error) {
		defer utils.RecoverPanicAsError(&err)
	}

If an error was already present, the panicked error will take precedence. Unfortunately there's
no good way to include both errors because you can't really have two chains of errors and still
play nice with the standard library's Unwrap behavior. But most of the time this shouldn't be an
issue, since the panic will probably occur before a meaningful error value was set.
*/
func RecoverPanicAsError(err *error) {
	if r := recover(); r != nil {
		var recoveredErr error
		if rerr, ok := r.(error); ok {
			recoveredErr = rerr
		} else {
			recoveredErr = fmt.Errorf("panic with value: %v", r)
		}
		*err = oops.New(recoveredErr, "panic recovered as error")
	}
}

var ErrSleepInterrupted = errors.New("sleep interrupted by context cancellation")

func SleepContext(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ErrSleepInterrupted
	case <-time.After(d):
		return nil
	}
}
