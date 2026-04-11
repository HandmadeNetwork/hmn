# `BackgroundPreviewGeneration` leaks HTTP response bodies

**File:** `src/assets/assets.go:298-320`
**Severity:** Medium — file-descriptor + connection leak scaling with asset backlog
**Status:** Confirmed

## The bug

```go
for _, asset := range assets {
    // ...
    resp, err := http.Get(assetUrl)
    if err != nil || resp.StatusCode != 200 {
        log.Error().Err(err).Msg("Failed to fetch asset file for preview generation")
        continue
    }
    defer resp.Body.Close()
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        log.Error().Err(err).Msg("Failed to read asset body for preview generation")
        continue
    }
    // ... extract, upload, update DB ...
}
```

Two leaks, both in the same few lines:

### 1. `defer` inside a loop

`defer resp.Body.Close()` does not run until the enclosing function (the goroutine) returns. In a long-running background worker processing every video asset in the DB, that means every response body stays open until the *entire job* finishes. For a large backlog this:

- Keeps a TCP connection in the `net/http` idle pool per asset (usually capped, but the underlying fds stay tied up until GC sweeps the bodies).
- Keeps the `bytes.Buffer` / `io.ReadCloser` chain alive in memory.
- Can trip the OS file-descriptor ulimit on the worker process.

### 2. Non-200 path never closes at all

```go
if err != nil || resp.StatusCode != 200 {
    // log
    continue
}
defer resp.Body.Close()
```

When `err == nil` but `resp.StatusCode != 200`, `resp` is non-nil and its body has not been closed — `continue` jumps past the `defer` statement entirely. The body of a non-200 response is just as leak-prone as a 200. In practice S3 can return 403/404/500 here for deleted or transient assets, and each one leaks a connection for the rest of the job run.

## Fix

Push the fetch into a helper that owns the lifetime, or inline an explicit close. Inline version:

```go
func fetchAssetBody(url string) ([]byte, error) {
    resp, err := http.Get(url)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    if resp.StatusCode != 200 {
        return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
    }
    return io.ReadAll(resp.Body)
}

for _, asset := range assets {
    // ...
    body, err := fetchAssetBody(hmnurl.BuildS3Asset(asset.S3Key))
    if err != nil {
        log.Error().Err(err).Msg("Failed to fetch asset")
        continue
    }
    // ...
}
```

The helper scopes the `defer` to a single iteration, closes on every path including the non-200 branch, and has room for a future timeout via `http.Client{Timeout: ...}` — the current code uses `http.Get` with the default (no) timeout, which is its own problem.
