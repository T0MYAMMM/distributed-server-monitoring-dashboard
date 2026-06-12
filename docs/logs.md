# Per-VM Log Monitoring

CloudGuard can tail log files on each registered VM and surface them in the
**Logs & Activity** page. It adopts the [log-geulis](https://github.com/T0MYAMMM/log-geulis)
format (`TS | LEVEL | MODULE | MESSAGE`). Because logs are high-volume, they are
stored in an **external database** (e.g. the `home-db` Postgres) rather than the
hub's SQLite — the core monitoring stays a zero-dependency single binary.

```
agent (--logs /path/a,/path/b)  ──POST /api/v1/logs──►  hub  ──►  Postgres (home-db)
                                                          │
dashboard /logs  ◄──GET .../logs , .../logs/stream (SSE)──┘
```

The feature is **off by default**: with no `LOG_DATABASE_URL` the hub returns
`503` on the log endpoints and the Logs page shows a "not enabled" state.

## 1. Provision the database on home-db

On the `home-db` machine (reachable on its Tailscale IP), create a database and
user (pure-Go `pgx`; any Postgres 13+ works):

```sql
CREATE DATABASE cloudguard_logs;
CREATE USER cloudguard WITH PASSWORD 'choose-a-strong-password';
GRANT ALL PRIVILEGES ON DATABASE cloudguard_logs TO cloudguard;
```

Make sure Postgres listens on the Tailscale interface and `pg_hba.conf` allows
the hub's tailnet address. The hub creates the `logs` table and indexes on first
connect.

## 2. Point the hub at it

Set `LOG_DATABASE_URL` on the hub (the `monitor-server` systemd unit):

```ini
Environment=LOG_DATABASE_URL=postgres://cloudguard:choose-a-strong-password@<home-db-tailscale-ip>:5432/cloudguard_logs
```

Then `sudo systemctl daemon-reload && sudo systemctl restart monitor-server`.
The log shows `log database connected` when it works (and `log database
unavailable; logs disabled` if it cannot reach it — the rest of the hub still
runs normally).

## 3. Ship logs from each VM

Run the agent with `--logs` (or the `MONITOR_LOGS` env var), a comma-separated
list of files to tail:

```bash
monitor-agent --name web-1 --server http://<hub-ip>:5000 \
  --logs /var/log/nginx/access.log,/var/log/app/app.log
```

For the systemd unit, add the flag to `ExecStart` (or set
`Environment=MONITOR_LOGS=...`). The agent ships only lines appended after it
starts, handles rotation/truncation, and parses log-geulis-formatted lines;
anything else is kept verbatim at `INFO`, tagged with the file's base name.

## 4. View

Open **Logs & Activity**, pick a server, filter by level or search text, and
toggle **Live** to stream new lines (SSE).

## Endpoints

| Method | Path | Auth | Notes |
|---|---|---|---|
| POST | `/api/v1/logs` | allow-list | Agent ingest: `{server, lines:[{ts,level,module,message,source_file}]}` |
| GET | `/api/v1/servers/{id}/logs?level=&q=&since=&until=&file=&limit=` | — | Filtered query (newest first) |
| GET | `/api/v1/servers/{id}/logs/stream?after=` | — | Live tail (Server-Sent Events) |

## Notes & retention

- The `logs` table grows unbounded; add a retention job on home-db if needed,
  e.g. a nightly `DELETE FROM logs WHERE ts < now() - interval '30 days';`.
- The log endpoints are public reads (like the rest of the tailnet-gated
  dashboard); ingest is allow-list gated like metrics.
