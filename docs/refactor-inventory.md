# Refactor Inventory & Migration Source of Truth

Status: Phase 0 (Audit & safety net). This document is the living source of
truth for the refactor; update it as work proceeds.

Conventions:
- "as-built" = behavior verified by reading the current code, not the PRD prose.
- Where the PRD or the refactor brief disagrees with the code, the code wins for
  the purpose of characterization, and the disagreement is logged in
  Section 7 (Discrepancies & decisions needed).

---

## 1. Backend HTTP surface (as-built)

Router: stdlib `net/http` method-aware `ServeMux` (Go 1.22+), built in
`internal/api/api.go` `(*API).Handler()`, wrapped by `withCORS`. CORS is
`*` for origin, methods `GET, POST, PUT, DELETE, OPTIONS`, headers
`Content-Type, Authorization`; `OPTIONS` short-circuits with 204.

| Method | Path | Handler | Auth (server-enforced) | Masking | Broadcasts | Notes |
|---|---|---|---|---|---|---|
| GET | `/api/servers` | `listServers` | none | masks `ip_address` unless valid Bearer | no | |
| GET | `/api/servers/{id}` | `getServer` | none | masks `ip_address` unless valid Bearer | no | 404 `Server not found` |
| POST | `/api/servers/update` | `updateServer` | none (allow-list gate) | n/a | yes | ingest; see Section 3 |
| DELETE | `/api/servers/{id}` | `deleteServer` | **none** | n/a | yes | deletes server row + allow-list entry |
| PUT | `/api/servers/{id}/status` | `setStatus` | **none** | unmasked sv returned | yes | body `{status}`; validates enum |
| PUT | `/api/servers/{id}/order` | `setOrder` | **none** | n/a | yes | body `{order_index:int}` (required) |
| POST | `/api/servers/{id}/heartbeat` | `heartbeat` | **none** | n/a | yes | sets running unless maintenance |
| GET | `/api/clients` | `listClients` | **none** | n/a | no | |
| POST | `/api/clients` | `addClient` | **none** | n/a | yes | body `{name}`; 400 if exists |
| GET | `/api/auth/status` | `authStatus` | none | n/a | no | `{initialized:bool}` |
| POST | `/api/auth/initialize` | `initialize` | none (only if uninit) | n/a | no | `{password}` -> `{success:true}` |
| POST | `/api/auth/login` | `login` | none | n/a | no | `{password}` -> `{token}` |
| POST | `/api/auth/reset-password` | `resetPassword` | password (old pw) | n/a | no | `{oldPassword,newPassword}` |
| GET | `/api/ws/dashboard` | `dashboardWS` | none | always masks `ip_address` | n/a | sends snapshot on connect, then registers |
| GET | `/healthz` | inline | none | n/a | no | `{status:"ok"}` |
| GET | `/download/<file>` | `http.FileServer` | none | n/a | n/a | only mounted if `AGENTS_DIR != ""` |

Critical as-built fact: **only `listServers`/`getServer` consult auth, and only
to decide masking. Every mutating admin route (delete, status, order, heartbeat,
add-client) is unauthenticated at the server.** Access control today is the
tailnet, not the API. See Discrepancy D3.

Response envelope today: success bodies are bare JSON (`{status:"success"}`,
`{token}`, a `Server`, or `[]Server`). Errors are `{"error": "<message>"}` with
the appropriate status code (see `internal/api/http.go`). The brief's target
envelope is `{"error":{"code","message"}}` — a shape change to be introduced
behind `/api/v1` while legacy paths keep the flat shape (Discrepancy D4).

---

## 2. Ingest payload contract (agent -> hub)

The agent POSTs `metrics.Collector.Sample()` (a `models.Server`) to
`POST <server>/api/servers/update`. The JSON shape is the `models.Server`
struct in `internal/models/server.go`. This is the **frozen wire contract** —
deployed agents cannot be updated atomically.

Fields sent by the agent (all snake_case JSON):
`id, name, type, location, ip_address, hostname, tailscale_ip, uptime,
network_in, network_out, cpu, memory, disk, os_type, cpu_info, total_memory,
total_disk`. (Agent does not send `status`, `order_index`, `first_seen`,
`last_update`; the hub sets those.)

Server-side handling of the report (`store.UpdateMetrics`):
- Matched by `name` (not `id`). Agent's `id` is ignored on ingest.
- Row must already exist (created by `AddClient`); ingest only UPDATEs, never
  INSERTs. No matching row -> `changed=false` -> HTTP 404 `Server not found`.
- Empty-string fields are backfilled with defaults via `nz()`
  (`Unknown`/`UN`/`127.0.0.1`/`N/A`).
- Always sets `status='running'`, refreshes `last_update` (UTC,
  `2006-01-02 15:04:05`).

Agent details (`cmd/agent`, `internal/metrics`): default interval 2s;
endpoint `--server` (or `MONITOR_SERVER`) + `/api/servers/update`; 3 retries
with 1s backoff; 8s HTTP client timeout. Static fields (location, public IP,
CPU model, OS, tailscale IP, hostname) cached once at startup. External lookups:
`api.ipify.org` (public IP), `ip-api.com` (country), `tailscale ip -4` with a
100.64.0.0/10 interface-scan fallback (`internal/metrics/{metrics,geo,tailscale}.go`).

Heartbeat path `POST /api/servers/{id}/heartbeat` exists in the router and is
listed in the PRD but is **not called by the current agent** — keep it working
for compatibility regardless.

---

## 3. Allow-list & status lifecycle (as-built)

- Allow-list table `allowed_clients(name unique)`. `IsClientAllowed` gates
  ingest: unknown name -> 403 `Client not allowed`. Empty name -> 400
  `Invalid data`.
- `AddClient(name)` is destructive-idempotent: deletes any prior allow-list and
  server rows for that name, re-inserts the allow-list entry, and inserts a
  fresh server row keyed `id = md5(name)` in `maintenance` state with
  `location='Pending'`, `type='VPS'`, zeros elsewhere.
- Status enum (`internal/models`): `running`, `stopped`, `maintenance`.
  Frontend renders `maintenance` as "Pending".
- Lifecycle: registered -> `maintenance`; first accepted report -> `running`;
  silence > `StaleAfterSeconds` (30) -> `stopped`. Sweep goroutine
  (`internal/monitor`) ticks every 15s, flips `running && last_update < cutoff`
  to `stopped`, broadcasts if any changed. Cutoff comparison is lexicographic on
  the TEXT timestamp (works because the format is sortable).
- On startup `store.init()` resets any `running` rows to `stopped` (agents must
  re-report).
- `Heartbeat(id)` sets `running` + refreshes `last_update` unless status is
  `maintenance`.
- `DeleteServer(id)` removes the server row and its allow-list entry in a tx.

---

## 4. Persistence (as-built)

SQLite via `modernc.org/sqlite` (pure Go, no cgo). `db.SetMaxOpenConns(1)` to
serialize writes. Schema created in `store.init()`; "migration" today is the
ad-hoc `store.migrate()` that ADD COLUMNs `hostname`/`tailscale_ip` if absent.
There is **no `schema_migrations` table yet** — the brief's versioned migration
list is new work (Phase 1).

Tables:
- `servers` — one row per machine, latest snapshot only (no history). Columns:
  `id, name, type, location, ip_address, hostname, tailscale_ip, status,
  uptime, network_in, network_out, cpu, memory, disk, os_type, cpu_info,
  total_memory, total_disk, order_index, first_seen, last_update`.
- `allowed_clients(id, name unique, created_at)` + `idx_client_name`.
- `admin_auth(id, password_hash BLOB, is_initialized)`.

Order: `ListServers` orders `order_index DESC, first_seen ASC`.

New tables required by Phase 2 (not present): `metrics_samples`,
`unknown_agents`, `alerts`, `schema_migrations`.

---

## 5. Auth (as-built)

`internal/auth`: bcrypt (`DefaultCost`) for the single admin password; JWT
HS256 with `ExpiresAt = now+24h`, `IssuedAt = now`, **no custom claims** (no
subject/role). `Valid` enforces HMAC method + signature + expiry. Secret from
`SECRET_KEY` env or persisted `data/.secret` (32 random bytes hex). Bearer is
read in `(*API).isAuthed` (`Authorization: Bearer <jwt>`, case-insensitive
scheme).

---

## 6. Configuration (as-built)

`internal/config.Load()`:

| Env var | Default | Notes |
|---|---|---|
| `ADDR` | `0.0.0.0:5000` | |
| `DATA_DIR` | `./data` | created `0755`; holds db + `.secret` |
| `DATABASE_PATH` | `<DATA_DIR>/servers.db` | |
| `AGENTS_DIR` | `./dist` | `""` disables `/download` |
| `SECRET_KEY` | auto, persisted to `data/.secret` | |

`StaleAfterSeconds` is **hardcoded to 30** in `Load()` — not currently an env
var (the brief's env contract does not require it to be; noted so Phase 1 can
add one with a 30s default without changing behavior).

`http.Server` timeouts: `ReadTimeout 15s`, `WriteTimeout 0` (for long-lived
WS), `IdleTimeout 60s`. Graceful shutdown on SIGINT/SIGTERM with a 5s timeout;
the sweep goroutine is cancelled via context. (Note: WS hub is not explicitly
drained on shutdown today — brief asks for explicit drain in Phase 1.)

---

## 7. Frontend surface (as-built)

Stack reality (verified, differs from PRD/brief — see D1):
- Next.js **14.0.4, Pages Router** (`frontend/pages/`), TypeScript.
- Styling is **Tailwind only**. `@chakra-ui/react`, `@emotion/react`,
  `@emotion/styled`, `framer-motion`, `react-beautiful-dnd`, `jsonwebtoken` are
  in `package.json` but **imported nowhere** (verified by ripgrep). Removing
  Chakra is therefore a dependency-cleanup, not a code-migration.
- Theme: `next-themes` (`attribute="class"`, `defaultTheme="system"`).
- Icons: `@heroicons/react` + FontAwesome/flag-icons CDN in `_document.tsx`.
- Toasts: `react-hot-toast`.

Routes:
- `/` (`pages/index.tsx`) — public dashboard; WS subscription + 5s REST poll
  fallback; desktop table (`ServerList`/`ServerCard`) and mobile cards
  (`MobileServerCard`); summary stats.
- `/login` (`pages/login.tsx`) — dual-mode init/login via `/api/auth/status`.
- `/admin` (`pages/admin/index.tsx`) — `ProtectedRoute`-gated; add client,
  client table, reset password, delete, set order.
- `/api/hello` — unused Next stub.

API client (`frontend/utils/api.ts`, `frontend/config/config.ts`):
- Base URL: `NEXT_PUBLIC_API_URL` else `${protocol}//${hostname}:5000` else
  `http://localhost:5000`. WS URL = base with `http`->`ws` + `/api/ws/dashboard`.
- Token in `localStorage['adminToken']`; `fetchWithAuth` adds
  `Authorization: Bearer`.
- Calls: `POST /api/auth/login`, `GET /api/auth/status`,
  `POST /api/auth/initialize`, `POST /api/auth/reset-password` (auth),
  `GET /api/servers`, `DELETE /api/servers/{id}` (auth),
  `PUT /api/servers/{id}/order` (**no auth header sent**),
  `POST /api/clients` (**no auth header sent**), WS `/api/ws/dashboard`.

Ordering UI: an inline **numeric `order_index` editor** in
`components/admin/ClientTable.tsx` (`setEditingOrder`), **not** drag-and-drop.
The PRD's "drag-and-drop ordering" and the brief's "drag-and-drop ordering
preserved" describe a feature that does not exist today (D2).

`frontend/types/server.ts` mirrors `models.Server` (snake_case) plus a UI-only
`is_expanded?`. `frontend/types/admin.ts` has `AdminStats`/`ClientAction`.

---

## 8. Distribution & deploy (as-built)

- `scripts/build.sh`: `CGO_ENABLED=0 -trimpath -ldflags "-s -w"`. Targets:
  server linux/{amd64,arm64}; agent linux/{amd64,arm64}, windows/amd64,
  darwin/{amd64,arm64}. Output `dist/`.
- `/download/<file>` serves `dist/` so tailnet hosts self-install.
- `scripts/install_agent.sh <name> <hub-url>` (systemd `server-monitor-agent`),
  `scripts/install_agent.ps1` (Windows SCM `ServerMonitorAgent`). Copies also
  exist in `dist/`.
- `deploy/monitor-server.service`: `WorkingDirectory=/opt/server-monitor`,
  `ADDR=0.0.0.0:5000`, `DATA_DIR`/`AGENTS_DIR` under `/opt/server-monitor`,
  `DynamicUser=yes`, `Restart=always`. `deploy/monitor-frontend.service` runs
  the dashboard.
- go.mod: `go 1.26`; deps gorilla/websocket, golang-jwt/v5, gopsutil/v4,
  x/crypto, modernc.org/sqlite.

Files referenced by deploy/docs (do **not** delete without asking, per working
agreements): `scripts/build.sh`, `scripts/install_agent.sh`,
`scripts/install_agent.ps1`, `deploy/*.service`, the `dist/` binaries,
`docs/add-machine.md`.

---

## 9. Test baseline

**There are zero existing tests** in the repo (`go` and frontend). Every
characterization test in Phase 0 is net-new. This raises the value of writing
them before any move in Phase 1.

---

## 10. Discrepancies & decisions needed

These are surfaced for review (working agreement: stop and present when a
decision trades off against a Section 2 constraint).

**Resolutions (2026-06-12):** D5 -> also mask `tailscale_ip` for anonymous
viewers (deliberate behavior change, scheduled as a labeled feature change, not
folded into the Phase 1 "no behavior change" move). D3 -> `/api/v1` mutations
require JWT; legacy `/api/*` stays open; frontend updated to always send the
token. D2 -> build real drag-and-drop in Phase 4. D1/D4 -> proceed as
recommended.

Characterization-test consequence of D5: Phase 0 tests assert **current**
behavior (only `ip_address` masked, `tailscale_ip` visible) so the Phase 1
restructure is provably behavior-preserving. The tailscale-masking change then
lands as its own commit that intentionally flips that one assertion.

- **D1 — "Chakra + Tailwind mix" is not real.** As-built frontend is
  Tailwind-only; Chakra/emotion/framer/dnd deps are dead. Impact: the "remove
  Chakra" acceptance criterion is satisfied by deleting dependencies; no
  component rewrite is forced by Chakra. The real frontend work is the new
  AppShell/design-token/shadcn rebuild, not de-Chakra-ing. Recommend: proceed,
  treat "zero Chakra imports" as already met and keep it met.

- **D2 — Drag-and-drop ordering does not exist.** Today it is a numeric
  `order_index` editor. The brief says ordering must be "preserved." Decision:
  preserve the numeric-editor behavior, or build real DnD as part of Phase 4?
  Recommend: build real DnD in Phase 4 as an enhancement (the `PUT .../order`
  contract already supports it), but this is net-new, not preservation.

- **D3 — Admin mutations are unauthenticated server-side.** Delete, set-status,
  set-order, add-client, heartbeat enforce no auth; the frontend doesn't even
  send a token on order/add-client. Section 4 of the brief wants auth
  middleware. **Adding auth enforcement to these routes is a behavior change**
  (would 401 the current frontend's order/add-client calls and any tailnet
  script). Options: (a) keep them open to preserve exact behavior (tailnet is
  the boundary), add middleware only as opt-in; (b) enforce auth on mutations
  under `/api/v1` and update the frontend to always send the token, while legacy
  `/api/*` stays open; (c) enforce everywhere and update the frontend. Recommend
  (b): tighten the v1 surface, keep legacy compatible. Needs your call before
  Phase 1 wiring.

- **D4 — Error envelope shape change.** Legacy is `{"error":"msg"}`; brief's
  target is `{"error":{"code","message"}}`. Plan: legacy paths keep the flat
  shape (characterization-locked); the new envelope applies only to `/api/v1`.
  Confirm acceptable.

  **Phase 1 status:** to stay strictly behavior-preserving, the error mapper
  currently renders the flat shape on *both* surfaces. The nested v1 envelope is
  deferred to land alongside the v1-consuming frontend (Phase 4), so no consumer
  sees a shape change mid-flight. (The v1 RequireAuth 401 already uses the nested
  shape, since that path has no legacy consumer.)

Deferred deliberate behavior changes (tracked, not yet applied):
- **D5 tailscale masking** — `internal/masking` masks only `ip_address` today
  (behavior-preserving). Flipping it to also mask `tailscale_ip` for anonymous
  viewers is a one-edit change in that package + its characterization assertion,
  to be committed as a labeled change.
- **WS auth state** — dashboard sockets are still always-masked (no behavior
  change). Per-connection auth-aware masking (admin WS sees real IPs live) lands
  with the auth-aware WS handshake in a later phase.

- **D5 — Tailscale IP masking conflict (Section 2 #3 vs as-built).** The brief's
  hard constraint says anonymous viewers must never see **public IPs or
  Tailscale IPs**, and also says keep masking semantics "exactly." As-built,
  **only `ip_address` is masked; `tailscale_ip` (and `hostname`) are never
  masked** — `models.Server` documents this as intentional. These two
  requirements conflict. Options: (a) preserve exactly (mask only `ip_address`);
  (b) honor the stated invariant and also mask `tailscale_ip` for anonymous
  viewers (a behavior change, and arguably a security improvement). Recommend
  (b) and lock it with a characterization test asserting the *new* desired
  masking — but this is explicitly a behavior change and needs your sign-off,
  since it contradicts "keep masking semantics exactly."

---

## 11. Migration progress tracker

| Phase | Scope | Status |
|---|---|---|
| 0 | Audit + characterization tests | DONE: inventory written; 3 test files green under `go vet` + `go test -race` (`internal/api`, `internal/store`, `internal/models`). Awaiting go-ahead for Phase 1. |
| 1 | Backend restructure (no behavior change) | DONE: domain / storage(sqlite+versioned migrations) / service(servers,auth) / transport(http,ws,middleware) / masking / config(slog, graceful drain, STALE_AFTER_SECONDS). /api/v1 aliases live; legacy preserved. Verified: race tests, build.sh targets, runtime smoke, real servers.db upgraded in place. |
| 2 | Backend features (B1 metrics, B2 observability, B3 alerts, B4 v1) | DONE. B1: metrics_samples + 5m rollups, history/summary endpoints, compaction (migration v3). B2: unknown_agents table + admin endpoint (v4). B3: alerts table (v5), status/threshold emission, webhook notifier, list/ack endpoints. B4: v1 canonical surface documented in backend README. Each with fake-clock/fake-repo + handler tests. |
| 3 | Frontend foundation (shell, tokens, client, WS, Query) | not started |
| 4 | Page rebuilds (Dashboard, Servers, Alerts, Admin) | not started |
| 5 | Hardening & docs | not started |
