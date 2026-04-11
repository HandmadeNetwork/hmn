# Bug 016: Unchecked ParseConfig error causes nil pointer dereference

**File:** `src/db/conn.go`  
**Functions:** `NewConnWithConfig` (line 42), `NewConnPoolWithConfig` (line 69)  
**Severity:** Medium — confusing panic with no actionable message; masks the actual configuration error  
**Reproduced by:** `TestBug_ParseConfigNilDeref_SingleConn`, `TestBug_ParseConfigNilDeref_Pool` in `src/db/db_bugs_test.go`

## Description

Both `NewConnWithConfig` and `NewConnPoolWithConfig` call `pgx.ParseConfig` /
`pgxpool.ParseConfig` and immediately use the returned pointer **without checking
the error**:

```go
// NewConnWithConfig (conn.go:42-52)
pgcfg, err := pgx.ParseConfig(cfg.DSN())   // ← err NOT checked

pgcfg.Tracer = multiTracer{ ... }           // ← nil deref if pgcfg == nil

conn, err := pgx.ConnectConfig(ctx, pgcfg) // ← err variable reassigned here
```

The second `:=` (for `conn, err :=`) reassigns the `err` variable, so the Go
compiler does not flag the first assignment as unused. The parse error is
silently discarded.

When `ParseConfig` returns `(nil, err)`, the code panics on the `pgcfg.Tracer`
assignment with:

```
runtime error: invalid memory address or nil pointer dereference
```

instead of the intended panic from `oops.New(err, "failed to connect to database")`.

The same pattern appears in `NewConnPoolWithConfig`:

```go
pgcfg, err := pgxpool.ParseConfig(cfg.DSN()) // ← err NOT checked

pgcfg.MinConns = cfg.MinConn                 // ← nil deref
pgcfg.MaxConns = cfg.MaxConn
pgcfg.ConnConfig.Tracer = multiTracer{ ... }

conn, err := pgxpool.NewWithConfig(ctx, pgcfg) // ← err variable reassigned
```

## Trigger conditions

`pgx.ParseConfig` / `pgxpool.ParseConfig` return `(nil, err)` for:

| Condition | Example DSN fragment |
|-----------|----------------------|
| Port out of range | `port=0`, `port=99999` |
| Invalid libpq key=value syntax | `user=x @@bad` |
| Unsupported sslmode | `sslmode=invalid` |

`port=0` is the most likely real-world trigger: it is the `int32` zero value and
will appear in the DSN whenever neither the caller nor `config.Config.Postgres`
supplies a port (e.g., in tests that do not load the full config).

## Fix

Check the error immediately after each `ParseConfig` call:

```go
pgcfg, err := pgx.ParseConfig(cfg.DSN())
if err != nil {
    panic(oops.New(err, "failed to parse database config"))
}
pgcfg.Tracer = multiTracer{ ... }
```

Apply the same pattern to `pgxpool.ParseConfig` in `NewConnPoolWithConfig`.
