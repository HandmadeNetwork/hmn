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
	"regexp"
	"strings"

	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/perf"
	"git.handmade.network/hmn/hmn/src/templates"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/rs/zerolog"
)

type Router struct {
	Routes []Route
}

type Route struct {
	Method  string
	Regex   *regexp.Regexp
	Handler Handler
}

type RouteBuilder struct {
	Router     *Router
	Middleware Middleware
}

type Handler func(c *RequestContext) ResponseData

type Middleware func(h Handler) Handler

func (rb *RouteBuilder) Handle(methods []string, regex *regexp.Regexp, h Handler) {
	h = rb.Middleware(h)
	for _, method := range methods {
		rb.Router.Routes = append(rb.Router.Routes, Route{
			Method:  method,
			Regex:   regex,
			Handler: h,
		})
	}
}

func (rb *RouteBuilder) AnyMethod(regex *regexp.Regexp, h Handler) {
	rb.Handle([]string{""}, regex, h)
}

func (rb *RouteBuilder) GET(regex *regexp.Regexp, h Handler) {
	rb.Handle([]string{http.MethodGet}, regex, h)
}

func (rb *RouteBuilder) POST(regex *regexp.Regexp, h Handler) {
	rb.Handle([]string{http.MethodPost}, regex, h)
}

func (r *Router) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	for _, route := range r.Routes {
		if route.Method != "" && req.Method != route.Method {
			continue
		}

		match := route.Regex.FindStringSubmatch(path)
		if match == nil {
			continue
		}

		c := &RequestContext{
			Route:  route.Regex.String(),
			Logger: logging.GlobalLogger(),
			Req:    req,
		}

		if len(match) > 0 {
			params := map[string]string{}
			subexpNames := route.Regex.SubexpNames()
			for i, paramValue := range match {
				paramName := subexpNames[i]
				if paramName == "" {
					continue
				}
				params[paramName] = paramValue
			}
			c.PathParams = params
		}

		doRequest(rw, c, route.Handler)

		return
	}

	// TODO(asaf): Replace this with a 404 handler? Isn't this going to crash the server?
	// We're doing panic recovery in doRequest, but not here. I don't think we should have this line in production.
	panic(fmt.Sprintf("Path '%s' did not match any routes! Make sure to register a wildcard route to act as a 404.", path))
}

type RequestContext struct {
	Route      string
	Logger     *zerolog.Logger
	Req        *http.Request
	PathParams map[string]string

	Conn           *pgxpool.Pool
	CurrentProject *models.Project
	CurrentUser    *models.User
	CurrentSession *models.Session
	Theme          string

	Perf *perf.RequestPerf
}

func (c *RequestContext) Context() context.Context {
	return c.Req.Context()
}

func (c *RequestContext) URL() *url.URL {
	return c.Req.URL
}

func (c *RequestContext) FullUrl() string {
	var scheme string // TODO(asaf): BEFORE RELEASE!! Fetch scheme from X-Forwarded-* headers or Forwarded header
	if c.Req.TLS != nil {
		scheme = "https://"
	} else {
		scheme = "http://"
	}
	return scheme + c.Req.Host + c.Req.URL.String()
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

	res.Header().Set("Location", dest)
	if c.Req.Method == "GET" || c.Req.Method == "HEAD" {
		res.Header().Set("Content-Type", "text/html; charset=utf-8")
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

var _ http.ResponseWriter = &ResponseData{}

func (rd *ResponseData) Header() http.Header {
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

func (rd *ResponseData) WriteHeader(status int) {
	rd.StatusCode = status
}

func (rd *ResponseData) SetCookie(cookie *http.Cookie) {
	rd.Header().Add("Set-Cookie", cookie.String())
}

func (rd *ResponseData) WriteTemplate(name string, data interface{}, rp *perf.RequestPerf) error {
	if rp != nil {
		rp.StartBlock("TEMPLATE", name)
		defer rp.EndBlock()
	}
	template, hasTemplate := templates.Templates[name]
	if !hasTemplate {
		panic(oops.New(nil, "Template not found: %s", name))
	}
	return template.Execute(rd, data)
}

func ErrorResponse(status int, errs ...error) ResponseData {
	return ResponseData{
		StatusCode: status,
		Errors:     errs,
	}
}

func doRequest(rw http.ResponseWriter, c *RequestContext, h Handler) {
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

	for name, vals := range res.Header() {
		for _, val := range vals {
			rw.Header().Add(name, val)
		}
	}
	rw.WriteHeader(res.StatusCode)

	if res.Body != nil {
		io.Copy(rw, res.Body)
	}
}
