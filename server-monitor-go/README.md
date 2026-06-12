# Distributed Server Monitor — Go backend

A real-time monitoring system for distributed servers: a live dashboard of CPU,
memory, disk, network, uptime, OS, location, and Tailscale identity for every
machine on your network. A clean, modular, dependency-light **Go** backend and
**single static-binary agents** that deploy trivially across a Tailscale
tailnet.

```
            agents (Go static binary, one per server)
   web-1 ───┐
   db-1  ───┤  POST /api/servers/update  (every 2s)
   win-1 ───┘            │
                         ▼
              ┌─────────────────────────┐        WebSocket push
              │   monitor-server (Go)   │ ───────────────────────►  Dashboard
              │  REST + WebSocket + WS  │        (Next.js, real-time)
              │      SQLite store       │ ◄───── GET /api/servers (poll fallback)
              └─────────────────────────┘
                 runs on the tailnet hub
                  http://<tailscale-ip>:5000
```

## Stack

| Concern | Implementation |
|---|---|
| HTTP API | net/http with the stdlib (Go 1.22+) method router — no framework |
| Real-time | gorilla/websocket fan-out hub, pushed on every change |
| Storage | SQLite via modernc.org/sqlite (pure Go, no cgo → static binary) |
| Auth | bcrypt password hashing + JWT (HS256) sessions |
| Agent | gopsutil metrics in one static binary — no runtime dependencies |
| Deploy | copy one binary; systemd unit (Linux) or SCM service (Windows) |

## Layout

Layered architecture: `transport → service → storage interface → sqlite`;
`domain` depends on nothing. `main.go` wires the graph with constructor
injection (no globals, no `init()` side effects).

```
server-monitor-go/
├── cmd/
│   ├── server/        # backend entry point (wiring, graceful shutdown)
│   └── agent/         # agent entry point (+ Unix/Windows service wrappers)
├── internal/
│   ├── config/        # env-driven configuration, JWT secret persistence
│   ├── domain/        # core types (Server, Status, Alert, …) + sentinel errors
│   ├── storage/
│   │   └── sqlite/    # SQLite persistence + versioned schema_migrations
│   ├── service/
│   │   ├── servers/   # lifecycle, ingest accept/reject, sweep (Clock-injected)
│   │   ├── auth/      # first-run init, login, reset, token verify
│   │   ├── metrics/   # history, fleet summary, rollup/prune compaction
│   │   └── alerts/    # status/threshold alerts + pluggable Notifier
│   ├── transport/
│   │   ├── http/      # thin handlers, route table (legacy + /api/v1), error mapper
│   │   │   └── middleware/  # request id, slog logging, recovery, CORS, auth
│   │   └── ws/        # WebSocket fan-out to dashboards
│   ├── masking/       # IP-masking rule (single place)
│   ├── auth/          # bcrypt + JWT (HS256) primitives
│   └── metrics/       # agent-side resource + geo/IP collection
├── deploy/            # systemd units for the hub (server + frontend)
├── scripts/           # build.sh, install_agent.sh, install_agent.ps1
└── dist/              # built static binaries (server + per-OS agents)
```

The Next.js dashboard lives in `../frontend`. It subscribes to the backend
WebSocket for instant updates and falls back to polling when the socket is
unavailable (`frontend/pages/index.tsx`, `frontend/config/config.ts`).

## Quick start (development)

```bash
# 1. Backend (serves API + WebSocket + agent downloads on :5000)
cd server-monitor-go
go run ./cmd/server                 # or: ./dist/monitor-server-linux-amd64

# 2. Frontend (dashboard on :3000)
cd ../frontend
npm install && npm run build && npm start

# 3. Register a client, then run an agent for this machine
curl -X POST http://localhost:5000/api/clients \
     -H 'Content-Type: application/json' -d '{"name":"local"}'
go run ./cmd/agent --name local --server http://localhost:5000
```

Open the dashboard, click **Total Servers → Admin**, set the admin password on
first visit, and you'll see `local` go from *Pending* → *Running* with live
metrics.

## Build release binaries

```bash
cd server-monitor-go
bash scripts/build.sh      # outputs static binaries to ./dist for all targets
```

Produces: `monitor-server-linux-{amd64,arm64}` and
`monitor-agent-{linux-amd64,linux-arm64,windows-amd64.exe,darwin-arm64,darwin-amd64}`.

---

# Deploying over Tailscale

The hub (backend + dashboard) runs on **one** machine on your tailnet. Every
other server runs a tiny agent that reports to the hub over the tailnet. No
ports are exposed to the public internet — all traffic stays inside Tailscale.

This tailnet's hub is **`100.98.88.100`** (host `t0myam-sumo-1`). Substitute
your own hub's Tailscale IP/hostname below.

### Step 1 — Put the hub on the tailnet

On the hub machine:

```bash
tailscale up                 # if not already connected
tailscale ip -4              # note the 100.x.y.z address  →  e.g. 100.98.88.100
```

### Step 2 — Run the hub (backend + dashboard)

Manually for a quick start:

```bash
# backend, reachable on the tailscale IP
cd server-monitor-go
ADDR=0.0.0.0:5000 AGENTS_DIR=./dist DATA_DIR=./data ./dist/monitor-server-linux-amd64 &

# dashboard
cd ../frontend && PORT=3000 npm start &
```

Or install as systemd services (survives reboots):

```bash
sudo mkdir -p /opt/server-monitor
sudo cp -r server-monitor-go/dist /opt/server-monitor/dist
sudo cp server-monitor-go/dist/monitor-server-linux-amd64 /opt/server-monitor/monitor-server
sudo cp -r frontend /opt/server-monitor/frontend
sudo cp server-monitor-go/deploy/monitor-server.service /etc/systemd/system/
sudo cp server-monitor-go/deploy/monitor-frontend.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now monitor-server monitor-frontend
```

Dashboard: **`http://100.98.88.100:3000`**  ·  API: **`http://100.98.88.100:5000`**

> The dashboard auto-detects the API at `<host>:5000`, so it works whether you
> reach it via the Tailscale IP, a MagicDNS name, or localhost — no rebuild
> needed.

### Step 3 — Register each server in the dashboard

For every machine you want to monitor, open the dashboard → **Admin** →
**Add Client**, and enter a name (e.g. `web-1`, `db-1`, `win-1`). A server is
only accepted once its name is on this allow-list. The admin panel also shows
copy-paste install commands per client.

### Step 4 — Install the agent on each server

On each machine (must be on the tailnet and able to reach the hub). The agent
binary is downloaded from the hub itself:

**Linux (systemd):**

```bash
# <node-name> must match the name you registered in Step 3
curl -fsSL http://100.98.88.100:5000/download/install_agent.sh -o install_agent.sh 2>/dev/null \
  || true
# (or copy server-monitor-go/scripts/install_agent.sh to the box)
sudo bash install_agent.sh web-1 http://100.98.88.100:5000
```

The script auto-detects the architecture, downloads
`monitor-agent-linux-<arch>` from the hub, and installs an auto-restarting
`server-monitor-agent` systemd service.

**Windows (run elevated PowerShell):**

```powershell
# copy scripts\install_agent.ps1 to the box, then:
.\install_agent.ps1 -NodeName win-1 -ServerUrl http://100.98.88.100:5000
```

Registers an auto-starting `ServerMonitorAgent` Windows service (the agent
binary is SCM-aware).

**Manual / macOS / any platform:**

```bash
# download the right binary from the hub
curl -fSL http://100.98.88.100:5000/download/monitor-agent-linux-amd64 -o monitor-agent
chmod +x monitor-agent
./monitor-agent --name web-1 --server http://100.98.88.100:5000 --interval 2s
```

### Step 5 — Watch them come online

Within a couple of seconds each agent's row flips from *Pending* to *Running*
with live metrics, updating in real time over the dashboard WebSocket.

### Managing agents

```bash
# Linux
systemctl status server-monitor-agent
journalctl -u server-monitor-agent -f
systemctl restart server-monitor-agent
```
```powershell
# Windows
Get-Service ServerMonitorAgent
Restart-Service ServerMonitorAgent
```

---

## Configuration (backend)

All via environment variables (see `internal/config`):

| Var | Default | Meaning |
|---|---|---|
| `ADDR` | `0.0.0.0:5000` | Listen address. `0.0.0.0` ⇒ reachable on the Tailscale IP. |
| `DATA_DIR` | `./data` | Holds `servers.db` and the auto-generated `.secret`. |
| `DATABASE_PATH` | `<DATA_DIR>/servers.db` | SQLite file. |
| `AGENTS_DIR` | `./dist` | Binaries served at `/download/<file>`. Empty ⇒ disabled. |
| `SECRET_KEY` | _(auto, persisted)_ | JWT signing key. Set to keep tokens stable across hosts. |
| `STALE_AFTER_SECONDS` | `30` | Silence before a `running` server is marked `stopped`. |
| `ALERT_WEBHOOK_URL` | _(empty)_ | Generic webhook for alert JSON. Empty ⇒ alert delivery disabled. |
| `ALERT_DISK_THRESHOLD` | `90` | Disk-usage percent that raises a threshold alert. `0` ⇒ disabled. |
| `LOG_DATABASE_URL` | _(empty)_ | External Postgres URL for per-VM log storage (e.g. home-db). Empty ⇒ logs feature disabled; core stays SQLite. See [docs/logs.md](../docs/logs.md). |

Agent flags: `--name` (must match a registered client), `--server` (hub base
URL, or `MONITOR_SERVER` env), `--interval` (default `2s`), `--logs`
(comma-separated log file paths to tail and ship, or `MONITOR_LOGS` env; see
[docs/logs.md](../docs/logs.md)).

## API contract

`/api/v1/...` is the **canonical surface**. Every legacy `/api/...` path below
remains functional as an alias to the same handler, so deployed agents and
older clients keep working unmodified. New endpoints (metrics, alerts,
unknown-agents) are v1-only.

**Difference on v1:** mutating routes require `Authorization: Bearer <jwt>`
(delete, force-status, set-order, add-client, acknowledge); the legacy paths
stay open (the tailnet is the boundary). Reads and agent ingest are open on
both surfaces.

Core contract (available on both `/api/...` and `/api/v1/...`):

| Method | Path | Auth (v1) | Notes |
|---|---|---|---|
| GET | `/servers` | — | List; IPs masked unless `Authorization: Bearer <jwt>`. |
| GET | `/servers/{id}` | — | Single server. |
| POST | `/servers/update` | — | Agent metrics ingest (allow-listed names only; 403 otherwise). |
| PUT | `/servers/{id}/status` | JWT | Force status (running/stopped/maintenance). |
| PUT | `/servers/{id}/order` | JWT | Set display order. |
| POST | `/servers/{id}/heartbeat` | — | Lightweight liveness. |
| DELETE | `/servers/{id}` | JWT | Remove server + allow-list entry. |
| GET | `/clients` | — | List allow-listed clients. |
| POST | `/clients` | JWT | Add an allow-listed client. |
| GET | `/auth/status` | — | `{initialized}`. |
| POST | `/auth/initialize` | — | Set admin password (first run). |
| POST | `/auth/login` | — | Returns a 1-day JWT. |
| POST | `/auth/reset-password` | — | Change admin password (verifies current). |
| GET | `/ws/dashboard` | — | WebSocket: pushes the (IP-masked) server list. |

New in v1 (no legacy alias):

| Method | Path | Auth | Notes |
|---|---|---|---|
| GET | `/api/v1/servers/{id}/metrics?range=1h\|6h\|24h\|7d` | — | Downsampled time series (≤500 points). |
| GET | `/api/v1/metrics/summary?range=...` | — | Fleet KPIs + previous-window deltas + uptime %. |
| GET | `/api/v1/alerts?severity=&limit=` | — | Recent alerts (status changes + threshold breaches). |
| POST | `/api/v1/alerts/{id}/acknowledge` | JWT | Acknowledge an alert. |
| GET | `/api/v1/admin/unknown-agents` | JWT | Recently rejected (unregistered) agent reports. |
| POST | `/api/v1/logs` | allow-list | Agent log ingest (external DB; 503 if disabled). |
| GET | `/api/v1/servers/{id}/logs?level=&q=&since=&until=&file=&limit=` | — | Query a server's logs. |
| GET | `/api/v1/servers/{id}/logs/stream?after=` | — | Live tail a server's logs (SSE). |

Unversioned:

| Method | Path | Notes |
|---|---|---|
| GET | `/download/<file>` | Serves agent binaries from `AGENTS_DIR`. |
| GET | `/healthz` | Liveness. |

Error bodies are `{"error": "<message>"}` on the legacy surface; the v1
auth gate returns the nested `{"error":{"code","message"}}` envelope, which
will become the v1-wide shape when the v1-consuming frontend lands.

## Behaviour notes

- **Status lifecycle:** a registered-but-not-yet-connected client shows
  *maintenance* ("Pending"); it becomes *running* on its first report and
  *stopped* after 30s of silence (checked every 15s by the monitor goroutine).
- **Identity:** servers are keyed by **name**; the public `id` is `md5(name)`.
  Reports are matched by name, so the agent's `--name` must equal the
  registered client name.
- **Security:** unauthenticated viewers see masked IPs (`***.***.***.**`);
  admins see real addresses. Passwords are bcrypt-hashed; sessions are JWT.
  CORS is open because access is expected to be gated by the tailnet.
