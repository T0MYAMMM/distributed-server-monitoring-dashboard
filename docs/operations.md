# Operations Runbook

How the live deployment is wired and how to operate it. Pairs with
`docs/logs.md` (log collection) and `docs/architecture.md` (code layout).

## Topology

```
                       Tailscale tailnet (private)
  hub  (VM-3-221-ubuntu, 100.98.88.100)
   ├─ monitor-server.service     Go backend + REST + WS         :5000
   ├─ monitor-frontend.service   Next.js dashboard              :8088
   ├─ server-monitor-agent       agent: metrics + node logs  (--name monitor-hub)
   ├─ cloudguard-hub-logbridge   journald → /var/log/cloudguard/hub.log
   └─ cloudguard-docker-logbridge docker logs → /var/log/cloudguard/hub.log
  log DB (external Postgres, separate box)  ── cloudguard_logs.logs
  agents (one per monitored node, push to the hub):
   ├─ monitor-hub   (this VM's own services + containers)
   ├─ neva-1        (niskala-app)
   └─ edts025131    (tlusr backfill crawlers)
```

- Dashboard: `http://100.98.88.100:8088` (admin password is set on first run via
  Admin → Login; reset via Admin → Reset Password).
- API: `http://100.98.88.100:5000` (legacy `/api`) and `/api/v1` (canonical).
- The services run **from the repo working tree** (not `/opt`); `WorkingDirectory`
  is `…/server-monitor-go` and `…/frontend`. `data/` holds the SQLite DB + JWT
  secret; `dist/` holds the agent binaries served at `/download`.

## Redeploy from `main`

The services run the built artifacts, so a deploy is: build → swap → restart.

```bash
cd server-monitor-go
go build -o /tmp/ms ./cmd/server          # build to temp (running binary is busy)
bash scripts/build.sh                      # refresh dist/ (agent downloads)
sudo systemctl stop monitor-server
cp /tmp/ms bin/monitor-server              # bin/ is gitignored
sudo systemctl start monitor-server

cd ../frontend && npm ci && npm run build  # then:
sudo systemctl restart monitor-frontend
```

Verify: `curl -s localhost:5000/healthz`, `curl -s -o /dev/null -w '%{http_code}'
localhost:8088/`. The live SQLite DB auto-migrates in place on backend start.

## Onboard a new node (metrics)

1. Dashboard → **Admin → Add Client** → register a name (or
   `curl -X POST http://<hub>:5000/api/clients -d '{"name":"<node>"}'`).
2. On the node (on the tailnet):
   ```bash
   curl -fsSL http://<hub>:5000/download/install_agent.sh | sudo bash -s <node> http://<hub>:5000
   ```
   …or download `monitor-agent-linux-amd64` and run it under systemd. It appears
   `running` within ~2s. See `docs/add-machine.md`.

## Onboard a node's logs

Register the node (above), then ship logs per `docs/logs.md` (pick the recipe
that matches how the app logs, and a persistence option). Concrete live
examples:

- **monitor-hub** (sudo) — `--logs /var/log/cloudguard/hub.log` folded into the
  root agent unit; two system bridges feed that file: a **journald bridge**
  (`os-vps`, `tlusr-panel`, `tlusr-parser`, `tlusr-scraper`, `monitor-frontend`)
  and a **docker bridge** (all running containers, module = container name).
  `monitor-server` is deliberately **excluded** to avoid a log feedback loop.
- **neva-1** (no sudo, linger=yes) — two **user systemd services**: a journald
  bridge for the niskala services and a log-shipping agent.
- **edts025131** (no sudo, linger=no) — **`cron @reboot`** launches a file-glob
  bridge over `/tmp/<campaign>.log` (tlusr backfill crawlers, module = campaign)
  plus a log-shipping agent.

## Health & troubleshooting

| Symptom | Check |
|---|---|
| Node shows `stopped` | agent service on the node; tailnet reachability to `:5000`; `journalctl -u server-monitor-agent`. |
| Node not appearing | name registered? agent `--name` matches? rejected reports show in **Admin → Recent Unknown Agents**. |
| Logs page says "not enabled" | `LOG_DATABASE_URL` set + DB reachable; hub log line `log database connected`. |
| A node's modules empty | that node's bridge/agent running; the apps have actually logged since the shipper started. |
| Agent exits status 2 with `--logs` | old installed binary; re-download from `/download`. |
| Duplicate metrics for a node | two agents reporting the same `--name`; fold `--logs` into the single root agent. |

## Backups & retention

- Core state: `server-monitor-go/data/servers.db` + `data/.secret` (back these
  up to keep history and stable JWTs).
- Log DB: prune `logs` on a schedule (see `docs/logs.md`).
