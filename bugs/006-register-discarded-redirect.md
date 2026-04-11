# `RegisterNewUser` discards redirect return, renders register page to logged-in users

**File:** `src/website/auth.go:178-208`
**Severity:** Low — functional, not security
**Status:** Confirmed

## The bug

```go
func RegisterNewUser(c *RequestContext) ResponseData {
    if c.CurrentUser != nil {
        c.Redirect(hmnurl.BuildUserSettings(c.CurrentUser.Username), http.StatusSeeOther)
    }

    // TODO(asaf): Do something to prevent bot registration
    // ... builds and returns the register page ...
    var res ResponseData
    res.MustWriteTemplate("auth_register.html", tmpl, c.Perf)
    return res
}
```

`c.Redirect(...)` builds a `ResponseData` value and returns it, but the caller does not `return` it. Control falls through and `RegisterNewUser` renders the register page to a user who is already logged in.

Compare with the correct form just below at `RegisterNewUserSubmit` (`auth.go:211-212`):

```go
if c.CurrentUser != nil {
    return c.RejectRequest("Can't register new user. You are already logged in")
}
```

## Impact

Not a security issue — the user can't actually register a second account from the rendered page because the POST handler rejects logged-in users correctly. The user just sees a confusing form. Cosmetic.

## Fix

Prepend `return`:

```go
if c.CurrentUser != nil {
    return c.Redirect(hmnurl.BuildUserSettings(c.CurrentUser.Username), http.StatusSeeOther)
}
```

## Related

Same anti-pattern in [bug 005](005-ticket-scanned-discarded-redirect.md). Worth a grep for `^\s*c\.Redirect\(` (statement, not return) across `src/website/` — there are likely more.
