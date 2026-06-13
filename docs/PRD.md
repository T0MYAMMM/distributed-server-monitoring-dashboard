# PRD — Distributed Server Monitoring Dashboard

| | |
|---|---|
| **Status** | As-built (documents the shipped product, post-refactor) |
| **Version** | 2.0 |
| **Last updated** | 2026-06-12 |
| **Owner** | Thomas Stefen |
| **Repo** | `distributed-server-monitoring-dashboard` |

---

## 1. Overview

A self-hosted, real-time monitoring dashboard for a fleet of distributed
servers. One machine on a private **Tailscale** tailnet acts as the **hub**
(Go backend + Next.js dashboard); every monitored machine runs a tiny,
dependency-free **Go agent** that reports system metrics to the hub every few
seconds. The dashboard shows live CPU, memory, disk, network, uptime, OS,
location, and Tailscale identity for every machine, updating in real time
over WebSocket.

**Design philosophy:** zero public exposure (everything rides the tailnet),
zero runtime dependencies (static binaries, pure-Go SQLite, stdlib HTTP
router), and a two-minute onboarding path per machine.

## 2. Problem statement

People who run a handful of machines — VPSes, home servers, laptops acting as
database hosts — have no lightweight way to see the health of all of them in
one place. Existing options are either heavyweight (Prometheus + Grafana +
exporters per box), SaaS (data leaves the network, subscription cost), or
single-host (htop/Glances per machine). There is no "open one page, see every
machine, installed in two minutes" option that stays entirely on a private
network.

## 3. Goals

1. **Single pane of glass** — every machine's live vitals on one page,
   updating within ~2 seconds of change.
2. **Trivial onboarding** — register a name, run one install command on the
   target machine, see it go live (~2 minutes per machine).
3. **Private by default** — all traffic stays inside the Tailscale tailnet;
   nothing is exposed to the public internet.
4. **Zero-dependency deployment** — single static binaries for hub and agent
   (Linux amd64/arm64, Windows, macOS Intel/ARM); agents download their own
   binary from the hub.
5. **Safe to share a screen** — viewers without admin credentials never see
   real public IP addresses.

## 4. Non-goals (current version)

- Historical metrics, graphs over time, or retention (only the latest report
  per server is stored).
- Alerting / notifications (email, Slack, pager) on status change or
  thresholds.
- Per-process, per-container, or application-level monitoring (DB query
  stats, service health checks).
- Multi-user accounts, roles, or audit logs (single shared admin password).
- TLS termination (transport security is delegated to Tailscale's WireGuard
  encryption).
- Horizontal scaling of the hub (single process, single SQLite file).

## 5. Users

| Persona | Needs |
|---|---|
| **Operator/owner** (admin) | Register/remove machines, see real IPs, force statuses, reorder the list, manage the admin password. |
| **Viewer** (anyone on the tailnet) | Open the dashboard, see live health of all machines; must not see sensitive data (public IPs). |
| **Monitored machine** (agent) | Report metrics with one static binary, survive reboots, recover automatically when the hub is unreachable. |

## 6. System architecture (as built)

```
        agents (Go static binary, one per server)
  web-1 ───┐
  db-1  ───┤  POST /api/servers/update  (every 2s)
  hub   ───┘            │
                        ▼
             ┌─────────────────────────┐     WebSocket push
             │   monitor-server (Go)   │ ────────────────────►  Dashboard
             │  REST + WebSocket + DB  │     (Next.js, real-time)
             └─────────────────────────┘
                runs on the tailnet hub
                http://<tailscale-ip>:5000
```

| Component | Implementation |
|---|---|
| Backend (`server-monitor-go/cmd/server`) | Go ≥1.22, stdlib `net/http` method router, no framework |
| Real-time push | gorilla/websocket fan-out hub; broadcast on every accepted report |
| Storage | SQLite via `modernc.org/sqlite` (pure Go, no cgo → static binary); schema auto-migrates |
| Auth | bcrypt password hash + JWT (HS256), 1-day sessions; signing key auto-generated and persisted (`data/.secret`) or pinned via `SECRET_KEY` |
| Agent (`server-monitor-go/cmd/agent`) | gopsutil metrics in one static binary; Unix daemon + Windows SCM service wrappers |
| Frontend (`frontend/`) | Next.js 14 App Router, TypeScript, Tailwind + shadcn/ui (Radix), TanStack Query, Recharts, lucide icons, dark-first theme (next-themes), WebSocket subscription with REST polling fallback |
| Deploy | systemd units (`deploy/monitor-server.service`, `deploy/monitor-frontend.service`), install scripts for Linux (bash) and Windows (PowerShell) |

## 7. Functional requirements (current feature set)

### 7.1 Metrics collection (agent)

The agent reports every `--interval` (default **2s**) to
`POST /api/servers/update`:

- **Live metrics:** CPU %, memory %, disk %, network in/out (KB/s), uptime (s)
- **System identity:** OS/distribution, CPU model + thread count, total
  memory (GB), total disk (GB), hostname
- **Network identity:** public IPv4/IPv6 (via api.ipify.org),
  **Tailscale IP** (via `tailscale ip -4`, falling back to a
  100.64.0.0/10 interface scan)
- **Location:** two-letter country code auto-detected via ip-api.com,
  rendered as a flag on the dashboard
- **Type heuristic:** VPS vs Dedicated Server

Agent flags: `--name` (must match a registered client), `--server` (hub URL,
or `MONITOR_SERVER` env), `--interval`.

### 7.2 Server lifecycle & allow-list

- Reports are accepted **only** from names registered in the allow-list
  (`allowed_clients`); unregistered names receive HTTP 403.
- Servers are keyed by name; public `id = md5(name)`.
- **Status lifecycle:** registered-but-silent → `maintenance` ("Pending");
  first accepted report → `running`; **30s of silence** → `stopped`
  (background monitor goroutine, 15s sweep). Admins can also force a status
  (running / stopped / maintenance) via the API/UI.
- Deleting a server removes both its row and its allow-list entry.

### 7.3 Dashboard (public, no login required)

- Live table of all servers: status badge, name, location flag, uptime,
  network in/out, CPU / memory / disk gauges.
- **Real-time:** WebSocket push on every change; automatic fallback to REST
  polling when the socket is unavailable.
- Expandable rows revealing system details: hostname, **Tailscale IP**, OS,
  CPU model, total memory/disk, IP address.
- Responsive: card-based layout on mobile (`MobileServerCard`).
- Dark / light theme toggle.
- Summary stats (total / running / stopped counts) with a path into the
  admin panel.
- **IP masking:** unauthenticated viewers see `***.***.***.**`; real
  addresses require an admin JWT.

### 7.4 Admin panel (`/admin`, JWT-protected)

- **First-run initialization:** set the admin password on first visit
  (`/api/auth/initialize`); login issues a 1-day JWT; password reset
  available from the panel.
- **Add Client** modal: registers a name on the allow-list; the new machine
  appears immediately as Pending.
- **Client table** with per-client, copy-to-clipboard install commands for
  Linux and Windows.
- Server management: force status, delete (server + allow-list entry),
  drag-and-drop display ordering (`order_index`, persisted via
  `PUT /api/servers/{id}/order`).
- Admin-only columns (real IPs, Tailscale IPs).

### 7.5 Distribution & onboarding

- `scripts/build.sh` cross-compiles all targets into `dist/`:
  `monitor-server-linux-{amd64,arm64}`,
  `monitor-agent-{linux-amd64,linux-arm64,windows-amd64.exe,darwin-amd64,darwin-arm64}`.
- The hub serves these binaries itself at `GET /download/<file>`
  (`AGENTS_DIR`), so target machines need no Go toolchain and no GitHub
  access.
- **Linux:** `install_agent.sh <name> <hub-url>` detects the architecture,
  downloads the agent from the hub, and installs an auto-restarting,
  boot-enabled systemd service (`server-monitor-agent`).
- **Windows:** `install_agent.ps1 -NodeName <n> -ServerUrl <url>` registers
  an auto-starting `ServerMonitorAgent` SCM service.
- **Hub:** systemd units for backend and frontend in `deploy/`.
- Step-by-step onboarding tutorial: `docs/add-machine.md`.

### 7.6 API contract

`/api/v1/...` is the **canonical surface**; every legacy `/api/...` path below
remains functional as an alias to the same handler (deployed agents and older
clients keep working). On v1, mutating routes require `Authorization: Bearer
<jwt>`; legacy paths stay open (the tailnet is the boundary). Errors are flat
`{"error":"msg"}` on legacy paths and nested `{"error":{"code","message"}}` on
v1. The WebSocket masks IPs for anonymous sockets and sends unmasked frames to
sockets that present a valid `?token=` (admin).

Core contract (both `/api/...` and `/api/v1/...`):

| Method | Path | Auth (v1) | Notes |
|---|---|---|---|
| GET | `/servers` | — | List; IPs masked unless admin |
| GET | `/servers/{id}` | — | Single server |
| POST | `/servers/update` | — | Agent ingest (allow-listed names only; 403 otherwise) |
| PUT | `/servers/{id}/status` | JWT | Force status |
| PUT | `/servers/{id}/order` | JWT | Set display order |
| POST | `/servers/{id}/heartbeat` | — | Lightweight liveness |
| DELETE | `/servers/{id}` | JWT | Remove server + allow-list entry |
| GET | `/clients` | — | List allow-listed clients |
| POST | `/clients` | JWT | Add an allow-listed client |
| GET | `/auth/status` | — | `{initialized}` |
| POST | `/auth/initialize` | — | First-run admin password |
| POST | `/auth/login` | — | Returns 1-day JWT |
| POST | `/auth/reset-password` | — | Change admin password |
| GET | `/ws/dashboard` | `?token=` | WebSocket: server list (masked unless admin token) |

New in v1 only:

| Method | Path | Auth | Notes |
|---|---|---|---|
| GET | `/api/v1/servers/{id}/metrics?range=1h\|6h\|24h\|7d` | — | Downsampled time series (<=500 points) |
| GET | `/api/v1/metrics/summary?range=...` | — | Fleet KPIs + previous-window deltas + uptime % |
| GET | `/api/v1/alerts?severity=&limit=` | — | Status-change + threshold alerts |
| POST | `/api/v1/alerts/{id}/acknowledge` | JWT | Acknowledge an alert |
| GET | `/api/v1/admin/unknown-agents` | JWT | Recently rejected (unregistered) reports |

Unversioned: `GET /download/<file>` (agent binaries), `GET /healthz` (liveness).

### 7.7 Configuration (hub)

| Env var | Default | Meaning |
|---|---|---|
| `ADDR` | `0.0.0.0:5000` | Listen address |
| `DATA_DIR` | `./data` | Holds `servers.db` + auto-generated `.secret` |
| `DATABASE_PATH` | `<DATA_DIR>/servers.db` | SQLite file |
| `AGENTS_DIR` | `./dist` | Binaries served at `/download` (empty ⇒ disabled) |
| `SECRET_KEY` | auto, persisted | JWT signing key |
| `STALE_AFTER_SECONDS` | `30` | Silence before `running` → `stopped` |
| `ALERT_WEBHOOK_URL` | _(empty)_ | Generic webhook for alert JSON (empty ⇒ disabled) |
| `ALERT_DISK_THRESHOLD` | `90` | Disk-usage percent that raises a threshold alert |
| `LOG_DATABASE_URL` | _(empty)_ | External Postgres for per-VM logs; empty ⇒ logs disabled |

Frontend: `PORT` (dashboard port); the dashboard auto-detects the API at
`<host>:5000`, overridable with `NEXT_PUBLIC_API_URL`.

### 7.8 Per-VM log monitoring

Beyond metrics, the agent can tail log files (`--logs`/`MONITOR_LOGS`) and ship
them to the hub, which stores them in an **external Postgres** (`LOG_DATABASE_URL`;
the core stays on SQLite). The **Logs & Activity** page tails and searches them
per node with a **multi-select app/module filter**, level filter, message
keyword grep, and a **live SSE tail**. Logs adopt the
[log-geulis](https://github.com/T0MYAMMM/log-geulis) format
(`TS | LEVEL | MODULE | MESSAGE`); journald and Docker sources are captured via
small "bridge" services. Endpoints: `POST /api/v1/logs`,
`GET /api/v1/servers/{id}/logs` (+`/modules`, `/stream`). Full cookbook:
`docs/logs.md`; live topology: `docs/operations.md`.

## 8. Non-functional requirements

| Requirement | Current behavior |
|---|---|
| **Freshness** | Metric-to-dashboard latency ≤ ~2s (report interval) + WebSocket push |
| **Stale detection** | `running → stopped` within 30–45s of agent silence |
| **Footprint** | Agent is a single binary, unprivileged where possible (`DynamicUser` systemd unit); hub serves the fleet from one small Go process + SQLite |
| **Resilience** | Agent and hub services auto-restart (`Restart=always`); agent retries until the hub returns; hub state persists across restarts (SQLite + persisted JWT secret) |
| **Security** | Tailnet-only exposure; allow-list ingest; bcrypt + JWT admin auth; IP masking for anonymous viewers; CORS deliberately open (access gated by the tailnet, not the browser) |
| **Portability** | Hub: Linux amd64/arm64. Agent: Linux amd64/arm64, Windows amd64, macOS Intel/ARM |

## 9. Success criteria

- A new machine goes from "never seen" to live on the dashboard in
  ≤ 2 minutes using only the documented two steps (register + one command).
- A viewer on the tailnet can assess fleet health (who's up, who's loaded)
  from a single page with no login.
- An agent or hub reboot requires zero manual intervention to recover.
- No monitoring traffic or dashboard page is reachable from outside the
  tailnet.

## 10. Known limitations / candidate roadmap

Observed in operation:

1. ~~**No ingest observability**~~ — **Addressed.** Rejected reports are logged
   (name + remote addr) and recorded in `unknown_agents`, surfaced in the admin
   panel and `GET /api/v1/admin/unknown-agents`.
2. ~~**No history**~~ — **Addressed.** `metrics_samples` + 5-minute rollups
   drive per-server charts, fleet summary KPIs, and uptime %
   (`GET /api/v1/servers/{id}/metrics`, `/api/v1/metrics/summary`).
3. ~~**No alerting**~~ — **Addressed.** Status-transition and disk-threshold
   alerts are recorded and delivered via a generic webhook (`ALERT_WEBHOOK_URL`,
   pluggable notifier); list + acknowledge under `/api/v1/alerts`.
4. **Single admin credential** — one shared password, no users/roles/audit.
5. **External lookups** — geo/IP detection depends on ip-api.com and
   ipify.org being reachable from each agent.
6. **Hard-deletes are unrecoverable in-app** — removing a client deletes its
   registration and server row with no archive/undo.
7. **Plain HTTP** — acceptable inside the tailnet, but a reverse proxy/TLS
   story would be needed for any non-Tailscale deployment.

## 11. References

- Root overview: `README.md`
- Backend/agent docs + deployment guide: `server-monitor-go/README.md`
- Machine onboarding tutorial: `docs/add-machine.md`
- systemd units: `server-monitor-go/deploy/`
- Install scripts: `server-monitor-go/scripts/`
