# `HashPassword` drops `rand.Reader` error

**File:** `src/auth/auth.go:155-179`
**Severity:** Low — currently unreachable on any supported platform, but the failure mode is catastrophic
**Status:** Confirmed

## The bug

```go
func HashPassword(password string) HashedPassword {
    salt := make([]byte, saltLength)
    io.ReadFull(rand.Reader, salt)
    saltEnc := base64.StdEncoding.EncodeToString(salt)
    // ... argon2 over (password, salt) ...
}
```

`io.ReadFull`'s error is discarded. If the system RNG ever fails, `salt` stays all-zero, the resulting hash is deterministic for any given password, and every user who registers while the RNG is broken gets the *same* salt. Cross-user hash collisions become trivially detectable and rainbow tables apply.

On modern Linux/macOS/Windows, `crypto/rand.Reader` is backed by `getrandom(2)` / `RtlGenRandom` / `/dev/urandom` and does not fail once initialized. On Go 1.19+ the reader blocks on first read until the kernel CSPRNG is seeded. So the reachable failure surface is essentially zero in normal operation.

The concern is less "will this happen" and more "if it ever does, the failure is silent and permanent." The hash is written to the DB; by the time anyone notices, damage is done.

## Fix

Panic on the error. This is the same posture taken by `crypto/tls`, `crypto/ecdsa`, and every other stdlib consumer of `rand.Reader`:

```go
if _, err := io.ReadFull(rand.Reader, salt); err != nil {
    panic(fmt.Sprintf("failed to read random salt: %v", err))
}
```

`HashPassword` already has no error return, so either panic or refactor to `HashPassword(password string) (HashedPassword, error)` and propagate. Panic is defensible here — a broken RNG is an unrecoverable system fault.
