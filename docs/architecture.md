# Architecture

This document describes how the codebase is organized after the refactor:
the Go backend's layering and the Next.js frontend's conventions. For the
shipped product behavior see `PRD.md`; for migration history see
`refactor-inventory.md`.

## Backend (`server-monitor-go`)

A layered, interface-driven architecture. Dependencies point inward only:
**transport -> service -> storage interface -> sqlite**; `domain` depends on
nothing. `cmd/server/main.go` builds the graph explicitly with constructor
injection — no globals, no `init()` side effects, no service locators.

```
internal/
  domain/        Core types (Server, Status, Alert, MetricSample, Client,
                 UnknownAgent, FleetSummary) and sentinel errors
                 (ErrNotFound, ErrNotAllowed, ErrUnauthorized, ErrConflict,
                 ErrInvalidInput). No external dependencies.
  config/        Env parsing + defaults; JWT secret persistence. One Load().
  storage/
    sqlite/      The core persistence (modernc.org/sqlite, pure Go, no cgo).
                 Owns the schema via an ordered, versioned schema_migrations
                 runner. A single Store satisfies the narrow repository
                 interfaces each service defines, because some operations
                 (AddClient, DeleteServer) span tables transactionally.
    postgres/    Optional external log store (pgx, pure Go) for high-volume
                 logs, kept off the hub's SQLite. Built only when
                 LOG_DATABASE_URL is set; see docs/logs.md.
  service/
    servers/     Lifecycle, ingest accept/reject, ordering, staleness sweep.
                 Injects a Clock (testable time) and an optional AlertSink.
    auth/        First-run init, login, password reset, token verify.
    metrics/     History writes, downsampling, fleet summary, rollup/prune
                 compaction. Injects a Clock.
    alerts/      Status/threshold alert emission + a pluggable Notifier.
    logs/        Log ingest + query over the external store (enabled only when
                 a log database is configured).
  transport/
    http/        Thin handlers (decode -> service -> encode), a route table
                 mounting legacy /api and canonical /api/v1, and one error
                 mapper. handlers/ logic is HTTP-only; business rules live in
                 service/.
      middleware/ Request IDs, slog request logging, panic recovery, CORS,
                 bearer-token auth (v1 mutations).
    ws/          gorilla/websocket fan-out hub; per-connection auth state so
                 admin sockets get unmasked frames.
  masking/       The IP-masking rule, in exactly one place.
  auth/          bcrypt + JWT (HS256) primitives.
  metrics/       Agent-side resource + geo/IP collection (gopsutil).
pkg/             (reserved) only what agent + server genuinely share.
```

### Rules

- **Interfaces are defined by the consumer.** Each service declares the
  repository interface it needs; the sqlite `Store` satisfies them structurally.
- **Errors** are domain sentinels. The single HTTP error mapper
  (`transport/http/errors.go`) translates them to status codes and the JSON
  error envelope. Legacy paths return `{"error":"message"}`; `/api/v1` returns
  `{"error":{"code","message"}}`.
- **Logging** is `log/slog` everywhere. Request middleware logs method, path,
  status, duration, and request ID. Rejected ingest reports log the offending
  name and remote address.
- **Migrations** are an ordered, versioned list with a `schema_migrations`
  table. A pre-versioning database is baselined, so an existing `servers.db`
  upgrades in place. New feature tables are additive, numbered entries.
- **Lifecycle** is context-aware: SIGINT/SIGTERM stops the sweep + compactor,
  drains the WS hub, stops accepting, and closes the DB.
- **Compatibility is non-negotiable:** deployed agents and an existing
  `servers.db` must keep working. The agent ingest DTO (`domain.Server` JSON)
  is frozen and locked by a contract test.

## Frontend (`frontend`)

Next.js App Router, TypeScript strict, Tailwind + shadcn/ui (Radix), one
styling system. Server state is TanStack Query; the WebSocket manager patches
the query cache (no parallel store). All colors resolve to CSS-variable design
tokens — components never hardcode hex.

```
src-less layout (project root is the app root):
  app/
    layout.tsx              Root: providers (theme, query, toaster)
    (dashboard)/
      layout.tsx            AppShell: Sidebar + Topbar + content + CommandPalette
      page.tsx              Dashboard
      servers/ alerts/ admin/  Feature pages
      logs/ analytics/ ...  "coming soon" placeholders
    login/page.tsx
  components/
    ui/                     shadcn primitives (button, card, badge, table, ...)
    shared/                 StatCard, StatusBadge, TrendBadge, ResourceBar,
                            PageHeader, EmptyState, CopyButton, ComingSoon
    layout/                 Sidebar, Topbar, ThemeToggle, CommandPalette
  features/
    servers/ metrics/ admin/   Feature components composed from shared/ + ui/
  lib/
    api/                    Typed client: one function per /api/v1 endpoint
    ws/                     WebSocket manager (reconnect backoff, token, cache)
    hooks/                  useServers, useMetrics, useAlerts, useAuth
    auth.ts query.tsx format.ts utils.ts
  config/nav.ts             Sidebar items as data
```

### Conventions

- **Data flow:** TanStack Query owns server state. `useServers` opens the WS,
  patches the `["servers"]` cache on push, and polls REST every 5s as a
  fallback. The WS re-handshakes (with the token) when auth changes so IP
  masking stays correct live.
- **Components** take typed props with explicit loading (skeleton) / empty /
  error states, and are composed from `shared/` + `ui/` primitives — a new page
  never re-implements a stat card or a table.
- **Auth** token lives in `localStorage` and is read per request; an
  auth-changed event drives WS re-handshake and React state.
- **Accessibility:** Radix gives keyboard nav + focus rings; status is never
  color-only (badges carry an icon + label).
