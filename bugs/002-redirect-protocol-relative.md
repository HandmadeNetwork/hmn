# ~~`Redirect` helper passes protocol-relative URLs unchanged~~

**File:** `src/website/requesthandling.go:311-368`
**Status:** **RETRACTED** — the helper is a faithful copy of `net/http.Redirect` and is not itself the bug. The real open redirect is in call sites that pass user-controlled input to `c.Redirect` without validation. See the "Actual finding" section below.

## Why the original claim is wrong

hmn's `Redirect` was copied from the Go standard library's `http.Redirect` in `net/http/server.go`. Comparing the two side by side:

```go
// net/http/server.go, Go 1.22
func Redirect(w ResponseWriter, r *Request, url string, code int) {
    if u, err := urlpkg.Parse(url); err == nil {
        if u.Scheme == "" && u.Host == "" {
            oldpath := r.URL.EscapedPath()
            // ... path-join logic ...
        }
    }
    // ... set Location, write body ...
}
```

The `u.Scheme == "" && u.Host == ""` check is not a safety guard. It's the RFC 7231 § 7.1.2 relative-reference resolver: "if this looks like a bare relative path, resolve it against the request path before emitting `Location:`". A protocol-relative URL like `//evil.com` is *not* a bare relative path — it already has a host — so this branch is correctly skipped and the URL is emitted verbatim. That is exactly what RFC 7231 says should happen.

`net/http.Redirect` is documented as a helper for emitting a redirect. It is *not* documented as an open-redirect defense. The stdlib expectation is that callers who accept user-controlled destinations validate those destinations themselves before calling `Redirect`. Every Go web app that uses `http.Redirect` has the same "bug" and none of them are bugs.

Original claim in this file — that the helper should reject `//evil.com` — would make hmn's helper behave *differently* from stdlib, which is worse: it would lull callers into thinking the helper is a safety gate and encourage them to stop validating at call sites. The fix belongs at the call sites.

## Actual finding

The real open redirect lives at `src/website/auth.go:151-155`:

```go
func LoginWithDiscord(c *RequestContext) ResponseData {
    destinationUrl := c.URL().Query().Get("redirect")
    if c.CurrentUser != nil {
        return c.Redirect(destinationUrl, http.StatusSeeOther)
    }
    // ... non-logged-in branch continues ...
}
```

`destinationUrl` is read straight from the query string and passed to `c.Redirect` without going through `urlIsLocal` / `safeLoginRedirectUrl`. Any value works: `https://evil.com`, `//evil.com`, anything. Condition: the user must already be signed in when they visit `/login/discord?redirect=...`. Exploitation chain:

1. Attacker sends victim a link to `https://handmade.network/login/discord?redirect=https://evil.com`.
2. Victim is already signed in (common — most people stay logged in).
3. Handler sees `c.CurrentUser != nil`, takes the early-return branch, and 303s the browser to `https://evil.com`.

Phishing payload: a login-themed landing page on `evil.com` that asks the victim to "re-confirm" their password. Because the victim just clicked a handmade.network link and the redirect chain is instantaneous, the site boundary is easy to miss.

The non-logged-in branch is fine — it stashes `destinationUrl` in the `pending_login` row and consumes it later via the Discord OAuth callback, which *should* run the result through `urlIsLocal` before honoring it. Worth verifying that the callback side does this.

### Fix

Gate the destination on `urlIsLocal` (after fixing [bug 001](001-urlIsLocal-open-redirect.md) — `urlIsLocal` itself is currently broken) or reuse `safeLoginRedirectUrl`:

```go
func LoginWithDiscord(c *RequestContext) ResponseData {
    destinationUrl := safeLoginRedirectUrl(c.URL().Query().Get("redirect"))
    if c.CurrentUser != nil {
        return c.Redirect(destinationUrl, http.StatusSeeOther)
    }
    // ...
}
```

Then grep the rest of `src/website/` for `c.Redirect(` to find any other call site that forwards query/form input without validation. A quick scan of `auth.go` shows the rest go through `safeLoginRedirectUrl` or build URLs from `hmnurl.*`, which is safe.
