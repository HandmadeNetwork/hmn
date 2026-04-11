# Password hash comparison uses `bytes.Equal`, not constant-time

**File:** `src/auth/auth.go:128, 149`
**Severity:** Low — practical exploitability is bounded, but this is the standard fix
**Status:** Confirmed

## The bug

```go
// Argon2id branch
newHash := argon2.IDKey(...)
newHashEnc := base64.StdEncoding.EncodeToString(newHash)
return bytes.Equal([]byte(newHashEnc), []byte(hashedPassword.Hash)), nil

// Django PBKDF2 branch
newHashEncoded := base64.StdEncoding.EncodeToString(newHash)
return bytes.Equal([]byte(newHashEncoded), []byte(hashedPassword.Hash)), nil
```

`bytes.Equal` short-circuits on the first mismatched byte. Both inputs here are secrets:

- `hashedPassword.Hash` is the stored DB value. The attacker doesn't know it.
- `newHashEnc` is derived from an attacker-supplied password plus a (known-to-attacker, salt-is-not-secret) configuration.

The theoretical attack is: time enough login attempts to learn how many leading bytes of `base64(argon2id(guess))` match the stored hash, narrowing the search space. In practice:

- Network jitter dwarfs single-byte timing differences.
- Argon2id dominates the wall time (40 MiB, 1 pass), so the compare noise is a tiny fraction of the total.
- The attacker would still need to brute-force a password whose hash matches a known prefix, which is essentially the same as cracking the hash offline.

So the real-world risk is tiny, but `crypto/subtle.ConstantTimeCompare` is the standard idiom and there's no downside.

## Fix

```go
import "crypto/subtle"

// Argon2id
return subtle.ConstantTimeCompare(
    []byte(newHashEnc),
    []byte(hashedPassword.Hash),
) == 1, nil
```

Same change at line 149. Note: `ConstantTimeCompare` returns 0 for different-length inputs without timing-safe-length comparison, which is fine here since both sides are base64-encoded fixed-length hashes.

## Related

- `src/website/middlewares.go:118` — CSRF token compare is a plain `!=` string compare. Same mitigation: `subtle.ConstantTimeCompare([]byte(a), []byte(b)) != 1`.
- `src/website/auth.go:965` — one-time token compare is `==`. Token entropy is high (long random), so timing attacks are unrealistic, but the same fix applies.
