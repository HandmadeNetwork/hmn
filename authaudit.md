# Auth Security Audit

Date: 2026-04-11
Scope: `src/auth/`, `src/website/auth.go`, `src/website/discord.go`, `src/website/middlewares.go`, `src/models/`, `src/config/`

Severity rules:
- Panic / crash → LOW unless it corrupts state or bypasses auth (process recovers).
- Bug that needs attacker to first tamper with DB or internal state → one tier lower than identical externally-reachable bug.
- Network timing oracles on high-entropy (>128-bit) tokens → INFO unless measurable.

---

## Confirmed Bugs

### BUG-1 (LOW): Nil pointer panic in Discord OAuth callback

**File:** `src/website/discord.go:73`

```go
if c.CurrentUser == nil {
    // ...
    if err == db.NotFound {
        c.Logger.Warn().Str("userId", c.CurrentUser.Username).Msg(...)  // nil deref
```

Trigger: unauthenticated Discord callback with stale/forged `state` param → `db.NotFound` → deref nil pointer → panic → `panicCatcherMiddleware` catches → HTTP 500 → next request fine.

Impact: crash-per-request. No auth bypass, no data leak, no persistent state damage. Attacker gets ugly logs + DoS-per-request (not cumulative). LOW because self-healing.

**Fix:** replace `c.CurrentUser.Username` with literal `"unauthenticated"`. Same pattern already used correctly at `middlewares.go:120-122`.

---

### BUG-2 (LOW): Inactive user leaks password correctness

**File:** `src/website/auth.go:110-121`

```go
success, err := tryLogin(c, user, password)
// ...
if !success {
    return showLoginWithFailure(...)                         // "Incorrect username or password"
}
if user.Status == models.UserStatusInactive {
    return c.RejectRequest("You must validate your email")   // DIFFERENT
}
```

`tryLogin` only early-exits on `UserStatusBanned`. Inactive + correct password returns `(true, nil)`. `LoginAction` then shows distinct error.

Oracle: attacker with candidate password can distinguish "valid pw + account not yet confirmed" from "wrong pw". Pre-condition is already-known password, so oracle gives almost no new info. Exploit window = registration-to-confirmation only. LOW.

**Fix:** move `UserStatusInactive` check into `tryLogin` and return generic failure, OR add it to the banned check so both map to generic error.

---

### BUG-3 (INFO): CSRF token comparison not constant-time

**File:** `src/website/middlewares.go:118`

```go
if c.CurrentSession == nil || csrfToken != c.CurrentSession.CSRFToken {
```

Plain `!=`. OWASP best practice says constant-time. Real-world exploit over network against 180-bit token: impractical — network jitter + session lookup time dominate byte-compare time by orders of magnitude. Same issue in OTT compare (`validateUsernameAndToken`).

**Fix:** one-liner, do it anyway:
```go
subtle.ConstantTimeCompare([]byte(csrfToken), []byte(c.CurrentSession.CSRFToken)) != 1
```

---

### BUG-4 (INFO): DeleteSessionCookie missing `Path: "/"`

**File:** `src/auth/session.go:133-137`

```go
var DeleteSessionCookie = &http.Cookie{
    Name:   SessionCookieName,
    Domain: config.Config.Auth.CookieDomain,
    MaxAge: -1,
    // Path unset
}
```

RFC 6265: browser default-path = directory of request URL. Logout served from `/logout` → default-path `/` → matches original cookie's `Path=/` in practice. Cosmetic inconsistency. Fix anyway for clarity.

---

## Security Risks

### RISK-1 (MEDIUM): No brute-force protection on login

`securityTimerMiddleware(100ms)` is the only floor. Argon2id ~50ms already. Added cost ~50ms per guess. No account lockout, no CAPTCHA, no backoff. Distributed attack unimpeded.

**Fix:** per-username failed-attempt counter in DB. Lock / CAPTCHA after N fails.

---

### RISK-2 (LOW): No max password length

**File:** `src/auth/auth.go:155-179`

`HashPassword` passes input directly to `argon2.IDKey`. No cap. Measured cost (see `TestLargePasswordDoS`): 1 MB password = ~25 ms on this machine, roughly 2× the base hash cost. Argon2id is memory-bound (40 MiB), not input-bound, so growing the password length does not meaningfully amplify CPU time. Original "multi-second burn" claim was wrong.

Still worth capping — defense in depth, avoids pointless work, rejects obviously hostile input early. Not a real DoS vector.

**Fix:** cap at 128 bytes at handler boundary. LOW priority.

---

### RISK-3 (LOW): No rate limit on password reset emails

`securityTimerMiddleware(~1.5s)` per request. No per-user or per-IP counter. Known email → spammed reset mails. Mild email-bomb vector.

**Fix:** skip send if last reset < 5 min ago, silent success response.

---

### RISK-4 (LOW): No bot protection on registration

`// TODO(asaf): Do something to prevent bot registration` — explicit tech debt.

---

### RISK-5 (INFO): `X-Forwarded-For` trusted in registration log

Trivially spoofed outside Cloudflare. Log reliability only, no auth impact.

---

### RISK-6 (INFO): `CookieSecure` has no runtime guard

No startup assertion that `config.Config.Env == Live` implies `CookieSecure == true`. Misconfigured deploy → session cookie over HTTP.

**Fix:** startup panic if `Live && !CookieSecure`.

---

## Modernization (OWASP 2023)

### MOD-1: Argon2id params below 2023 baseline

**File:** `src/auth/auth.go:163-168`

Current: `t=1, m=40 MiB, p=1`. OWASP 2023 min: `t=1, m=64 MiB, p=4`. Not catastrophic. Bump params and extend `IsOutdated()` to re-hash on param drift too.

---

### MOD-2: No "sign out all other sessions"

`CreateSession` inserts without invalidating old. Sessions accumulate. No settings UI to revoke. Fix: add user-facing action that calls `DeleteSessionForUser` then re-creates current session.

---

### MOD-3: `securityTimerMiddleware` not on token-submit endpoints

`EmailConfirmationSubmit` / `DoPasswordResetSubmit` lack timing floor. Tokens are 122-bit UUIDs so not practically enumerable. INFO.

---

## What Is Done Well

- Argon2id new hashes, automatic rehash of legacy Django PBKDF2 on login.
- `subtle.ConstantTimeCompare` for password hash compare.
- `crypto/rand` for all tokens.
- Logout deletes session row immediately.
- Ban deletes all user sessions.
- Periodic cleanup of expired sessions + tokens.
- CSRF synchronizer token pattern with per-session DB token.
- Discord OAuth state = one-time `pending_login` token, DB-validated, 10-min expiry.
- `SafeRedirectUrl` uses full host match (not prefix).
- Duplicate registration silent (no enumeration).
- Admin routes return 404 (no endpoint probe).
- Banned users get generic failure (no status leak).

---

## Priority Order

1. **RISK-1** brute-force protection — architectural, medium effort, only remaining MEDIUM.
2. **BUG-2** inactive user oracle — one-line reorder.
3. **BUG-1** Discord nil ptr — one-line fix, ugly crash.
4. **MOD-1** Argon2id params — parameter bump, re-hash on login.
5. **RISK-6** `CookieSecure` startup guard — one-line assertion.
6. **RISK-2** password length cap — one-line, defense in depth.
7. Cosmetic: **BUG-3**, **BUG-4**, **MOD-3**, **RISK-3**, **RISK-5**.

---

## Test File

`src/auth/auth_security_test.go`:
- Real tests that exercise production code: `HashAndCheckPassword`, `SessionIdEntropy`, `CSRFTokenEntropy`, `DeleteSessionCookiePath`, `LargePasswordDoS`.
- Source-pattern regression tests (fail until fix lands): `LoginActionInactiveOrdering`, `DiscordCallbackNilDeref`, `CSRFMiddlewareConstantTime`, `DeleteSessionCookiePath`, `Argon2idParams`.

Earlier iteration contained "documentation tests" that only called `t.Log` and always passed. Those are removed. All tests now either execute production code or assert on source patterns so they fail when the bug is present.

Run:
```
go test ./src/auth/... -v
```

Expected: failing tests for each unfixed bug (regression guards). Passing tests for roundtrip + entropy.
