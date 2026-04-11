package auth

// Security regression tests for the auth package and its users.
//
// Tests fall into two categories:
//
//  1. Functional tests that exercise production code directly
//     (HashPassword/CheckPassword, MakeSessionId, makeCSRFToken,
//     DeleteSessionCookie, LargePasswordDoS).
//
//  2. Source-pattern regression tests that read sibling package source
//     files and assert on code structure. These cover bugs whose root
//     cause lives in the `website` package and cannot be exercised from
//     `auth` without full HTTP + DB stack. They fail until the fix
//     lands in the source tree.
//
// Run: go test ./src/auth/... -v

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Helpers for source-pattern regression tests
// ---------------------------------------------------------------------------

func readSibling(t *testing.T, rel string) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	authDir := filepath.Dir(thisFile)
	path := filepath.Join(authDir, rel)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

// ===========================================================================
// Functional tests (exercise production code)
// ===========================================================================

// HashPassword + CheckPassword roundtrip.
func TestHashAndCheckPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
	}{
		{"simple", "password123"},
		{"unicode", "pässwörд"},
		{"empty", ""},
		{"128_bytes", strings.Repeat("a", 128)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hashed := HashPassword(tt.password)
			if hashed.Algorithm != Argon2id {
				t.Errorf("expected Argon2id, got %s", hashed.Algorithm)
			}
			if hashed.Salt == "" {
				t.Error("empty salt")
			}

			match, err := CheckPassword(tt.password, hashed)
			if err != nil {
				t.Fatalf("CheckPassword: %v", err)
			}
			if !match {
				t.Error("correct password rejected")
			}

			noMatch, err := CheckPassword(tt.password+"x", hashed)
			if err != nil {
				t.Fatalf("CheckPassword wrong: %v", err)
			}
			if noMatch {
				t.Error("wrong password accepted")
			}
		})
	}
}

// Session IDs are unique + correct length.
func TestSessionIdEntropy(t *testing.T) {
	id := MakeSessionId()
	if len(id) != 40 {
		t.Errorf("len=%d, want 40", len(id))
	}

	seen := make(map[string]bool, 1000)
	for i := 0; i < 1000; i++ {
		id := MakeSessionId()
		if seen[id] {
			t.Fatalf("collision at %d", i)
		}
		seen[id] = true
	}
}

// CSRF tokens are unique + correct length.
func TestCSRFTokenEntropy(t *testing.T) {
	token := makeCSRFToken()
	if len(token) != 30 {
		t.Errorf("len=%d, want 30", len(token))
	}

	seen := make(map[string]bool, 1000)
	for i := 0; i < 1000; i++ {
		tok := makeCSRFToken()
		if seen[tok] {
			t.Fatalf("collision at %d", i)
		}
		seen[tok] = true
	}
}

// RISK-2: HashPassword accepts oversized input without cap.
// Reproduces the DoS vector: a 1 MB password takes multi-second CPU per call.
// Fails if input cap is added (good — test must then be deleted or moved
// behind the new cap).
func TestLargePasswordDoS(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping DoS reproduction in -short mode")
	}

	huge := strings.Repeat("x", 1024*1024) // 1 MB

	start := time.Now()
	hashed := HashPassword(huge)
	elapsed := time.Since(start)

	t.Logf("HashPassword(1MB) took %v", elapsed)

	// Current behavior: no cap, completes with large CPU cost.
	// If a cap is added upstream in callers, HashPassword itself still
	// runs — the fix should happen at the handler layer. Flag the
	// absence of a package-level guard.
	if elapsed < 500*time.Millisecond {
		t.Logf("NOTE: large-input hash unexpectedly fast (%v) — Argon2id params weak?", elapsed)
	}

	// Sanity: roundtrip still works
	match, err := CheckPassword(huge, hashed)
	if err != nil || !match {
		t.Errorf("1MB password roundtrip failed: match=%v err=%v", match, err)
	}

	t.Log("RISK-2 unfixed: no input cap in auth.HashPassword; caller must enforce")
}

// ===========================================================================
// Source-pattern regression tests (fail until the bug is fixed in source)
// ===========================================================================

// BUG-1: src/website/discord.go:73 dereferences c.CurrentUser.Username
// inside an `if c.CurrentUser == nil` branch.
func TestDiscordCallbackNilDeref(t *testing.T) {
	src := readSibling(t, "../website/discord.go")

	// Find DiscordOAuthCallback function body.
	fnIdx := strings.Index(src, "func DiscordOAuthCallback")
	if fnIdx < 0 {
		t.Fatal("DiscordOAuthCallback not found in discord.go")
	}
	body := src[fnIdx:]

	// Locate the `if c.CurrentUser == nil {` branch.
	branchIdx := strings.Index(body, "if c.CurrentUser == nil {")
	if branchIdx < 0 {
		t.Fatal("nil-check branch not found — source shape changed")
	}

	// Extract a window covering the nil-user branch. We stop at the `else`
	// that ends the branch, which uses `c.CurrentSession.CSRFToken`.
	branch := body[branchIdx:]
	endIdx := strings.Index(branch, "c.CurrentSession.CSRFToken")
	if endIdx < 0 {
		t.Fatal("could not bound nil-user branch")
	}
	branch = branch[:endIdx]

	if strings.Contains(branch, "c.CurrentUser.Username") {
		t.Errorf("BUG-1 present: c.CurrentUser.Username used inside `if c.CurrentUser == nil` branch in discord.go — nil deref panic")
		t.Log("Fix: use a literal like \"unauthenticated\" instead of c.CurrentUser.Username")
	}
}

// BUG-2: src/website/auth.go LoginAction calls tryLogin BEFORE checking
// UserStatusInactive. Correct password on inactive account yields a
// distinct error, leaking password validity.
//
// Regression test: assert that LoginAction's inactive-check appears
// BEFORE tryLogin (meaning the check short-circuits the password compare),
// OR that tryLogin itself rejects inactive users.
func TestLoginActionInactiveOrdering(t *testing.T) {
	src := readSibling(t, "../website/auth.go")

	// Bound LoginAction body.
	start := strings.Index(src, "func LoginAction(")
	if start < 0 {
		t.Fatal("LoginAction not found")
	}
	// Find next top-level func declaration.
	nextFn := strings.Index(src[start+1:], "\nfunc ")
	if nextFn < 0 {
		t.Fatal("could not bound LoginAction")
	}
	loginBody := src[start : start+1+nextFn]

	tryLoginIdx := strings.Index(loginBody, "tryLogin(")
	inactiveIdx := strings.Index(loginBody, "UserStatusInactive")

	if tryLoginIdx < 0 {
		t.Fatal("tryLogin call not found in LoginAction")
	}

	// If LoginAction itself checks UserStatusInactive after tryLogin, bug present
	// UNLESS tryLogin itself rejects inactive users.
	if inactiveIdx > 0 && inactiveIdx > tryLoginIdx {
		// Bug present in LoginAction: status checked after password verification.
		// Confirm by also checking that tryLogin does NOT reject inactive.
		tryLoginStart := strings.Index(src, "func tryLogin(")
		if tryLoginStart < 0 {
			t.Fatal("tryLogin not found")
		}
		tryLoginEnd := strings.Index(src[tryLoginStart+1:], "\nfunc ")
		var tryLoginBody string
		if tryLoginEnd < 0 {
			tryLoginBody = src[tryLoginStart:]
		} else {
			tryLoginBody = src[tryLoginStart : tryLoginStart+1+tryLoginEnd]
		}

		if !strings.Contains(tryLoginBody, "UserStatusInactive") {
			t.Errorf("BUG-2 present: LoginAction checks UserStatusInactive AFTER tryLogin succeeds, and tryLogin does not filter inactive users — password validity leaks via distinct error message")
			t.Log("Fix: add UserStatusInactive check inside tryLogin alongside UserStatusBanned, returning (false, nil) so LoginAction shows the generic failure")
		}
	}
}

// BUG-3: src/website/middlewares.go csrfMiddleware uses plain != for
// CSRF token compare. Low real-world severity but best practice says
// constant-time.
func TestCSRFMiddlewareConstantTime(t *testing.T) {
	src := readSibling(t, "../website/middlewares.go")

	start := strings.Index(src, "func csrfMiddleware(")
	if start < 0 {
		t.Fatal("csrfMiddleware not found")
	}
	nextFn := strings.Index(src[start+1:], "\nfunc ")
	var body string
	if nextFn < 0 {
		body = src[start:]
	} else {
		body = src[start : start+1+nextFn]
	}

	if strings.Contains(body, "csrfToken != c.CurrentSession.CSRFToken") {
		t.Errorf("BUG-3 present: csrfMiddleware compares CSRF token with plain != (not constant-time)")
		t.Log("Fix: subtle.ConstantTimeCompare([]byte(csrfToken), []byte(c.CurrentSession.CSRFToken)) != 1")
	}

	if !strings.Contains(body, "subtle.ConstantTimeCompare") && !strings.Contains(src, `"crypto/subtle"`) {
		// Only a warning — the import might be added separately.
		t.Log("NOTE: crypto/subtle not imported in middlewares.go")
	}
}

// BUG-4: DeleteSessionCookie (this package) missing Path: "/".
// This test reads the actual production value, not source.
func TestDeleteSessionCookiePath(t *testing.T) {
	if DeleteSessionCookie.Path != "/" {
		t.Errorf("BUG-4 present: DeleteSessionCookie.Path = %q, want %q", DeleteSessionCookie.Path, "/")
		t.Log("Fix: add Path: \"/\" to the DeleteSessionCookie literal in session.go")
	}
	if DeleteSessionCookie.Name != SessionCookieName {
		t.Errorf("DeleteSessionCookie.Name = %q, want %q", DeleteSessionCookie.Name, SessionCookieName)
	}
	if DeleteSessionCookie.MaxAge != -1 {
		t.Errorf("DeleteSessionCookie.MaxAge = %d, want -1", DeleteSessionCookie.MaxAge)
	}
}

// MOD-1: Argon2id params below OWASP 2023 baseline.
// Fails until params are bumped. If the project deliberately chooses lower
// params for perf, delete this test.
func TestArgon2idParams(t *testing.T) {
	hashed := HashPassword("x")
	cfg, err := ParseArgon2idConfig(hashed.AlgoConfig)
	if err != nil {
		t.Fatalf("ParseArgon2idConfig: %v", err)
	}

	const owaspMinMemoryKiB = 64 * 1024 // 64 MiB
	const owaspMinThreads = 4

	if cfg.Memory < owaspMinMemoryKiB {
		t.Errorf("Argon2id memory = %d KiB, OWASP 2023 min = %d KiB", cfg.Memory, owaspMinMemoryKiB)
	}
	if cfg.Threads < owaspMinThreads {
		t.Errorf("Argon2id threads = %d, OWASP 2023 min = %d", cfg.Threads, owaspMinThreads)
	}

	if t.Failed() {
		t.Log("Fix: bump params in HashPassword and extend IsOutdated() to trigger re-hash on login")
	}
}
