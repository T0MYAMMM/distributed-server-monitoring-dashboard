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

```
server-monitor-go/
├── cmd/
│   ├── server/        # backend entry point (main, graceful shutdown)
│   └── agent/         # agent entry point (+ Unix/Windows service wrappers)
├── internal/
│   ├── config/        # env-driven configuration, JWT secret persistence
│   ├── models/        # shared Server/Client types (JSON contract)
│   ├── store/         # SQLite persistence layer (all SQL lives here)
│   ├── auth/          # bcrypt password hashing + JWT (HS256) sessions
│   ├── hub/           # WebSocket fan-out to dashboards
│   ├── api/           # HTTP handlers, CORS, routing, download endpoint
│   ├── monitor/       # background staleness checker (running→stopped)
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

Agent flags: `--name` (must match a registered client), `--server` (hub base
URL, or `MONITOR_SERVER` env), `--interval` (default `2s`).

## API contract

All endpoints consumed by the dashboard and agents:

| Method | Path | Notes |
|---|---|---|
| GET | `/api/servers` | List; IPs masked unless `Authorization: Bearer <jwt>`. |
| GET | `/api/servers/{id}` | Single server. |
| POST | `/api/servers/update` | Agent metrics ingest (allow-listed names only). |
| PUT | `/api/servers/{id}/status` | Force status (running/stopped/maintenance). |
| PUT | `/api/servers/{id}/order` | Set display order. |
| POST | `/api/servers/{id}/heartbeat` | Lightweight liveness. |
| DELETE | `/api/servers/{id}` | Remove server + allow-list entry. |
| GET / POST | `/api/clients` | List / add allow-listed clients. |
| GET | `/api/auth/status` | `{initialized}`. |
| POST | `/api/auth/initialize` | Set admin password (first run). |
| POST | `/api/auth/login` | Returns a 1-day JWT. |
| POST | `/api/auth/reset-password` | Change admin password. |
| GET | `/api/ws/dashboard` | WebSocket: pushes the (IP-masked) server list. |
| GET | `/download/<file>` | Serves agent binaries from `AGENTS_DIR`. |
| GET | `/healthz` | Liveness. |

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
