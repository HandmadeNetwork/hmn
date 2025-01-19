package utils

import (
	"errors"
	"testing"

	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/stretchr/testify/assert"
)

type MyError struct{}

func (err *MyError) Error() string {
	return "I want to get off MR BONES WILD RIDE"
}

func TestMust(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		f := func() error { return nil }
		Must(f())
	})
	t.Run("non-nil error", func(t *testing.T) {
		f := func() error { return &MyError{} }
		assert.Panics(t, func() {
			Must(f())
		})
	})
	t.Run("nil *MyError", func(t *testing.T) {
		f := func() *MyError { return nil }
		Must(f())
	})
	t.Run("non-nil *MyError", func(t *testing.T) {
		f := func() *MyError { return &MyError{} }
		assert.Panics(t, func() {
			Must(f())
		})
	})
}

func TestMust1(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		f := func() (int, error) { return 0, nil }
		a := Must1(f())
		assert.Equal(t, 0, a)
	})
	t.Run("non-nil error", func(t *testing.T) {
		f := func() (int, error) { return 0, &MyError{} }
		assert.Panics(t, func() {
			Must1(f())
		})
	})
	t.Run("nil *MyError", func(t *testing.T) {
		f := func() (int, *MyError) { return 0, nil }
		a := Must1(f())
		assert.Equal(t, 0, a)
	})
	t.Run("non-nil *MyError", func(t *testing.T) {
		f := func() (int, *MyError) { return 0, &MyError{} }
		assert.Panics(t, func() {
			Must1(f())
		})
	})
}

func TestMust2(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		f := func() (int, int, error) { return 0, 1, nil }
		a, b := Must2(f())
		assert.Equal(t, 0, a)
		assert.Equal(t, 1, b)
	})
	t.Run("non-nil error", func(t *testing.T) {
		f := func() (int, int, error) { return 0, 1, &MyError{} }
		assert.Panics(t, func() {
			Must2(f())
		})
	})
	t.Run("nil *MyError", func(t *testing.T) {
		f := func() (int, int, *MyError) { return 0, 1, nil }
		a, b := Must2(f())
		assert.Equal(t, 0, a)
		assert.Equal(t, 1, b)
	})
	t.Run("non-nil *MyError", func(t *testing.T) {
		f := func() (int, int, *MyError) { return 0, 1, &MyError{} }
		assert.Panics(t, func() {
			Must2(f())
		})
	})
}

var sentinelError = errors.New("sentinel")

func TestRecoverPanicAsError(t *testing.T) {
	t.Run("no panic, no error", func(t *testing.T) {
		f := func() (err error) {
			defer RecoverPanicAsError(&err)
			return nil
		}
		err := f()
		assert.Nil(t, err)
	})
	t.Run("no panic, error", func(t *testing.T) {
		f := func() (err error) {
			defer RecoverPanicAsError(&err)
			return sentinelError
		}
		err := f()
		assert.True(t, errors.Is(err, sentinelError))
	})
	t.Run("panic, no error", func(t *testing.T) {
		f := func() (err error) {
			defer RecoverPanicAsError(&err)
			panic("blerp")
		}
		err := f()
		var asOops *oops.Error
		assert.ErrorContains(t, err, "blerp")
		assert.True(t, errors.As(err, &asOops))
	})
	t.Run("panic, error", func(t *testing.T) {
		f := func() (err error) {
			defer RecoverPanicAsError(&err)
			err = sentinelError
			panic("blerp")
		}
		err := f()
		var asOops *oops.Error
		assert.ErrorContains(t, err, "blerp")
		assert.ErrorContains(t, err, "sentinel")
		assert.True(t, errors.As(err, &asOops))
		assert.True(t, errors.Is(err, sentinelError))
	})
}
