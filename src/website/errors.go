package website

import (
	"fmt"
	"net/http"
	"strings"

	"git.handmade.network/hmn/hmn/src/templates"
)

func FourOhFour(c *RequestContext) ResponseData {
	var res ResponseData
	res.StatusCode = http.StatusNotFound

	if c.Req.Header["Accept"] != nil && strings.Contains(c.Req.Header["Accept"][0], "text/html") {
		templateData := struct {
			templates.BaseData
			Wanted string
		}{
			BaseData: getBaseData(c, "Page not found", nil),
			Wanted:   c.FullUrl(),
		}
		res.MustWriteTemplate("404.html", templateData, c.Perf)
	} else {
		res.Write([]byte("Not Found"))
	}
	return res
}

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
