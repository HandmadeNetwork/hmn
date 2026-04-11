# Bug 017: Data race on rows.Close() in Iterator concurrent close paths

**File:** `src/db/db.go`  
**Functions:** `Iterator.Close` (line 569), `QueryIterator` goroutine (line 247)  
**Severity:** Medium — data race; observable with `-race`; can corrupt connection state  
**Reproduced by:** `TestBug_IteratorConcurrentClose` in `src/db/db_bugs_test.go` (run with `-race`)

## Description

`QueryIterator` spawns a goroutine to close the iterator if the context is
cancelled:

```go
// db.go:247-257
go func() {
    done := ctx.Done()
    if done == nil {
        return
    }
    select {
    case <-done:
        it.Close()      // ← goroutine close path
    case <-it.closed:
    }
}()
```

`Iterator.Close` does:

```go
// db.go:569-575
func (it *Iterator[any]) Close() {
    it.rows.Close()              // ← no lock
    select {
    case it.closed <- struct{}{}:
    default:
    }
}
```

If the context is cancelled **and** the caller also calls `it.Close()` at the
same time (e.g., in a `defer it.Close()` after a request is cancelled), both
code paths call `it.rows.Close()` concurrently with no synchronisation.

The `it.closed` buffered channel (size 1) only serves to exit the goroutine; it
does not prevent `it.rows.Close()` from being called simultaneously from two
goroutines.

## Race window

```
goroutine                    caller
─────────────────────────────────────────
ctx cancelled →
  it.rows.Close() ────┐
                      │  it.rows.Close()   ← concurrent call
                      └──────────────────── DATA RACE
  it.closed <- {}
                          it.closed <- {} (default, no-op)
```

## Impact

`pgx.Rows` does not document thread-safety for `Close()`. Concurrent calls can
corrupt the internal connection state, potentially deadlocking or returning
wrong data from a subsequent query.

## Fix

Add a `sync.Once` (or a mutex) to `Iterator.Close` so `it.rows.Close()` is only
called once regardless of how many goroutines race to close it:

```go
type Iterator[T any] struct {
    // ... existing fields
    closeOnce sync.Once
    closed    chan struct{}
}

func (it *Iterator[T]) Close() {
    it.closeOnce.Do(func() {
        it.rows.Close()
    })
    select {
    case it.closed <- struct{}{}:
    default:
    }
}
```

Alternatively, replace the `closed` channel with a mutex-guarded boolean and
call `it.rows.Close()` inside the lock.

## Note on type-parameter naming

`Iterator.Close` is also declared with the type parameter literally named `any`:

```go
func (it *Iterator[any]) Close() {   // "any" shadows the built-in predeclared identifier
```

All other `Iterator` methods use `T`. While this compiles and behaves correctly
in Go 1.25, it is misleading and inconsistent. It should be `Iterator[T]`.
