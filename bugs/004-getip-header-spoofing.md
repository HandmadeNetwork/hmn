# `GetIP()` trusts client IP-forwarding headers unconditionally

**File:** `src/website/requesthandling.go:257-298`
**Severity:** Medium — IP spoofing bypasses rate limits, audit logs, and any per-IP allow/deny logic
**Status:** Confirmed

## The bug

```go
func (c *RequestContext) GetIP() *netip.Prefix {
    ipString := ""

    if ipString == "" {
        cf, hasCf := c.Req.Header["Cf-Connecting-Ip"]
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
        // ... parse host:port ...
    }
    // ...
}
```

Identical pattern to [bug 003](003-fullurl-xff-proto-spoofing.md) — both `Cf-Connecting-Ip` and `X-Forwarded-For` are read from any client with no trusted-proxy check. If production sits behind Cloudflare, the real client IP arrives in `Cf-Connecting-Ip`, which is fine *provided* the origin refuses direct connections. If the origin is ever reachable without going through Cloudflare (common slip: forgot firewall rules, staging host, direct-by-IP), any attacker sets `Cf-Connecting-Ip: 1.2.3.4` and the site accepts it.

`X-Forwarded-For` is worse: it's comma-delimited (`client, proxy1, proxy2`), but the code takes `forwarded[0]` which is the first *header instance*, not the first IP in a comma-delimited list. For a single-header request the value includes all proxies as one string, and `netip.ParsePrefix` will reject it.

## Second bug in the same function

```go
if ipString != "" {
    res, err := netip.ParsePrefix(fmt.Sprintf("%s/32", ipString))
    if err == nil {
        return &res
    }
}
```

`/32` is a v4 prefix. For an IPv6 address this builds `2001:db8::1/32` which parses as a /32 on a 128-bit address — a nonsense prefix covering a huge swath of v6 space. Any caller that uses the `Prefix` for containment checks will incorrectly match millions of unrelated hosts.

## Impact

Anywhere `GetIP()` is used for rate limiting, audit trails, or per-IP blocking is compromised. The registration log at `src/website/auth.go:222-227` records `ip` and `X-Forwarded-For` — useful forensically but also attacker-controlled.

## Fix

1. Gate header parsing behind a `TrustProxyHeaders` config flag.
2. For `X-Forwarded-For`, split on commas and take the leftmost *trusted* entry (i.e. pop trusted-proxy IPs from the right).
3. For `netip.ParsePrefix`, pick the prefix length from the address family: `/32` for v4, `/128` for v6. Or skip the prefix wrapper and return `netip.Addr` directly.
