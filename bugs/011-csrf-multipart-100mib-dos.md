# `csrfMiddleware` parses 100 MiB multipart per request

**File:** `src/website/middlewares.go:112-133`
**Severity:** Medium — memory-amplification DoS knob on every authenticated POST
**Status:** Confirmed

## The bug

```go
func csrfMiddleware(h Handler) Handler {
    return func(c *RequestContext) ResponseData {
        c.Req.ParseMultipartForm(100 * 1024 * 1024)
        csrfToken := c.Req.Form.Get(auth.CSRFFieldName)
        // ...
    }
}
```

Two issues:

1. The error return from `ParseMultipartForm` is ignored. A malformed multipart body passes through silently with an empty `Form`, which then fails CSRF (token missing) and logs the user out — not a security issue, but a confusing UX.
2. 100 MiB is the in-memory budget per request. `ParseMultipartForm` will buffer up to that much before spilling to disk, and every authenticated POST handler chained through `csrfMiddleware` pays that cost.

## Impact

`csrfMiddleware` wraps every POST handler that mutates state. Grep in `routes.go`:

```
hmnOnly.POST(... csrfMiddleware(SnippetEditSubmit))
hmnOnly.POST(... csrfMiddleware(TicketEditSubmit))
hmnOnly.POST(... csrfMiddleware(DiscordUnlink))
// dozens more
```

An authenticated attacker (valid session + CSRF token) can open N concurrent POSTs each carrying a 100 MiB multipart body and tie up N × 100 MiB of heap. With modest concurrency you exhaust server RAM. Even without malice, a legitimately large upload hitting one of these endpoints gets fully buffered before the handler decides whether it wanted a file at all.

Unauthenticated attackers are bounded because they fail the CSRF check and get logged out — but the parse happens *before* the CSRF check, so the memory has already been committed for the duration of the request.

## Fix

- Lower the limit drastically (e.g. 1 MiB) for CSRF parsing. Handlers that legitimately need larger uploads can call `ParseMultipartForm` themselves with a larger budget *after* CSRF succeeds.
- Better: don't parse multipart for CSRF at all. The CSRF token can be read from a header (`X-CSRF-Token`) or from the URL-encoded form on non-multipart requests. `c.Req.PostFormValue` handles the common case without buffering file parts.
- Check the error from `ParseMultipartForm` and reject with 400 on parse failure instead of falling through.

```go
if err := c.Req.ParseMultipartForm(1 * 1024 * 1024); err != nil && !errors.Is(err, http.ErrNotMultipart) {
    return c.ErrorResponse(http.StatusBadRequest, oops.New(err, "invalid form"))
}
```
