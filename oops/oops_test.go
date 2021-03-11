package oops

import (
	"errors"
	"testing"

	"github.com/rs/zerolog"
)

var SampleErrorValue = errors.New("some error occurred that you should handle")

type SampleErrorType struct {
	Message string
}

func (s SampleErrorType) Error() string {
	return s.Message
}

func init() {
	zerolog.ErrorStackMarshaler = ZerologStackMarshaler
}

func TestNew(t *testing.T) {
	t.Run("errors.Is", func(t *testing.T) {
		err := New(SampleErrorValue, "test error")
		if !errors.Is(err, SampleErrorValue) {
			t.Fatal("error did not appear to wrap the sample value")
		}
	})
	t.Run("errors.As", func(t *testing.T) {
		err := New(SampleErrorType{Message: "some fancy error type has occurred"}, "test error")
		var sErr SampleErrorType
		if !errors.As(err, &sErr) {
			t.Fatal("error did not appear to wrap the sample error type")
		}
	})
}
