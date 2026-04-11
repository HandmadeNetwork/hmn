# `ParseArgon2idConfig` panics on malformed config string

**File:** `src/auth/auth.go:77-106`
**Severity:** Medium — any malformed hash row crashes the login path
**Status:** Confirmed

## The bug

```go
func ParseArgon2idConfig(cfg string) (Argon2idConfig, error) {
    parts := strings.Split(cfg, ",")

    t64, err := strconv.ParseUint(parts[0][2:], 10, 32)
    // ...
    m64, err := strconv.ParseUint(parts[1][2:], 10, 32)
    // ...
    p64, err := strconv.ParseUint(parts[2][2:], 10, 8)
    // ...
    l64, err := strconv.ParseUint(parts[3][2:], 10, 32)
```

No length check on `parts`, and no length check on the individual entries before slicing `[2:]`. Four distinct panics lurking:

- `cfg == ""` → `parts == [""]`, `parts[1]` panics with index out of range.
- `cfg == "t=1"` → `parts == ["t=1"]`, same.
- `cfg == "t=1,"` → `parts[1] == ""`, `parts[1][2:]` panics with "slice bounds out of range".
- `cfg == "t=1,m,p=1,l=64"` → `parts[1] == "m"`, length 1, `[2:]` panics.

The shape `t=X,m=Y,p=Z,l=W` comes from `Argon2idConfig.String()` (`auth.go:108-110`). The only path that feeds user input into this parser is `ParsePasswordString` reading a row out of `hmn_user.password`. So the question is: can a malformed row ever land in that column?

- Django port migration left historical rows. Intended.
- Manual DBA edit (`UPDATE hmn_user SET password = ...` for a test user).
- A future bug upstream of `HashPassword().String()`.

Any of those crashes the worker goroutine mid-login. With `panicCatcherMiddleware` the request gets a 500 error page, but the user cannot sign in and the log is a raw stack trace.

## Fix

```go
func ParseArgon2idConfig(cfg string) (Argon2idConfig, error) {
    parts := strings.Split(cfg, ",")
    if len(parts) != 4 {
        return Argon2idConfig{}, oops.New(nil, "argon2id config must have 4 fields, got %d", len(parts))
    }
    parse := func(s, prefix string, bits int) (uint64, error) {
        if !strings.HasPrefix(s, prefix) {
            return 0, oops.New(nil, "expected prefix %q, got %q", prefix, s)
        }
        return strconv.ParseUint(s[len(prefix):], 10, bits)
    }
    t, err := parse(parts[0], "t=", 32)
    if err != nil { return Argon2idConfig{}, err }
    m, err := parse(parts[1], "m=", 32)
    if err != nil { return Argon2idConfig{}, err }
    p, err := parse(parts[2], "p=", 8)
    if err != nil { return Argon2idConfig{}, err }
    l, err := parse(parts[3], "l=", 32)
    if err != nil { return Argon2idConfig{}, err }
    return Argon2idConfig{
        Time: uint32(t), Memory: uint32(m),
        Threads: uint8(p), KeyLength: uint32(l),
    }, nil
}
```

Same style fix in `ParsePasswordString` at `auth.go:48-60` — it already length-checks via `len(pieces) < 4`, so the shape is fine, but worth a once-over.
