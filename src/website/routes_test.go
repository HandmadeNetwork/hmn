package website

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestLogContextErrors(t *testing.T) {
	err1 := errors.New("test error 1")
	err2 := errors.New("test error 2")

	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	logger.Print("sanity check")

	assert.Contains(t, buf.String(), "sanity check")

	router := &Router{}
	routes := RouteBuilder{
		Router: router,
		Middleware: func(h Handler) Handler {
			return func(c *RequestContext) (res ResponseData) {
				c.Logger = &logger
				defer LogContextErrors(c, &res)
				return h(c)
			}
		},
	}

	routes.GET("^/test$", func(c *RequestContext) ResponseData {
		return ErrorResponse(http.StatusInternalServerError, err1, err2)
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
