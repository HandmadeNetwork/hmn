package db

// Bug reproduction tests for the db package.
//
// Run:
//   go test ./src/db/... -run TestBug -v
//
// The iterator race test requires the race detector:
//   go test ./src/db/... -run TestBug_IteratorConcurrentClose -race -count=50
//
// (Race detector requires cgo, which may not be installed on all dev boxes.
//  Without -race the iterator test is a no-op smoke test.)

import (
	"context"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"

	"git.handmade.network/hmn/hmn/src/config"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// =============================================================================
// Bug 016: Unchecked pgx.ParseConfig / pgxpool.ParseConfig errors in
//          NewConnWithConfig and NewConnPoolWithConfig (src/db/conn.go).
//
// Code shape:
//
//   pgcfg, err := pgx.ParseConfig(cfg.DSN())  // err NOT checked
//   pgcfg.Tracer = multiTracer{...}            // nil deref if pgcfg == nil
//   conn, err := pgx.ConnectConfig(...)        // `err` silently shadowed
//
// A DSN with port=99999 causes pgx.ParseConfig to return (nil, err). Without
// the error check, the next line panics with a runtime nil-deref instead of
// the intended wrapped "failed to parse database config" panic.
//
// These tests drive the production functions directly, recover the panic,
// and assert the recovered value is a wrapped parse error — NOT a Go runtime
// nil-deref.
//
// Bug present → test FAILS (panic is a runtime.Error with "nil pointer").
// Bug fixed   → test PASSES (panic carries the wrapped parse error).
// =============================================================================

// badPortConfig produces a config whose DSN is syntactically invalid.
// port=99999 is out of uint16 range and causes pgx.ParseConfig to return nil.
// All other fields are non-zero so overrideDefaultConfig does not substitute
// defaults from config.Config.Postgres.
func badPortConfig() config.PostgresConfig {
	return config.PostgresConfig{
		User:     "u",
		Password: "p",
		Hostname: "localhost",
		Port:     99999,
		DbName:   "d",
	}
}

// isRuntimeNilDeref reports whether a recovered panic value is the Go runtime
// "nil pointer dereference" error. A correctly-handled ParseConfig error should
// produce a non-runtime panic (oops.Error) with a readable message instead.
func isRuntimeNilDeref(r any) bool {
	if r == nil {
		return false
	}
	if re, ok := r.(runtime.Error); ok {
		msg := strings.ToLower(re.Error())
		return strings.Contains(msg, "nil pointer") || strings.Contains(msg, "invalid memory address")
	}
	return false
}

func TestBug_ParseConfigNilDeref_SingleConn(t *testing.T) {
	var recovered any
	func() {
		defer func() { recovered = recover() }()
		// Drive production code path. This MUST panic — we are passing a bad DSN.
		// The question is: does it panic with a wrapped parse error (fix applied),
		// or a runtime nil-deref (bug present)?
		NewConnWithConfig(badPortConfig())
	}()

	if recovered == nil {
		t.Fatal("expected NewConnWithConfig to panic on bad DSN")
	}

	if isRuntimeNilDeref(recovered) {
		t.Errorf("BUG 016 present: NewConnWithConfig panicked with runtime nil-deref instead of a wrapped parse error.\n"+
			"conn.go does not check the error returned by pgx.ParseConfig before assigning pgcfg.Tracer.\n"+
			"Recovered panic: %v", recovered)
		t.Log("Fix: after `pgcfg, err := pgx.ParseConfig(...)`, check err and panic(oops.New(err, ...)) before touching pgcfg.")
	}
}

func TestBug_ParseConfigNilDeref_Pool(t *testing.T) {
	var recovered any
	func() {
		defer func() { recovered = recover() }()
		NewConnPoolWithConfig(badPortConfig())
	}()

	if recovered == nil {
		t.Fatal("expected NewConnPoolWithConfig to panic on bad DSN")
	}

	if isRuntimeNilDeref(recovered) {
		t.Errorf("BUG 016 present: NewConnPoolWithConfig panicked with runtime nil-deref instead of a wrapped parse error.\n"+
			"conn.go does not check the error returned by pgxpool.ParseConfig before accessing pgcfg.MinConns.\n"+
			"Recovered panic: %v", recovered)
		t.Log("Fix: after `pgcfg, err := pgxpool.ParseConfig(...)`, check err and panic(oops.New(err, ...)) before touching pgcfg.")
	}
}

// =============================================================================
// Bug 017: Data race on rows.Close() between the context-cancel goroutine and
//          the caller (src/db/db.go — Iterator.Close / QueryIterator goroutine).
//
// Iterator.Close calls it.rows.Close() without a lock:
//
//   func (it *Iterator[any]) Close() {
//       it.rows.Close()              // <-- no synchronisation
//       select {
//       case it.closed <- struct{}{}:
//       default:
//       }
//   }
//
// QueryIterator spawns a goroutine that calls it.Close() on ctx.Done(). If the
// caller also calls Close() concurrently (common: `defer it.Close()` in a
// request whose context just got cancelled), both paths reach it.rows.Close()
// at the same time. The `closed` channel guards only the goroutine's exit,
// not the rows access.
//
// Test: exercises the actual Iterator.Close from db.go + a mirror of the
// QueryIterator goroutine pattern, backed by a mock `pgx.Rows` whose state is
// written without a mutex. `go test -race` reports a DATA RACE on the mock's
// internal counter when the bug is present.
// =============================================================================

// racyRows is an intentionally unsynchronised mock of pgx.Rows. The closeCount
// field is written without a lock so the race detector will fire if Close is
// reached from two goroutines simultaneously.
type racyRows struct {
	closeCount int
}

func (r *racyRows) Close()                                       { r.closeCount++ }
func (r *racyRows) Err() error                                   { return nil }
func (r *racyRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (r *racyRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *racyRows) Next() bool                                   { return false }
func (r *racyRows) Scan(_ ...any) error                          { return nil }
func (r *racyRows) Values() ([]any, error)                       { return nil, nil }
func (r *racyRows) RawValues() [][]byte                          { return nil }
func (r *racyRows) Conn() *pgx.Conn                              { return nil }

var _ pgx.Rows = (*racyRows)(nil)

func TestBug_IteratorConcurrentClose(t *testing.T) {
	// Higher iteration count improves the probability that the race detector
	// observes the overlapping window.
	const iterations = 300

	for range iterations {
		ctx, cancel := context.WithCancel(context.Background())

		mock := &racyRows{}
		it := &Iterator[int]{
			rows:             mock,
			destType:         reflect.TypeFor[int](),
			destTypeIsScalar: true,
			closed:           make(chan struct{}, 1),
		}

		// Mirror the goroutine spawned by QueryIterator (db.go:247-257).
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case <-ctx.Done():
				it.Close() // goroutine close path — races with caller below
			case <-it.closed:
			}
		}()

		// Cancel context and close from caller concurrently.
		cancel()
		it.Close() // caller close path

		wg.Wait()
	}

	// Note: this test exercises real production Iterator.Close. Under `-race`,
	// the data race on racyRows.closeCount is reported as a DATA RACE,
	// failing the test. Without `-race`, the test is a pure smoke test and
	// passes regardless of whether Iterator.Close is guarded by sync.Once.
	// Always run this with `go test -race`.
}
