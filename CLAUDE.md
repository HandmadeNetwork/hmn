# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is the [Handmade Network](https://handmade.network) website — a full-stack Go web application with PostgreSQL. It hosts forums, blogs, game jams, podcasts, project pages, and event ticketing, with integrations for Discord, Twitch, and Stripe.

## Development Setup

1. Copy the config: `cp src/config/config.go.example src/config/config.go`
2. Add to your hosts file (`C:\Windows\System32\Drivers\etc\hosts` on Windows):
   ```
   127.0.0.1 handmade.local hero.handmade.local 4coder.handmade.local
   ```
3. Initialize the database: `go run . db seed --create-user`
4. Run the server: `go run .` — site runs at `http://handmade.local:9001`

Seed users (all with password `password`): `admin` (superuser), `alice`, `bob`, `charlie`.

## Common Commands

```bash
go run .                    # Run the website
go run . help               # List all CLI subcommands
go run . db seed            # Re-seed the database
go run . buildcss           # Rebuild CSS via esbuild
go test ./...               # Run all tests
go test ./src/hmnurl/...    # Run URL tests (100% coverage required for Build* functions)
```

## Architecture

**Entry point:** `main.go` → `src/website/` package handles everything.

**Key packages:**

- `src/website/` — All HTTP handlers (~14K lines). Custom regex-based router, no external router library. Middleware for auth, CSRF, and performance tracking.
- `src/hmnurl/` — URL building functions and routing regex definitions. All routes are defined here and must have 100% test coverage on `Build*` functions.
- `src/db/` — Custom query builder wrapping `pgx/v5`. Uses placeholder syntax like `$columns`, `$columns{prefix}` to reduce boilerplate. Migrations live in `src/migration/`.
- `src/models/` — Data type definitions (User, Project, Post, Thread, Jam, Podcast, Ticket, Asset, etc.).
- `src/auth/` — Session-based auth, Discord OAuth, password reset flows.
- `src/config/` — Environment config (Dev/Beta/Live). Connection details for Postgres, email, Discord, Twitch, DigitalOcean Spaces, Stripe.
- `src/parsing/` — Content pipeline: custom BBCode fork, goldmark Markdown fork, Chroma syntax highlighting, GGCode embedded code, spoilers, MathJax.
- `src/assets/` — S3-compatible storage (DigitalOcean Spaces in production, fake S3 server locally). SHA1 integrity checks.
- `src/discord/` — Discord bot and OAuth integration.
- `src/twitch/` — Twitch subscription monitoring.
- `src/jobs/` — Background job queue (session cleanup, Discord bot, Twitch, email bounces, asset preview generation).
- `src/templates/` — Go template system. Set `DevConfig.LiveTemplates = true` in config for live reloading during development.
- `src/admintools/` — Moderation and site management UI.
- `src/perf/` — Performance monitoring (pprof available at `localhost:9002/debug/pprof/`).
- `src/oops/` — Error handling with stack traces.

**Multi-project routing:** The site serves both official projects (e.g., `hero.handmade.local`) and personal projects under subdomains. The `hmnurl` package centralizes all URL construction and regex matching for this.

**Static files:** `public/` directory. CSS source lives in `src/templates/src/` and is compiled by `buildcss`.
