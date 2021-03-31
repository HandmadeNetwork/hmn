package website

import (
	"bytes"
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

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

func (r *HMNRouter) WrapHandler(handler HMNHandler) HMNHandler {
	for i := len(r.Wrappers) - 1; i >= 0; i-- {
		handler = r.Wrappers[i](handler)
	}
	return handler
}

func (r *HMNRouter) Handle(method, route string, handler HMNHandler) {
	r.HttpRouter.Handle(method, route, handleHmnHandler(route, r.WrapHandler(handler)))
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

func MakeStdHandler(h HMNHandler, name string) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		handleHmnHandler(name, h)(rw, req, nil)
	})
}

type RequestContext struct {
	StatusCode int
	Body       *bytes.Buffer
	Logger     *zerolog.Logger
	Req        *http.Request
	Errors     []error

	rw http.ResponseWriter

	currentProject *models.Project
	currentUser    *models.User
	// currentMember *models.Member
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

func (c *RequestContext) SetCookie(cookie *http.Cookie) {
	c.rw.Header().Add("Set-Cookie", cookie.String())
}

func (c *RequestContext) GetFormValues() (url.Values, error) {
	err := c.Req.ParseForm()
	if err != nil {
		return nil, err
	}

	return c.Req.PostForm, nil
}

// The logic of this function is copy-pasted from the Go standard library.
// https://golang.org/pkg/net/http/#Redirect
func (c *RequestContext) Redirect(dest string, code int) {
	if u, err := url.Parse(dest); err == nil {
		// If url was relative, make its path absolute by
		// combining with request path.
		// The client would probably do this for us,
		// but doing it ourselves is more reliable.
		// See RFC 7231, section 7.1.2
		if u.Scheme == "" && u.Host == "" {
			oldpath := c.Req.URL.Path
			if oldpath == "" { // should not happen, but avoid a crash if it does
				oldpath = "/"
			}

			// no leading http://server
			if dest == "" || dest[0] != '/' {
				// make relative path absolute
				olddir, _ := path.Split(oldpath)
				dest = olddir + dest
			}

			var query string
			if i := strings.Index(dest, "?"); i != -1 {
				dest, query = dest[:i], dest[i:]
			}

			// clean up but preserve trailing slash
			trailing := strings.HasSuffix(dest, "/")
			dest = path.Clean(dest)
			if trailing && !strings.HasSuffix(dest, "/") {
				dest += "/"
			}
			dest += query
		}
	}

	h := c.Headers()

	// RFC 7231 notes that a short HTML body is usually included in
	// the response because older user agents may not understand 301/307.
	// Do it only if the request didn't already have a Content-Type header.
	_, hadCT := h["Content-Type"]

	// Escape stuff
	destUrl, _ := url.Parse(dest)
	dest = destUrl.String()

	h.Set("Location", dest)
	if !hadCT && (c.Req.Method == "GET" || c.Req.Method == "HEAD") {
		h.Set("Content-Type", "text/html; charset=utf-8")
	}
	c.StatusCode = code

	// Shouldn't send the body for POST or HEAD; that leaves GET.
	if !hadCT && c.Req.Method == "GET" {
		body := "<a href=\"" + html.EscapeString(dest) + "\">" + http.StatusText(code) + "</a>.\n"
		fmt.Fprintln(c.Body, body)
	}
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
