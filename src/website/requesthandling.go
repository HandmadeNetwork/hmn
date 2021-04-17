package website

import (
	"bytes"
	"context"
	"html"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/templates"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/julienschmidt/httprouter"
	"github.com/rs/zerolog"
)

// The typical handler. Handles a request and returns data about the response.
type HMNHandler func(c *RequestContext) ResponseData

// A special handler that runs before the primary handler. Intended to set
// information on the context for later handlers, or to give the request a
// means to bail out early if preconditions are not met (like auth). If `ok`
// is false, the request will immediately bail out, no further handlers will
// be run, and it will respond with the provided response data.
//
// The response data from this function will still be fed through any after
// handlers, to ensure that errors will get logged and whatnot.
type HMNBeforeHandler func(c *RequestContext) (ok bool, res ResponseData)

// A special handler that runs after the primary handler and can modify the
// response information. Intended for error logging, error pages,
// cleanup, etc.
type HMNAfterHandler func(c *RequestContext, res ResponseData) ResponseData

func (h HMNHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	c := NewRequestContext(rw, req, nil)
	doRequest(rw, c, h)
}

type RequestContext struct {
	Logger     *zerolog.Logger
	Req        *http.Request
	PathParams httprouter.Params

	Conn           *pgxpool.Pool
	CurrentProject *models.Project
	CurrentUser    *models.User
}

func NewRequestContext(rw http.ResponseWriter, req *http.Request, pathParams httprouter.Params) *RequestContext {
	return &RequestContext{
		Logger:     logging.GlobalLogger(),
		Req:        req,
		PathParams: pathParams,
	}
}

func (c *RequestContext) Context() context.Context {
	return c.Req.Context()
}

func (c *RequestContext) URL() *url.URL {
	return c.Req.URL
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
func (c *RequestContext) Redirect(dest string, code int) ResponseData {
	var res ResponseData

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

	// Escape stuff
	destUrl, _ := url.Parse(dest)
	dest = destUrl.String()

	res.Headers().Set("Location", dest)
	if c.Req.Method == "GET" || c.Req.Method == "HEAD" {
		res.Headers().Set("Content-Type", "text/html; charset=utf-8")
	}
	res.StatusCode = code

	// Shouldn't send the body for POST or HEAD; that leaves GET.
	if c.Req.Method == "GET" {
		res.Write([]byte("<a href=\"" + html.EscapeString(dest) + "\">" + http.StatusText(code) + "</a>.\n"))
	}

	return res
}

type ResponseData struct {
	StatusCode int
	Body       *bytes.Buffer
	Errors     []error

	header http.Header
}

func (rd *ResponseData) Headers() http.Header {
	if rd.header == nil {
		rd.header = make(http.Header)
	}

	return rd.header
}

func (rd *ResponseData) Write(p []byte) (n int, err error) {
	if rd.Body == nil {
		rd.Body = new(bytes.Buffer)
	}

	return rd.Body.Write(p)
}

func (rd *ResponseData) SetCookie(cookie *http.Cookie) {
	rd.Headers().Add("Set-Cookie", cookie.String())
}

func (rd *ResponseData) WriteTemplate(name string, data interface{}) error {
	return templates.Templates[name].Execute(rd, data)
}

type RouteBuilder struct {
	Router         *httprouter.Router
	BeforeHandlers []HMNBeforeHandler
	AfterHandlers  []HMNAfterHandler
}

func (b RouteBuilder) ChainHandlers(h HMNHandler) HMNHandler {
	return func(c *RequestContext) ResponseData {
		beforeOk := true
		var res ResponseData
		for _, before := range b.BeforeHandlers {
			if ok, errorRes := before(c); !ok {
				beforeOk = false
				res = errorRes
			}
		}

		if beforeOk {
			res = h(c)
		}

		for _, after := range b.AfterHandlers {
			res = after(c, res)
		}
		return res
	}
}

func (b *RouteBuilder) Handle(method, route string, handler HMNHandler) {
	h := b.ChainHandlers(handler)
	b.Router.Handle(method, route, func(rw http.ResponseWriter, req *http.Request, p httprouter.Params) {
		c := NewRequestContext(rw, req, p)
		doRequest(rw, c, h)
	})
}

func (b *RouteBuilder) GET(route string, handler HMNHandler) {
	b.Handle(http.MethodGet, route, handler)
}

func (b *RouteBuilder) POST(route string, handler HMNHandler) {
	b.Handle(http.MethodPost, route, handler)
}

// TODO: More methods

func (b *RouteBuilder) ServeFiles(path string, root http.FileSystem) {
	b.Router.ServeFiles(path, root)
}

func ErrorResponse(status int, errs ...error) ResponseData {
	return ResponseData{
		StatusCode: status,
		Errors:     errs,
	}
}

func doRequest(rw http.ResponseWriter, c *RequestContext, h HMNHandler) {
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

	res := h(c)

	if res.StatusCode == 0 {
		res.StatusCode = http.StatusOK
	}

	for name, vals := range res.Headers() {
		for _, val := range vals {
			rw.Header().Add(name, val)
		}
	}
	rw.WriteHeader(res.StatusCode)
	io.Copy(rw, res.Body)
}
