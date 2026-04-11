# `GetBotEvents` returns a slice sharing a backing array with concurrent writers

**File:** `src/discord/gateway.go:32-52`
**Severity:** Medium — data race, flagged by `-race`, observable as torn reads
**Status:** Confirmed

## The bug

```go
var botEvents = make([]BotEvent, 0, 1000)
var botEventsMutex = sync.Mutex{}

func RecordBotEvent(name, extra string) {
    botEventsMutex.Lock()
    defer botEventsMutex.Unlock()
    if len(botEvents) > 1000 {
        botEvents = botEvents[len(botEvents)-500:]
    }
    botEvents = append(botEvents, BotEvent{...})
}

func GetBotEvents() []BotEvent {
    botEventsMutex.Lock()
    defer botEventsMutex.Unlock()
    return botEvents[:]
}
```

`GetBotEvents` locks, slices, and unlocks. The returned slice shares a backing array with the package-level `botEvents`. After the caller has the slice, a concurrent `RecordBotEvent` can take the mutex and `append` into the same backing array. Appends that do not exceed `cap(botEvents)` write into the existing backing store, at indices that may or may not overlap what the caller is iterating. Either way, the reader and the writer touch the same memory without holding a common lock — `go test -race` will flag it.

The reslicing path makes it worse:

```go
botEvents = botEvents[len(botEvents)-500:]
```

After this, `botEvents` points into the middle of the old backing array. `cap(botEvents)` is whatever remains (`oldcap - (oldlen-500)`). Subsequent appends write into the same backing array as any prior reader holding `botEvents[:]` from before the reslice. The two slices disagree about where "index 0" is, so the same element has different indices in the reader and the writer.

## Impact

`GetBotEvents` is exposed via the admin tools page (grep: `GetBotEvents` is used to render a debug list of recent Discord events). Torn reads are cosmetic there — a garbled event in an admin UI is annoying, not critical — but:

1. The race is real under `-race`, which means CI will flag it if a test exercises both functions concurrently.
2. If this slice is ever passed to JSON serialization or a template, a torn read can produce nonsense strings. `BotEvent` has two string fields; a torn string header (pointer + length) is undefined behavior at the Go level.

## Fix

Return a copy under the lock:

```go
func GetBotEvents() []BotEvent {
    botEventsMutex.Lock()
    defer botEventsMutex.Unlock()
    out := make([]BotEvent, len(botEvents))
    copy(out, botEvents)
    return out
}
```

The slice is capped at ~1000 entries, so the copy cost is trivial and the ownership story becomes clean: the caller owns its copy, the writer owns `botEvents`.

## Bonus nit

`if len(botEvents) > 1000 { botEvents = botEvents[len(botEvents)-500:] }` means the ring buffer can grow to `len == 1001` before it ever shrinks. If `cap(botEvents)` was 1000 exactly (from the `make`), append at len 1000 grows to a new backing array of cap ~2000. The "bounded to 1000 entries" claim in the variable's capacity hint is misleading. Not a bug, just confusing.
