# Bug Audit

Heuristic audit of the hmn codebase (~50k Go LoC). Each package rated on two axes:

- **Likelihood** — how likely the package is to contain latent bugs, based on complexity, surface area, mutable state, and amount of hand-rolled logic vs. well-tested third-party code.
- **Severity** — the expected blast radius of a bug that does exist in the package (data loss, auth bypass, XSS, RCE, DoS, wrong page rendered, etc.).

Rating scale: **Low / Medium / High / Critical**.

Spot-check findings are listed per-package. This is a heuristic review, not an exhaustive one — anything listed as "likely" is worth a focused look, not a guaranteed bug.

---

## Summary table

| Package | Likelihood | Severity | Notes |
|---|---|---|---|
| `src/auth/` | Medium | **Critical** | Password verify is non-constant-time; silent rand.Reader error; parser panics on malformed config |
| `src/website/` (handlers) | **High** | **High** | ~14k LoC, lots of hand-rolled auth/CSRF/redirect logic. Open redirect, IP spoofing, nil derefs, discarded return values confirmed |
| `src/website/requesthandling.go` | Medium | High | Trusts client-supplied proxy headers unconditionally; potential open-redirect in Redirect helper |
| `src/website/tickets.go` | Medium | High | Nil deref on CurrentUser, lost Redirect return, unsafe string slicing, payment flow is safety-critical |
| `src/website/stripe.go` | Low | **Critical** | Small file, signature-verified — but any bug here = money |
| `src/db/` | Medium | High | Heavy reflection + custom `$columns` mini-DSL. Panics in hot paths, fragile iterator lifecycle |
| `src/parsing/` | **High** | **High** | Custom BBCode + Markdown fork + GGCode + spoilers. Classic XSS/injection surface |
| `src/assets/` | Medium | High | `exec.Command` arg splitting on spaces, response-body leak in background job, no content-type validation |
| `src/discord/` | Medium | Medium | Long-lived websocket state machine, goroutine lifecycle, shared mutable slice returned under lock |
| `src/twitch/` | Medium | Medium | Similar profile — long-running subscription loop, stateful |
| `src/jobs/` | Low | Medium | Small, focused; primary risk is misuse by callers |
| `src/hmnurl/` | Low | High | Central URL builder, 100% test coverage on Build*; bugs here would misroute everything |
| `src/hmndata/` | Medium | Medium | Query helpers — bugs tend to be wrong-rows-returned, not security |
| `src/email/` | Low | Medium | Mostly SMTP + templates; bounce handling has moderate complexity |
| `src/admintools/` | Low | High | Staff-only, so fewer hostile inputs, but mistakes affect all data |
| `src/migration/` | Low | **Critical** | Run-once; a bug here corrupts production on deploy |
| `src/models/` | Low | Low | Mostly type declarations |
| `src/templates/` | Low | Medium | Template loader + helpers; bugs usually render-time panics |
| `src/perf/`, `src/oops/`, `src/logging/`, `src/config/`, `src/utils/` | Low | Low | Small utility packages |
| `src/calendar/`, `src/links/`, `src/embed/` | Low | Low | Small, narrow scope |

---

## High-priority packages

### `src/auth/` — Medium / Critical

Small (~480 LoC) but safety-critical. Hand-rolled hash handling.

Concrete findings in `auth.go`:

- `auth.go:128` and `auth.go:149` — `bytes.Equal` used to compare password hashes. Not documented as constant-time; should be `crypto/subtle.ConstantTimeCompare`. Timing-attack surface is small (hash is deterministic per salt), but this is a standard fix.
- `auth.go:160` — `io.ReadFull(rand.Reader, salt)` drops the error. If `rand.Reader` ever fails, the salt is all-zero and you silently ship a broken hash. Cheap to fix: panic or return error.
- `auth.go:80-95` — `ParseArgon2idConfig` indexes `parts[0][2:]`, `parts[1][2:]`, etc. with no length check on `parts` or the substrings. Malformed DB rows → panic during login.
- `session.go:33,43` — `base64(..)[:40]` / `[:30]` truncates base64 output. Effective entropy is still high enough (30 base64 chars ≈ 180 bits), so not a security issue, just an odd idiom.
- `session.go:118-129` — CSRF comparison `csrfToken != c.CurrentSession.CSRFToken` is not constant-time (see website/middlewares).

### `src/website/` — High / High

By far the largest package (~14k LoC, 46 files). The most likely place to find bugs and the most likely place for those bugs to matter.

`requesthandling.go`:

- `FullUrl()` `requesthandling.go:234-252` — trusts `X-Forwarded-Proto` from the client unconditionally. The `var scheme string; if scheme == ""` pattern is dead-code-shaped and suggests this was meant to be gated behind a trusted-proxy check. FullUrl is used in email links and the login redirect flow, so a forged header can influence those.
- `GetIP()` `requesthandling.go:257-298` — same pattern: `Cf-Connecting-Ip` and `X-Forwarded-For` are trusted from any client. If the host is ever reached directly (not via Cloudflare), IP spoofing bypasses rate limits / audit logs. Also, `netip.ParsePrefix("%s/32", ipString)` always builds a `/32` even for IPv6, giving a nonsense prefix.
- `Redirect()` `requesthandling.go:311-368` — the "is this a local path" check is `u.Scheme == "" && u.Host == ""`. A protocol-relative URL like `//evil.com/foo` parses with `Host == "evil.com"`, so the guard passes the value through unchanged → open redirect if any caller passes user-controlled `dest`. Callers combine with `urlIsLocal` (see below), which is also broken.

`middlewares.go`:

- `csrfMiddleware` `middlewares.go:112-133` — CSRF token equality is a plain string compare, not constant-time. Low risk but trivial to fix.
- `csrfMiddleware` `middlewares.go:116` — calls `ParseMultipartForm(100 * 1024 * 1024)` and ignores the error. 100 MiB memory budget per request is a DoS knob for authenticated endpoints.

`auth.go` (handlers):

- `urlIsLocal` `auth.go:972-979` — uses `strings.HasSuffix(urlParsed.Host, baseUrl.Host)`. If `baseUrl.Host = "handmade.network"`, then `evilhandmade.network` passes. **Open redirect.** Combine with login-redirect flow to phish credentials. Use exact host match, or match on a leading-dot boundary.
- `validateUsernameAndToken` `auth.go:965` — token compared with `==`, not constant-time. Tokens are long random strings so practical risk is bounded, but this is the one place constant-time really matters.
- `RegisterNewUser` `auth.go:180` — `c.Redirect(...)` is called but its return value is discarded. The function then falls through and re-renders the register page for users who are already signed in. Functional bug, not a security issue.
- `RegisterNewUserSubmit` `auth.go:214` — `c.Req.ParseForm()` error ignored.

`tickets.go`:

- `TicketScanned` `tickets.go:738-745` — `if !c.CurrentUser.IsStaff { c.Redirect(...) }`. Two bugs stacked: the redirect's return value is discarded (control falls through), and `c.CurrentUser` is dereferenced without a nil check. Currently the fallthrough body is a stub, but when scanning logic lands this is a hostile-input nil panic and an auth bypass.
- `canEditTicket` `tickets.go:747-753` — `user.IsStaff` dereferences `user` without nil check. Called from `TicketEdit` / `TicketEditSubmit` which appear to be gated by `needsAuth`, so currently safe, but the function is a landmine.
- `TicketEditSubmit` `tickets.go:630-631` — `newName[:min(len(newName), 500)]` slices bytes, will split a multi-byte UTF-8 character and produce invalid strings stored in the DB. Use rune iteration.
- `TicketPurchase` `tickets.go:345` — the deferred "cleanup pending ticket" Exec uses `c` as context, which may already be canceled by the time the handler returns. Cleanup can silently fail under cancellation.

`stripe.go`:

- 99 LoC, signature verification via the Stripe SDK, looks tight. Severity is Critical because this is the money path; likelihood Low. Worth a second pair of eyes on `confirmStripeTicketPurchase` which blindly trusts `session.PaymentIntent.ID` / `AmountTotal` at `tickets.go:418` — no check that `AmountTotal` matches the configured price. A misconfigured Stripe event could confirm a ticket for the wrong amount.

### `src/db/` — Medium / High

Custom query builder over pgx, heavy reflection, generic iterator.

- `db.go:447-527` `Iterator.Next` panics on essentially any reflection mismatch. Callers in `website/` use the non-`Must*` variants, which catches errors — but scattered `MustQuery*` calls will crash the process on schema drift.
- `db.go:219-260` — the iterator lifecycle launches a goroutine per query to watch for context cancellation, and relies on `it.Close()` being called from either the context or `ToSlice`/`Next`'s terminal path. `QueryOne` `db.go:76-98` calls `defer rows.Close()` (correct), but any caller that only does `rows.Next()` once and forgets to close leaks a pgx connection until the request context dies. High leakage risk for long-running requests.
- `db.go:597-626` `followPathThroughStructs` mutates the destination struct (allocates intermediate pointer fields) before assigning — if a panic fires mid-row, the partially-populated struct is returned to callers that caught the panic via `recover` in middleware. Subtle.
- `db.go:286-332` `$columns` regex is a simple `\$columns({(.*?)})?`. Fine, but consider: a SQL string containing `$columns` inside a string literal (e.g. `WHERE note = '$columns'`) would be rewritten. Unlikely in practice, low severity.
- `db.go:21-26` shared `pgTypeMap` initialized in `init`, used read-only at runtime. OK.

### `src/parsing/` — High / High

The highest-XSS-risk package. Hand-rolled extensions, raw HTML enabled for education, custom BBCode fork.

- `parsing.go:80,91` — `EducationPreviewMarkdown` / `EducationRealMarkdown` both set `html.WithUnsafe()`. This permits raw HTML in markdown source → any education content author can emit arbitrary `<script>`. Relies entirely on trust in the author role. Acceptable if enforced at the handler/middleware layer (`educationAuthorsOnly`), but worth an explicit comment where the markdown variant is declared.
- `bbcode.go:82-109` — `[quote=foo]` puts the raw tag value into the `cite` attribute and into `href` of the quotewho link via `hmnurl.BuildUserProfile(cite)`. The `frustra/bbcode` library escapes attribute values, so XSS is unlikely, but confirm — and `BuildUserProfile` should reject usernames that contain URL-dangerous characters.
- `bbcode.go:245-300` `bbcodeParser.Parse` — unbounded nested-tag scan. Pathological input like `[x][x][x]...` recurses via goldmark + the library; worst case is quadratic in source length. DoS-leaning; treat as low.
- `ggcode.go`, `spoilers.go`, `embed.go`, `mathjax.go` — each adds a goldmark extension with custom rendering. Each custom renderer is an XSS vector if any of them stringifies user input into an HTML attribute without escaping. Worth a pass with fresh eyes.
- `chroma.go` / syntax highlighting — Chroma is well-tested; low risk.

### `src/assets/` — Medium / High

File upload + S3 + FFmpeg shell-out.

- `assets.go:236-241` — FFmpeg args built via `fmt.Sprintf` and then `strings.Split(args, " ")`. Temp filename is `os.CreateTemp` output, so no attacker injection today, but `strings.Split` on a space is fragile on any platform whose temp dir contains spaces. Rewrite as a pre-built `[]string{"-i", file.Name(), ...}`.
- `assets.go:315` — `defer resp.Body.Close()` is inside a `for _, asset := range assets` loop. Deferred closes don't run until the whole goroutine exits, so a large backlog of assets keeps a growing pile of unclosed response bodies open. Use an inline close or wrap the body in a func.
- `assets.go:311` — when `resp.StatusCode != 200`, the body is never closed at all (no defer hits that path cleanly). Same leak, worse.
- `assets.go:85-195` `Create` — no validation that `in.ContentType` matches `in.Content`. A client that can influence the content type can upload a file labeled `text/html` and have it served with that MIME by S3/DO. XSS via asset URL. Whether this is exploitable depends on which caller paths let users pick their own content type.
- `assets.go:98` — SHA1 for integrity. Fine for dedupe/integrity, not a security property; flag for the record.
- `assets.go:114` — auto-creates the S3 bucket on `NoSuchBucket`. Harmless in practice but worth noting for prod vs. staging credential safety.

### `src/discord/` — Medium / Medium

Largest non-website package. Long-running websocket state machine, OAuth, rate limiter.

- `gateway.go:48-52` `GetBotEvents` returns `botEvents[:]` while holding the mutex, then releases the mutex. Caller and subsequent `RecordBotEvent` writer share the same backing array. `RecordBotEvent` does `botEvents = botEvents[len(botEvents)-500:]` (`gateway.go:38-40`) and `append`. Depending on cap this will mutate indices the caller is reading → data race. Return a copy.
- `gateway.go:115` — reconnect jitter uses `math/rand.Int63n` without explicit seed. Fine functionally.
- `rest.go`, `ratelimiting.go`, `message_handling.go` — long files, lots of API state. Didn't spot-audit but the Discord layer is classic "works until Discord sends you something unexpected" territory.
- `markdown.go` / `payloads.go` have tests; that's a positive signal.

### `src/twitch/` — Medium / Medium

Similar profile to Discord: long-running subscription / webhook handler. Not audited in detail; flag as medium on priors.

### `src/hmnurl/` — Low / High

Central URL constructor. `Build*` functions are required to have 100% test coverage (per CLAUDE.md) and the test file exists. Low likelihood. Severity is High because every route passes through here — a bug misroutes the whole site.

### `src/migration/` — Low / Critical

Each migration runs once. A bug that passes code review and hits production corrupts data irreversibly. The `Initial.go` is 4.6k LoC which is unusual but expected for a port. Treat all migrations as code that must be reviewed with a higher bar than runtime code.

---

## Lower-priority packages

- **`src/hmndata/`** — query helpers on top of `db`. Bugs tend to be "query returns wrong rows for edge-case filter combinations" rather than security issues. Medium likelihood just because of volume, Medium severity.
- **`src/admintools/`** — staff-only, narrower attack surface, but mistakes can affect all data at once.
- **`src/models/`** — mostly type declarations and small helpers. Low risk.
- **`src/templates/`** — template loader + mapping; ~575 LoC in `mapping.go`. Risk is runtime panics during template execution.
- **`src/email/`** — ~3 files, modest. `preprocessor.go` has tests.
- **`src/jobs/`** — 110 LoC, tested. Risk comes from misuse at call sites, not this package.
- **`src/config/`, `src/logging/`, `src/oops/`, `src/perf/`, `src/utils/`, `src/calendar/`, `src/links/`, `src/embed/`, `src/ansicolor/`, `src/buildcss/`, `src/initimage/`, `src/hmns3/`, `src/rawdata/`** — small, narrow, or infrastructure. Low likelihood, low/medium severity.

---

## Recommended next steps

In rough priority order:

1. Fix `urlIsLocal` in `website/auth.go:972` and the `Redirect` open-redirect in `website/requesthandling.go:311`. These are real exploitable bugs today.
2. Gate `FullUrl()` / `GetIP()` proxy-header trust behind a config flag that names the trusted proxy.
3. Fix the `TicketScanned` / `canEditTicket` nil-deref landmines in `website/tickets.go:738`.
4. Fix the `RegisterNewUser` discarded `c.Redirect(...)` return at `website/auth.go:180` (and grep for similar discarded Redirect returns — there are probably more).
5. Close the per-iteration `resp.Body` leak in `assets.go:310-320`.
6. Replace `bytes.Equal` / `==` on password hashes and tokens with `crypto/subtle.ConstantTimeCompare` in `auth/auth.go` and `website/auth.go:965`.
7. Audit every goldmark extension in `src/parsing/` for attribute-context escaping. Education markdown's `WithUnsafe()` should have a comment tying it to the author-only trust boundary.
8. Confirm `stripe.go` / `confirmStripeTicketPurchase` validates `AmountTotal` against the configured price before marking a ticket paid.
9. Return a copy from `discord.GetBotEvents()` to kill the data race.
10. Add length checks to `auth.ParseArgon2idConfig` before slicing `parts[N][2:]`.
