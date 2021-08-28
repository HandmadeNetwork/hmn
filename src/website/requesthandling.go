package website

import (
	"bytes"
	"context"
	"fmt"
	"html"
	"html/template"
	"io"
	"net"
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

		path = strings.TrimSuffix(path, "/")
		if path == "" {
			path = "/"
		}

		match := route.Regex.FindStringSubmatch(path)
		if match == nil {
			continue
		}

		c := &RequestContext{
			Route:  route.Regex.String(),
			Logger: logging.GlobalLogger(),
			Req:    req,
			Res:    rw,
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

	panic(fmt.Sprintf("Path '%s' did not match any routes! Make sure to register a wildcard route to act as a 404.", path))
}

type RequestContext struct {
	Route      string
	Logger     *zerolog.Logger
	Req        *http.Request
	PathParams map[string]string

	// NOTE(asaf): This is the http package's internal response object. Not just a ResponseWriter.
	//             We sometimes need the original response object so that some functions of the http package can set connection-management flags on it.
	Res http.ResponseWriter

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
	var scheme string

	if scheme == "" {
		proto, hasProto := c.Req.Header["X-Forwarded-Proto"]
		if hasProto {
			scheme = fmt.Sprintf("%s://", proto)
		}
	}

	if scheme == "" {
		if c.Req.TLS != nil {
			scheme = "https://"
		} else {
			scheme = "http://"
		}
	}
	return scheme + c.Req.Host + c.Req.URL.String()
}

// NOTE(asaf): Assumes port is present (it should be for RemoteAddr according to the docs)
var ipRegex = regexp.MustCompile(`^(\[(?P<addrv6>[^\]]+)\]:\d+)|((?P<addrv4>[^:]+):\d+)$`)

func (c *RequestContext) GetIP() *net.IPNet {
	ipString := ""

	if ipString == "" {
		cf, hasCf := c.Req.Header["CF-Connecting-IP"]
		if hasCf {
			ipString = cf[0]
		}
	}

	if ipString == "" {
		forwarded, hasForwarded := c.Req.Header["X-Forwarded-For"]
		if hasForwarded {
			ipString = forwarded[0]
		}
	}

	if ipString == "" {
		ipString = c.Req.RemoteAddr
		if ipString != "" {
			matches := ipRegex.FindStringSubmatch(ipString)
			if matches != nil {
				v4 := matches[ipRegex.SubexpIndex("addrv4")]
				v6 := matches[ipRegex.SubexpIndex("addrv6")]
				if v4 != "" {
					ipString = v4
				} else {
					ipString = v6
				}
			}
		}
	}

	if ipString != "" {
		_, res, err := net.ParseCIDR(fmt.Sprintf("%s/32", ipString))
		if err == nil {
			return res
		}
	}

	return nil
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

func (c *RequestContext) ErrorResponse(status int, errs ...error) ResponseData {
	res := ResponseData{
		StatusCode: status,
		Errors:     errs,
	}
	res.MustWriteTemplate("error.html", getBaseData(c), c.Perf)
	return res
}

type ResponseData struct {
	StatusCode    int
	Body          *bytes.Buffer
	Errors        []error
	FutureNotices []templates.Notice

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

func (rd *ResponseData) AddFutureNotice(class string, content string) {
	rd.FutureNotices = append(rd.FutureNotices, templates.Notice{Class: class, Content: template.HTML(content)})
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

func (rd *ResponseData) MustWriteTemplate(name string, data interface{}, rp *perf.RequestPerf) {
	err := rd.WriteTemplate(name, data, rp)
	if err != nil {
		panic(err)
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
			rw.Write([]byte("There was a problem handling your request.\nPlease notify an admin at team@handmade.network"))
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
