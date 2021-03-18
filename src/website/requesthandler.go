package website

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"

	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/templates"
	"github.com/julienschmidt/httprouter"
	"github.com/rs/zerolog"
)

type HMNRouter struct {
	HttpRouter *httprouter.Router
}

func (r *HMNRouter) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	r.HttpRouter.ServeHTTP(rw, req)
}

func (r *HMNRouter) Handle(method, route string, handler HMNHandler) {
	r.HttpRouter.Handle(method, route, handleHmnHandler(route, handler))
}

func (r *HMNRouter) GET(route string, handler HMNHandler) {
	r.Handle(http.MethodGet, route, handler)
}

func (r *HMNRouter) ServeFiles(path string, root http.FileSystem) {
	r.HttpRouter.ServeFiles(path, root)
}

type HMNHandler func(c *RequestContext, p httprouter.Params)

type RequestContext struct {
	StatusCode int
	Body       io.ReadWriter
	Logger     zerolog.Context

	rw  http.ResponseWriter
	req *http.Request
}

func newRequestContext(rw http.ResponseWriter, req *http.Request, route string) *RequestContext {
	return &RequestContext{
		StatusCode: http.StatusOK,
		Body:       new(bytes.Buffer),
		Logger:     logging.With().Str("route", route),

		rw:  rw,
		req: req,
	}
}

func (c *RequestContext) Context() context.Context {
	return c.req.Context()
}

func (c *RequestContext) URL() *url.URL {
	return c.req.URL
}

func (c *RequestContext) Headers() http.Header {
	return c.rw.Header()
}

func (c *RequestContext) WriteTemplate(name string, data interface{}) error {
	return templates.Templates[name].Execute(c.Body, data)
}

func handleHmnHandler(route string, h HMNHandler) httprouter.Handle {
	return func(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
		c := newRequestContext(rw, r, route)
		h(c, p)

		rw.WriteHeader(c.StatusCode)
		io.Copy(rw, c.Body)
	}
}
