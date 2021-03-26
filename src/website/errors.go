package website

import "fmt"

// A SafeError can be used to wrap another error and explicitly provide
// an error message that is safe to show to a user. This allows the original
// error to easily be logged and for servers to consistently return errors
// in a standard format, without having to worry about leaking sensitive
// info (assuming you use the right middleware!).
type SafeError struct {
	Wrapped error
	Msg     string
}

func NewSafeError(err error, msg string, args ...interface{}) error {
	return &SafeError{
		Wrapped: err,
		Msg:     fmt.Sprintf(msg, args...),
	}
}

func (s *SafeError) Error() string {
	return s.Msg
}
