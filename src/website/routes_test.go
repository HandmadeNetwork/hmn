package website

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestLogContextErrors(t *testing.T) {
	err1 := errors.New("test error 1")
	err2 := errors.New("test error 2")

	defer zerolog.SetGlobalLevel(zerolog.GlobalLevel())
	zerolog.SetGlobalLevel(zerolog.TraceLevel)

	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	logger.Print("sanity check")

	assert.Contains(t, buf.String(), "sanity check")

	router := &Router{}
	routes := RouteBuilder{
		Router: router,
		Middlewares: []Middleware{
			func(h Handler) Handler {
				return func(c *RequestContext) (res ResponseData) {
					c.Logger = &logger
					defer logContextErrorsMiddleware(h)
					return h(c)
				}
			},
		},
	}

	routes.GET(regexp.MustCompile("^/test$"), func(c *RequestContext) ResponseData {
		return c.ErrorResponse(http.StatusInternalServerError, err1, err2)
	})

	srv := httptest.NewServer(router)
	defer srv.Close()

	res, err := http.Get(srv.URL + "/test")
	if assert.Nil(t, err) {
		defer res.Body.Close()

		t.Logf("Log contents: %s", buf.String())

		assert.Equal(t, http.StatusInternalServerError, res.StatusCode)

		assert.Contains(t, buf.String(), err1.Error())
		assert.Contains(t, buf.String(), err2.Error())
	}
}
