# Open redirect in `urlIsLocal`

**File:** `src/website/auth.go:972-979`
**Severity:** High — exploitable open redirect on Logout and login flow
**Status:** Confirmed

## The bug

```go
func urlIsLocal(url string) bool {
    urlParsed, err := neturl.Parse(url)
    if err != nil {
        return false
    }
    baseUrl := utils.Must1(neturl.Parse(config.Config.BaseUrl))
    return strings.HasSuffix(urlParsed.Host, baseUrl.Host)
}
```

`strings.HasSuffix` is a substring check, not a host-boundary check. If `config.BaseUrl` is `https://handmade.network`, then:

- `evilhandmade.network` → passes (suffix matches)
- `handmade.network.evil.com` → does not pass (not a suffix), but
- `xhandmade.network` → passes

Any attacker-controlled hostname ending in `handmade.network` bypasses the guard.

## Exploitation

`urlIsLocal` gates the `redirect` query parameter on:

- `Logout` — `src/website/auth.go:165-176` reads `?redirect=` and passes it through `urlIsLocal`.
- `safeLoginRedirectUrl` — `src/website/auth.go:134-140`, used by `LoginPage`, `LoginPageSubmit`, and `RegisterNewUser` to accept `?destination=`.

An attacker can craft a login link like:

```
https://handmade.network/login?destination=https://evilhandmade.network/phish
```

After the user logs in, the site 303s them to the phishing page. Classic credential-harvest chain: the user just typed their password and now lands on a visually identical clone.

## Fix

Compare the host exactly, or match on a leading-dot boundary for subdomains:

```go
func urlIsLocal(raw string) bool {
    u, err := neturl.Parse(raw)
    if err != nil {
        return false
    }
    base := utils.Must1(neturl.Parse(config.Config.BaseUrl))
    return u.Host == base.Host ||
        strings.HasSuffix(u.Host, "."+base.Host)
}
```

A stricter alternative is to reject any `redirect` that has a non-empty `Host` at all and only accept path-only destinations.
