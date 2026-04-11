# `FullUrl()` trusts client `X-Forwarded-Proto` unconditionally

**File:** `src/website/requesthandling.go:233-252`
**Severity:** Medium — lets an attacker forge the scheme of site-generated URLs
**Status:** Confirmed

## The bug

```go
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
```

`var scheme string; if scheme == "" { ... }` is dead-code-shaped — the first guard always fires, which strongly suggests the intended shape was a config-gated trusted-proxy check that got lost. As written, `X-Forwarded-Proto` from *any* client (not just Cloudflare) is trusted.

## Impact

`FullUrl()` is used to build absolute URLs the server then emits back:

- `needsAuthWithNotice` — `src/website/middlewares.go:75` — `BuildLoginPage(c.FullUrl(), ...)`, i.e. the `redirect=` parameter embedded in the login URL.
- `logContextErrors` — `src/website/middlewares.go:151` — logged, lower risk.

An attacker who can make a victim request `/some-auth-page` with a forged `X-Forwarded-Proto: javascript` (via reflected-input or open redirect, depending on deploy) influences the scheme of the login-redirect URL stamped into the response. The browser won't execute `javascript://...` from a `Location:` header directly, but this becomes a primitive for scheme confusion on any page that pastes `FullUrl()` into HTML.

More mundanely, this corrupts email links the site generates off `FullUrl()` to `http://` on an `https://` site, which can trigger security software / SEG rewrites.

## Fix

Gate header trust behind a config flag naming the trusted proxy, identical to how real reverse-proxy packages handle this. Minimum patch:

```go
func (c *RequestContext) FullUrl() string {
    scheme := "http://"
    if c.Req.TLS != nil {
        scheme = "https://"
    }
    if config.Config.TrustProxyHeaders {
        if proto := c.Req.Header.Get("X-Forwarded-Proto"); proto != "" {
            scheme = proto + "://"
        }
    }
    return scheme + c.Req.Host + c.Req.URL.String()
}
```

Same fix applies verbatim to `GetIP()` — see [bug 004](004-getip-header-spoofing.md).
