package website

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"

	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/templates"
	"github.com/julienschmidt/httprouter"
	"github.com/rs/zerolog"
)

type HMNRouter struct {
	HttpRouter *httprouter.Router
	Wrappers   []HMNHandlerWrapper
}

func (r *HMNRouter) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	r.HttpRouter.ServeHTTP(rw, req)
}

func (r *HMNRouter) Handle(method, route string, handler HMNHandler) {
	for i := len(r.Wrappers) - 1; i >= 0; i-- {
		handler = r.Wrappers[i](handler)
	}
	r.HttpRouter.Handle(method, route, handleHmnHandler(route, handler))
}

func (r *HMNRouter) GET(route string, handler HMNHandler) {
	r.Handle(http.MethodGet, route, handler)
}

func (r *HMNRouter) POST(route string, handler HMNHandler) {
	r.Handle(http.MethodPost, route, handler)
}

// TODO: More methods

func (r *HMNRouter) ServeFiles(path string, root http.FileSystem) {
	r.HttpRouter.ServeFiles(path, root)
}

func (r *HMNRouter) WithWrappers(wrappers ...HMNHandlerWrapper) *HMNRouter {
	result := *r
	result.Wrappers = append(result.Wrappers, wrappers...)
	return &result
}

type HMNHandler func(c *RequestContext, p httprouter.Params)
type HMNHandlerWrapper func(h HMNHandler) HMNHandler

type RequestContext struct {
	StatusCode int
	Body       *bytes.Buffer
	Logger     *zerolog.Logger
	Req        *http.Request
	Errors     []error

	rw http.ResponseWriter

	currentProject *models.Project
}

func newRequestContext(rw http.ResponseWriter, req *http.Request, route string) *RequestContext {
	logger := logging.With().Str("route", route).Logger()

	return &RequestContext{
		StatusCode: http.StatusOK,
		Body:       new(bytes.Buffer),
		Logger:     &logger,
		Req:        req,

		rw: rw,
	}
}

func (c *RequestContext) Context() context.Context {
	return c.Req.Context()
}

func (c *RequestContext) URL() *url.URL {
	return c.Req.URL
}

func (c *RequestContext) Headers() http.Header {
	return c.rw.Header()
}

func (c *RequestContext) WriteTemplate(name string, data interface{}) error {
	return templates.Templates[name].Execute(c.Body, data)
}

func (c *RequestContext) AddErrors(errs ...error) {
	c.Errors = append(c.Errors, errs...)
}

func (c *RequestContext) Errored(status int, errs ...error) {
	c.StatusCode = status
	c.AddErrors(errs...)
}

func handleHmnHandler(route string, h HMNHandler) httprouter.Handle {
	return func(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
		c := newRequestContext(rw, r, route)
		defer func() {
			/*
				This panic recovery is the last resort. If you want to render
				an error page or something, make it a request wrapper.
			*/
			if recovered := recover(); recovered != nil {
				rw.WriteHeader(http.StatusInternalServerError)
				logging.LogPanicValue(c.Logger, recovered, "request panicked and was not handled")
			}
		}()

		h(c, p)

		rw.WriteHeader(c.StatusCode)
		io.Copy(rw, c.Body)
	}
}
