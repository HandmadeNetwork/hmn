# Database connection code audit

Audited: `src/db/conn.go`, `src/db/db.go`, `src/db/query_builder.go`
Go version target: 1.25 (per `go.mod`)

Severity rules:
- Panic / crash → not HIGH (process recovers, supervisor restarts).
- Bug reachable only via deploy-time config / internal state → one tier lower than identical runtime-reachable bug.
- No security impact → rated on correctness / stability only.

---

## Bugs

### Bug 016 (INFO) — Unchecked ParseConfig error → nil pointer dereference

**File:** `src/db/conn.go:42`, `conn.go:69`
**Detail:** `bugs/016-db-parsecfg-nil-deref.md`
**Tests:** `TestBug_ParseConfigNilDeref_SingleConn`, `TestBug_ParseConfigNilDeref_Pool`

Both `NewConnWithConfig` and `NewConnPoolWithConfig` drop the error from `pgx.ParseConfig` / `pgxpool.ParseConfig` and immediately dereference the returned pointer. The `err` variable is silently shadowed by the later `pgx.ConnectConfig` / `pgxpool.NewWithConfig` assignment — the compiler does not flag this.

Real bug, confirmed: `pgx.ParseConfig` returns `(nil, err)` for `port=0`, `port=99999`, negative ports, etc.

**Severity reassessment (was not rated in original audit):** INFO, not a security bug.
- Triggered only by a malformed DSN, which comes from `config.Config.Postgres` (deploy-time env / config file). Not attacker-reachable at runtime.
- Triggered only during startup (`NewConn*`). If it fires, the process never starts, supervisor will restart into the same nil-deref loop until config is fixed.
- Real-world impact: developer sees `runtime error: invalid memory address` instead of `failed to parse database config: invalid port (outside range)`. Diagnostic UX regression, not a fault tolerance gap.
- Per the rule "bugs needing internal tampering → one tier lower" and "panic is not HIGH": this is an INFO-level clarity fix, not a LOW/MEDIUM bug.

**Fix:** check `err` immediately after each `ParseConfig` call and panic with a wrapped error before touching the returned pointer. One-line fix in each function.

---

### Bug 017 (LOW) — Data race on rows.Close() via concurrent Iterator close paths

**File:** `src/db/db.go:569` (`Iterator.Close`), `db.go:247` (goroutine)
**Detail:** `bugs/017-db-iterator-concurrent-close-race.md`
**Test:** `TestBug_IteratorConcurrentClose` (requires `-race`)

`QueryIterator` spawns a goroutine that calls `it.Close()` on context cancellation. `Iterator.Close()` calls `it.rows.Close()` with no mutex protection. If the caller also calls `Close()` at the same time (e.g., `defer it.Close()` in a request whose context is cancelled mid-iteration), both paths reach `it.rows.Close()` concurrently. The `it.closed` buffered channel guards only the goroutine's exit, not the rows access.

Real bug, confirmed by inspection. Timing window:
```
T0  goroutine: <-ctx.Done() wakes
T1  goroutine: it.Close() → it.rows.Close() begins
T2  caller:    it.Close() → it.rows.Close() begins (concurrent with T1)
T3  both:      race on pgx internal rows state
```

**Severity reassessment (was not rated in original audit):** LOW.
- No security impact. Stability / correctness bug.
- Trigger is common (`defer it.Close()` with cancellable context) but the window is narrow (both paths must land inside `rows.Close()` simultaneously).
- Worst case: pgx connection-pool corruption. Pool self-heals via reconnection. Not data loss.
- `-race` will catch it in dev; production impact depends on pgx's internal tolerance for double-close.

**Fix:** wrap `rows.Close()` in `sync.Once`. Simplest, correct, zero-cost in the non-racing path. See M4 for the larger rewrite that eliminates the goroutine entirely.

---

## Modernisation opportunities (Go 1.25)

### M1 — `time.Now().Sub(x)` → `time.Since(x)`
**File:** `src/db/db.go:229`

```go
duration := time.Now().Sub(queryStart)  // current
duration := time.Since(queryStart)      // idiomatic
```

Cosmetic. Flagged by `gosimple`.

---

### M2 — Untyped string context keys
**Files:** `src/db/conn.go:129`, `src/perf/perf.go:201`

```go
const perfBlockContextKey = "__dbPerfBlock"   // conn.go
const PerfContextKey = "HMNPerf"               // perf/perf.go
```

Plain string keys can collide with third-party middleware. Idiom since Go 1.7:

```go
type contextKey string
const perfBlockContextKey contextKey = "__dbPerfBlock"
```

Flagged by `go vet` / staticcheck SA1029.

---

### M3 — `Iterator.Close` type parameter named `any`
**File:** `src/db/db.go:569`

```go
func (it *Iterator[any]) Close() {   // shadows predeclared identifier `any`
```

All other `Iterator[T]` methods use `T`. Rename to `T` for consistency — purely cosmetic.

---

### M4 — Adopt `iter.Seq` / range-over-func (Go 1.23+)
**File:** `src/db/db.go` — `Iterator[T]`, `QueryIterator`, `Query`, `QueryOne`

`iter.Seq[V]` and range-over-function are available at Go 1.25. `Iterator[T]` could expose:

```go
func (it *Iterator[T]) All() iter.Seq[*T] {
    return func(yield func(*T) bool) {
        defer it.Close()
        for {
            val, ok := it.Next()
            if !ok || !yield(val) {
                return
            }
        }
    }
}
```

Callers:
```go
for row := range it.All() {
    // break triggers defer it.Close() — no goroutine needed
}
```

**Larger benefit tied to Bug 017:** the context-cancel goroutine in `QueryIterator` (the source of the race) can be eliminated. Range-over-func runs `yield` on the caller's goroutine, so `break` unwinds through `defer it.Close()` on that same goroutine. No concurrent close path = no race.

Additive: existing `.Next()` / `.Close()` / `.ToSlice()` API stays, callers migrate incrementally.

---

## Items checked and found clean

| Area | Verdict |
|------|---------|
| `QueryBuilder.Add` placeholder counting | Correct — panics on mismatch |
| `Iterator.Next` reflection path | No issues found |
| `followPathThroughStructs` | Correctly initialises nil pointer fields |
| `setValueFromDB` scalar / pointer / slice paths | No issues found |
| `getColumnNamesAndPaths` anonymous-struct prefix logic | Correct per `TestPaths` |
| `compileQuery` regex replacement | Correct |
| `multiTracer` query tracing | Correct |
| Context-cancel goroutine exit logic (`it.closed` channel) | Logic correct; Bug 017 is the race on `rows.Close()` itself |

---

## Priority

Neither bug is security-critical. Order by real impact:

1. **Bug 017** — real concurrency bug with common trigger. Fix with `sync.Once` (small) or M4 rewrite (larger cleanup).
2. **M2** — context key shadowing. One-line fix per package, flagged by vet.
3. **M3** — Iterator `any` shadow. One-character fix.
4. **M1** — `time.Since`. Cosmetic.
5. **Bug 016** — diagnostic clarity. One-line fix per function. No runtime impact once deployed.
6. **M4** — iter.Seq adoption. Architectural; subsumes Bug 017 fix.

---

## Test File Assessment

`src/db/db_bugs_test.go` had three tests. Two were theater; one was real.

### `TestBug_ParseConfigNilDeref_SingleConn` / `_Pool` — BROKEN

Neither test exercised production code. Both replicated the three-line pattern from `conn.go` inline, then asserted:

```go
assert.False(t, isNilDeref, "BUG REPRODUCED: ...")
```

Since the test itself manually dereferences the `nil` config, `isNilDeref` is **always true**, and `assert.False` always fails. The tests are permanently red regardless of whether `conn.go` is fixed. Fixing `conn.go` would not change the outcome because the test never calls `NewConnWithConfig` or `NewConnPoolWithConfig`.

This is a regression test that tests nothing about the thing under test. Grade: useless.

**Fixed below** — rewritten to actually call the production functions with a known-bad port (`99999`), recover the resulting panic, and assert the recovered message is a wrapped parse error, not a Go runtime nil-deref.

### `TestBug_IteratorConcurrentClose` — GOOD

Directly exercises production `Iterator.Close`. Mirrors the real `QueryIterator` goroutine pattern. Uses a `racyRows` mock whose `closeCount++` writes without a mutex, so the `-race` detector will fire if `Close` is called concurrently from two goroutines. 300-iteration loop improves race detection reliability.

One caveat: requires `-race` flag. On a Windows machine without cgo, `-race` cannot run — the test silently passes under `go test` without `-race`. Worth adding a build-tag or a runtime guard.

Grade: correct regression test. Keep, with added `-race` documentation.
