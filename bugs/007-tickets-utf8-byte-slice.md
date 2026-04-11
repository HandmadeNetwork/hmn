# `TicketEditSubmit` truncates by bytes, splits multi-byte UTF-8

**File:** `src/website/tickets.go:626-631`
**Severity:** Low — data quality, not security
**Status:** Confirmed

## The bug

```go
newName := c.Req.Form.Get("name")
newEmail := c.Req.Form.Get("email")

// Trim to reasonable length
newName = newName[:min(len(newName), 500)]
newEmail = newEmail[:min(len(newEmail), 500)]
```

`len(newName)` is the byte length. A name that fills the 500-byte budget may have its final character bisected mid-codepoint, producing a `string` with an invalid UTF-8 tail. That value is then stored in the `ticket.name` column.

Downstream consumers (templates, JSON encoders, logs) generally tolerate invalid UTF-8 by replacing the bad bytes with U+FFFD, but:

- `json.Marshal` will emit `\ufffd` in place of the bad byte, producing a mismatched display name on the client.
- Any `LIKE` match or unique-index check on the column will see the mangled string, not what the user typed.
- `encoding/json` used to error outright on invalid UTF-8; current Go silently substitutes, but that's a compat guarantee worth not leaning on.

## Fix

Truncate on a rune boundary:

```go
func truncateRunes(s string, max int) string {
    runes := []rune(s)
    if len(runes) > max {
        runes = runes[:max]
    }
    return string(runes)
}

newName = truncateRunes(newName, 500)
newEmail = truncateRunes(newEmail, 500)
```

Or enforce at the database layer with a `varchar(500)` column and a `CHECK (char_length(name) <= 500)` constraint so the limit lives in exactly one place.
