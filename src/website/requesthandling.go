package website

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"html/template"
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/perf"
	"git.handmade.network/hmn/hmn/src/templates"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

type Router struct {
	Routes []Route
}

type Route struct {
	Method  string
	Regexes []*regexp.Regexp
	Handler Handler
}

func (r *Route) String() string {
	var routeStrings []string
	for _, regex := range r.Regexes {
		routeStrings = append(routeStrings, regex.String())
	}
	return fmt.Sprintf("%s %v", r.Method, routeStrings)
}

type RouteBuilder struct {
	Router      *Router
	Prefixes    []*regexp.Regexp
	Middlewares []Middleware
}

type Handler func(c *RequestContext) ResponseData
type Middleware func(h Handler) Handler

func applyMiddlewares(h Handler, ms []Middleware) Handler {
	result := h
	for i := len(ms) - 1; i >= 0; i-- {
		result = ms[i](result)
	}
	return result
}

func (rb *RouteBuilder) Handle(methods []string, regex *regexp.Regexp, h Handler) {
	// Ensure that this regex matches the start of the string
	regexStr := regex.String()
	if len(regexStr) == 0 || regexStr[0] != '^' {
		panic("All routing regexes must begin with '^'")
	}

	h = applyMiddlewares(h, rb.Middlewares)
	for _, method := range methods {
		rb.Router.Routes = append(rb.Router.Routes, Route{
			Method:  method,
			Regexes: append(rb.Prefixes, regex),
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

func (rb *RouteBuilder) WithMiddleware(ms ...Middleware) RouteBuilder {
	newRb := *rb
	newRb.Middlewares = append(rb.Middlewares, ms...)

	return newRb
}

func (rb *RouteBuilder) Group(regex *regexp.Regexp, ms ...Middleware) RouteBuilder {
	newRb := *rb
	newRb.Prefixes = append(newRb.Prefixes, regex)
	newRb.Middlewares = append(rb.Middlewares, ms...)

	return newRb
}

func (r *Router) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	method := req.Method
	if method == http.MethodHead {
		method = http.MethodGet // HEADs map to GETs for the purposes of routing
	}

nextroute:
	for _, route := range r.Routes {
		if route.Method != "" && method != route.Method {
			continue
		}

		currentPath := strings.TrimSuffix(req.URL.Path, "/")
		if currentPath == "" {
			currentPath = "/"
		}

		var params map[string]string
		for _, regex := range route.Regexes {
			match := regex.FindStringSubmatch(currentPath)
			if len(match) == 0 {
				continue nextroute
			}

			if params == nil {
				params = map[string]string{}
			}
			subexpNames := regex.SubexpNames()
			for i, paramValue := range match {
				paramName := subexpNames[i]
				if paramName == "" {
					continue
				}
				if _, alreadyExists := params[paramName]; alreadyExists {
					logging.Warn().
						Str("route", route.String()).
						Str("paramName", paramName).
						Msg("duplicate names for path parameters; last one wins")
				}
				params[paramName] = paramValue
			}

			// Make sure that we never consume trailing slashes even if the route regex matches them
			toConsume := strings.TrimSuffix(match[0], "/")
			currentPath = currentPath[len(toConsume):]
			if currentPath == "" {
				currentPath = "/"
			}
		}

		c := &RequestContext{
			Route:      route.String(),
			Logger:     logging.GlobalLogger(),
			Req:        req,
			Res:        rw,
			PathParams: params,

			ctx: req.Context(),
		}
		c.PathParams = params

		doRequest(rw, c, route.Handler)

		return
	}

	panic(fmt.Sprintf("Path '%s' did not match any routes! Make sure to register a wildcard route to act as a 404.", req.URL))
}

type RequestContext struct {
	Route      string
	Logger     *zerolog.Logger
	Req        *http.Request
	PathParams map[string]string

	// NOTE(asaf): This is the http package's internal response object. Not just a ResponseWriter.
	//             We sometimes need the original response object so that some functions of the http package can set connection-management flags on it.
	Res http.ResponseWriter

	Conn                  *pgxpool.Pool
	CurrentProject        *models.Project
	CurrentProjectLogoUrl string
	CurrentUser           *models.User
	CurrentSession        *models.Session
	UrlContext            *hmnurl.UrlContext

	CurrentUserCanEditCurrentProject bool

	Perf          *perf.RequestPerf
	PerfCollector *perf.PerfCollector

	ctx context.Context
}

// Our RequestContext is a context.Context

var _ context.Context = &RequestContext{}

func (c *RequestContext) Deadline() (time.Time, bool) {
	return c.ctx.Deadline()
}

func (c *RequestContext) Done() <-chan struct{} {
	return c.ctx.Done()
}

func (c *RequestContext) Err() error {
	return c.ctx.Err()
}

func (c *RequestContext) Value(key any) any {
	switch key {
	case perf.PerfContextKey:
		return c.Perf
	default:
		return c.ctx.Value(key)
	}
}

// Plus it does many other things specific to us

func (c *RequestContext) URL() *url.URL {
	return c.Req.URL
}

func (c *RequestContext) FullUrl() string {
	var scheme string

	if scheme == "" {
		proto, hasProto := c.Req.Header["X-Forwarded-Proto"]
		if hasProto {
			scheme = fmt.Sprintf("%s://", proto[0])
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

func (c *RequestContext) GetIP() *netip.Prefix {
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
		res, err := netip.ParsePrefix(fmt.Sprintf("%s/32", ipString))
		if err == nil {
			return &res
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
	destUrl, err := url.Parse(dest)
	if err != nil {
		c.Logger.Warn().Err(err).Str("dest", dest).Msg("Failed to parse redirect URI")
		return c.Redirect(hmnurl.BuildHomepage(), http.StatusSeeOther)
	}
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
	defer func() {
		if r := recover(); r != nil {
			logContextErrors(c, errs...)
			panic(r)
		}
	}()

	res := ResponseData{
		StatusCode: status,
		Errors:     errs,
	}
	res.MustWriteTemplate("error.html", getBaseData(c, ""), c.Perf)
	return res
}

func (c *RequestContext) RejectRequest(reason string) ResponseData {
	type RejectData struct {
		templates.BaseData
		RejectReason string
	}

	var res ResponseData
	err := res.WriteTemplate("reject.html", RejectData{
		BaseData:     getBaseData(c, "Rejected"),
		RejectReason: reason,
	}, c.Perf)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "Failed to render reject template"))
	}
	return res
}

type ResponseData struct {
	StatusCode    int
	Body          *bytes.Buffer
	Errors        []error
	FutureNotices []templates.Notice

	header http.Header

	hijacked bool
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
		b := rp.StartBlock("TEMPLATE", name)
		defer b.End()
	}
	return templates.GetTemplate(name).Execute(rd, data)
}

func (rd *ResponseData) MustWriteTemplate(name string, data interface{}, rp *perf.RequestPerf) {
	err := rd.WriteTemplate(name, data, rp)
	if err != nil {
		panic(err)
	}
}

func (rd *ResponseData) WriteJson(data any, rp *perf.RequestPerf) {
	dataJson, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	rd.Header().Set("Content-Type", "application/json")
	rd.Write(dataJson)
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

	// Run the chosen handler
	res := h(c)

	if res.hijacked {
		// NOTE(asaf): In case we forward the request/response to another handler
		//             (like esbuild).
		return
	}

	if res.StatusCode == 0 {
		res.StatusCode = http.StatusOK
	}

	// Set Content-Type and Content-Length if necessary. This behavior would in
	// some cases be handled by http.ResponseWriter.Write, but we extract it so
	// that HEAD requests always return both headers.

	var preamble []byte // Any bytes we read to determine Content-Type
	if res.Body != nil {
		bodyLen := res.Body.Len()

		if res.Header().Get("Content-Type") == "" {
			preamble = res.Body.Next(512)
			rw.Header().Set("Content-Type", http.DetectContentType(preamble))
		}
		if res.Header().Get("Content-Length") == "" {
			rw.Header().Set("Content-Length", strconv.Itoa(bodyLen))
		}
	}

	// Ensure we send no body for HEAD requests
	if c.Req.Method == http.MethodHead {
		res.Body = nil
	}

	// Send remaining response headers
	for name, vals := range res.Header() {
		for _, val := range vals {
			rw.Header().Add(name, val)
		}
	}
	rw.WriteHeader(res.StatusCode)

	// Send response body
	if res.Body != nil {
		// Write preamble, if any
		_, err := rw.Write(preamble)
		if err != nil {
			if errors.Is(err, syscall.EPIPE) {
				// NOTE(asaf): Can be triggered when other side hangs up
				logging.Debug().Msg("Broken pipe")
			} else {
				logging.Error().Err(err).Msg("Failed to write response preamble")
			}
		}

		// Write remainder of body
		_, err = io.Copy(rw, res.Body)
		if err != nil {
			if errors.Is(err, syscall.EPIPE) {
				// NOTE(asaf): Can be triggered when other side hangs up
				logging.Debug().Msg("Broken pipe")
			} else {
				logging.Error().Err(err).Msg("copied res.Body")
			}
		}
	}
}
