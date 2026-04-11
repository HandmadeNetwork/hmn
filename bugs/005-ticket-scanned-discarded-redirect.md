# `TicketScanned` discards redirect return, falls through non-staff guard

**File:** `src/website/tickets.go:738-745`
**Severity:** Medium today (stub handler), High when real logic lands
**Status:** Confirmed

## The bug

```go
func TicketScanned(c *RequestContext) ResponseData {
    if !c.CurrentUser.IsStaff {
        c.Redirect("https://www.youtube.com/watch?v=dQw4w9WgXcQ", http.StatusSeeOther)
    }

    // TODO(ben): Actually build ticket-scanning logic closer to the time of the event.
    return ResponseData{}
}
```

Two bugs stacked:

1. The staff guard constructs a redirect but throws it away. `ResponseData` is a value, not a pointer — `c.Redirect(...)` returns it, and the caller must `return` it. Currently control falls through to the `return ResponseData{}` below, which emits a blank 200 OK for non-staff users. This is an auth-bypass pattern.
2. `c.CurrentUser.IsStaff` dereferences `c.CurrentUser` without a nil check. Today the route is registered as `needsAuth(TicketScanned)` at `src/website/routes.go:146`, so `CurrentUser` is guaranteed non-nil by the middleware — this landmine is dormant. If the handler is ever wired without `needsAuth` (e.g. to allow QR-code scans from a logged-out kiosk), it becomes a hostile-input panic.

## Why it matters

The body is currently `return ResponseData{}` — no actual scan logic, so the fall-through only leaks the fact that the endpoint exists. The moment the TODO is implemented, the non-staff branch will run it for anyone who loads `/tickets/:id/scanned`. Fixing the guard now costs one keyword and one `return`, and removes the footgun permanently.

## Fix

```go
func TicketScanned(c *RequestContext) ResponseData {
    if c.CurrentUser == nil || !c.CurrentUser.IsStaff {
        return FourOhFour(c)
    }

    // TODO(ben): Actually build ticket-scanning logic closer to the time of the event.
    return ResponseData{}
}
```

Consider wrapping with `adminsOnly(TicketScanned)` in `routes.go` instead of hand-rolling the check — that middleware already exists at `src/website/middlewares.go:82-90` and handles both the nil check and the staff check.

## Related

Grep for other discarded `c.Redirect` returns — the same anti-pattern appears at least at `src/website/auth.go:180` (see [bug 006](006-register-discarded-redirect.md)). Worth a full sweep.
