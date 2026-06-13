# Per-VM Log Monitoring

CloudGuard tails logs on each registered node and surfaces them in the
**Logs & Activity** page. It adopts the [log-geulis](https://github.com/T0MYAMMM/log-geulis)
format (`TS | LEVEL | MODULE | MESSAGE`). Because logs are high-volume, they are
stored in an **external Postgres** rather than the hub's SQLite — the core
monitoring stays a zero-dependency single binary.

```
node: app ─┐
           ├─ (a) log file ──────────────┐
           ├─ (b) journald (systemd) ─┐   │   agent --logs FILE(s)
           └─ (c) docker (json-file) ─┴─ bridge ─► FILE ─► agent ──POST /api/v1/logs──► hub ─► Postgres
                                                                                          │
                       dashboard /logs  ◄── GET .../logs , .../logs/stream (SSE) ─────────┘
```

The feature is **off by default**: with no `LOG_DATABASE_URL` the hub returns
`503` on the log endpoints and the Logs page shows a "not enabled" state.

## 1. Enable on the hub

Provision a database (pure-Go `pgx`; any Postgres 13+; e.g. on a dedicated
box):

```sql
CREATE DATABASE cloudguard_logs;
CREATE USER cloudguard WITH PASSWORD 'choose-a-strong-password';
GRANT ALL PRIVILEGES ON DATABASE cloudguard_logs TO cloudguard;
```

Make Postgres listen on the tailnet interface and allow the hub in `pg_hba.conf`.
Then set `LOG_DATABASE_URL` on the `monitor-server` unit (a drop-in keeps it out
of the tracked unit):

```ini
# /etc/systemd/system/monitor-server.service.d/override.conf  (chmod 600)
[Service]
Environment=LOG_DATABASE_URL=postgres://cloudguard:...@<db-tailscale-ip>:5432/cloudguard_logs
```

`sudo systemctl daemon-reload && sudo systemctl restart monitor-server`. The hub
logs `log database connected` and creates the `logs` table + indexes on first
connect. If it can't reach the DB it logs `log database unavailable; logs
disabled` and the rest of the hub runs normally.

## 2. Ship logs from a node

The agent ships any files passed to `--logs` (or `MONITOR_LOGS`), comma-
separated. It ships only lines appended after it starts, handles rotation/
truncation, parses log-geulis lines, and keeps anything else verbatim at `INFO`
tagged with the file's base name. Ingest is allow-list gated, so the node must
be a registered client (Admin → Add Client) and `--name` must match.

```bash
monitor-agent --name web-1 --server http://<hub-ip>:5000 \
  --logs /var/log/nginx/access.log,/var/log/app/app.log
```

The right way to wire `--logs` depends on **how the app logs**. Pick a recipe:

### (a) App writes a log file
Point `--logs` straight at it. Fold it into the node's existing agent unit so
there's one agent (metrics + logs, no duplication):

```ini
# /etc/systemd/system/server-monitor-agent.service.d/override.conf
[Service]
ExecStart=
ExecStart=/opt/server-monitor-agent/monitor-agent --name <node> \
  --server http://<hub-ip>:5000 --interval 2s --logs /var/log/app/app.log
```

> The installed agent binary must support `--logs`. If it's an old build it will
> exit with status 2 on the unknown flag — replace it:
> `curl -fsSL http://<hub-ip>:5000/download/monitor-agent-linux-amd64 -o /opt/server-monitor-agent/monitor-agent`.

### (b) App uses journald (systemd service)
journald is binary, not a file — bridge it. A small service follows the units
and writes a log-geulis file the agent tails, using the **systemd unit name as
the module**:

```bash
# /usr/local/bin/cloudguard-logbridge.sh
journalctl -n 0 -f -o json --no-pager -u app-api -u app-web \
  | python3 -u -c '
import sys, json, datetime, re
ANSI = re.compile(r"\x1b\[[0-9;]*[A-Za-z]")
for line in sys.stdin:
    try: e = json.loads(line)
    except Exception: continue
    unit = (e.get("_SYSTEMD_UNIT","") or e.get("SYSLOG_IDENTIFIER","")).removesuffix(".service")
    if unit in ("init.scope",""): continue
    msg = e.get("MESSAGE","")
    if isinstance(msg, list): msg = bytes(msg).decode("utf-8","replace")
    msg = ANSI.sub("", msg)
    ts = datetime.datetime.fromtimestamp(int(e["__REALTIME_TIMESTAMP"])/1e6, datetime.timezone.utc).isoformat()
    print(f"{ts} | INFO | {unit} | {msg}", flush=True)
' >> /var/log/cloudguard/node.log
```

Then `--logs /var/log/cloudguard/node.log`. Use `-o json` (not `short-iso`) so
the *unit* name is the module, not the process `comm` (e.g. `node`/`python`).

### (c) App runs in Docker (json-file driver)
Container logs aren't in journald either. Bridge them, reconciling `docker ps`
so new containers are picked up, with the **container name as the module**:

```bash
# reconcile loop; per container: docker logs -f --since 3s --timestamps <c>
#   | python3 (strip ANSI, "TS | INFO | <container> | <msg>") >> /var/log/cloudguard/node.log
```

(See the live hub's `/usr/local/bin/cloudguard-docker-logbridge.sh` for the full
reconciling script.)

### (d) App writes to arbitrary/rotating files (e.g. /tmp glob)
When files appear dynamically (one per job/site), reconcile a glob and follow
each with `tail -F`, tagging the **filename (campaign) as the module**. If the
files are already log-geulis, re-emit them keeping the original logger in the
message:

```bash
for f in /tmp/job_*.log; do
  camp="$(basename "$f" .log)"
  ( tail -F -n 0 "$f" | sed -u "s/^\\([^|]*| [^|]*\\)| \\([^|]*\\)| /\\1| $camp | [\\2] /" >> "$LIVE" ) &
done
```

## 3. Persistence

The shipper must survive disconnects and reboots. Choose by what the node allows:

| Option | When | How |
|---|---|---|
| Fold `--logs` into the **root agent unit** | you have sudo | drop-in override (recipe a); bridges as system services. Single agent, no duplicate metrics. **Preferred.** |
| **User systemd service** + linger | no root, but `loginctl show-user $USER -p Linger` is `yes` | units in `~/.config/systemd/user/`, `systemctl --user enable --now`. |
| **`cron @reboot`** | no sudo, no linger | `@reboot ~/.cloudguard/start.sh` (a launcher that `setsid`s the bridge + agent). Note: under `set -e`, install with `{ crontab -l||true; echo ...; } \| crontab -` (a `grep` returning 1 aborts otherwise). |

Bridges should strip ANSI color codes and `trap 'kill 0' EXIT` so followers die
with the bridge on restart (avoids duplicate tails).

## 4. View — the Logs & Activity page

Pick a **server**, then:
- **App / module** — a checkbox **multi-select** populated per node; choose one
  or several apps to watch together. Resets when you switch nodes.
- **Level** — DEBUG / INFO / WARN / ERROR.
- **Search** — pure keyword **grep over the message** (module has its own filter).
- **Tail / Live** — toggle to stream new lines over SSE (auto-scroll).

Modules appear in the dropdown only **after an app actually logs** — quiet
services/containers show up once they emit a line.

## Endpoints

| Method | Path | Auth | Notes |
|---|---|---|---|
| POST | `/api/v1/logs` | allow-list | Ingest `{server, lines:[{ts,level,module,message,source_file}]}` |
| GET | `/api/v1/servers/{id}/logs?level=&module=&q=&since=&until=&file=&limit=` | — | Filtered query; `module` repeatable for multi-select |
| GET | `/api/v1/servers/{id}/logs/modules` | — | Distinct module names for a server |
| GET | `/api/v1/servers/{id}/logs/stream?after=&level=&module=&q=` | — | Live tail (SSE) |

## Retention & notes

- The `logs` table grows unbounded; add a nightly prune on the log DB, e.g.
  `DELETE FROM logs WHERE ts < now() - interval '30 days';`.
- Log endpoints are public reads (tailnet-gated, like the rest of the dashboard);
  ingest is allow-list gated like metrics.
- A node's log shipper and the apps it tails are independent — if the node
  reboots, ensure the apps are themselves persistent; the shipper only forwards
  whatever exists.
- See `docs/operations.md` for the live fleet's concrete per-node setups.
