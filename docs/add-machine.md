# Tutorial: add a new machine to the monitoring dashboard

This walks through connecting a brand-new server to the dashboard by pulling
this repo. Takes about 2 minutes per machine.

**Prerequisites on the new machine:**
- It can reach the hub — easiest is joining your Tailscale tailnet
- `git` and `curl` (or `wget`)
- Root/Administrator access (to install the agent as a service)

Throughout, replace `100.98.88.100` with your hub's Tailscale IP
(`tailscale ip -4` on the hub) and `my-server-1` with the name you want shown
on the dashboard.

---

## Step 1 — Join the machine to your tailnet

```bash
# install tailscale if needed: https://tailscale.com/download
sudo tailscale up
tailscale ip -4        # confirm it got a 100.x.y.z address
```

Sanity check that it can reach the hub:

```bash
curl http://100.98.88.100:5000/healthz
# → {"status":"ok"}
```

## Step 2 — Register the machine in the dashboard

From any browser on the tailnet, open the admin panel:

1. Go to `http://100.98.88.100:8088` → click the arrow on **Total Servers**
   (or navigate to `/admin`) and log in
2. Click **Add Client**
3. Enter the node name, e.g. `my-server-1`, and confirm

The dashboard now shows the machine as **Maintenance / Pending** — it is
allow-listed and waiting for its agent. The hub only accepts reports from
registered names, so this step must come first.

## Step 3 — Pull the repo and install the agent

On the new machine:

```bash
git clone https://github.com/T0MYAMMM/distributed-server-monitoring-dashboard.git
cd distributed-server-monitoring-dashboard
sudo bash server-monitor-go/scripts/install_agent.sh my-server-1 http://100.98.88.100:5000
```

The script:
- detects the CPU architecture (amd64/arm64)
- downloads the matching static agent binary **from the hub itself**
  (`/download/monitor-agent-linux-<arch>`) — no Go toolchain needed
- installs an auto-restarting, boot-enabled systemd service
  (`server-monitor-agent`)

### Windows instead?

Clone the repo (or just copy `server-monitor-go/scripts/install_agent.ps1`),
then in an **elevated** PowerShell:

```powershell
.\server-monitor-go\scripts\install_agent.ps1 -NodeName my-server-1 -ServerUrl http://100.98.88.100:5000
```

This registers an auto-starting `ServerMonitorAgent` Windows service.

## Step 4 — Verify

Within a couple of seconds the dashboard row flips from *Pending* to
**Running** with live CPU/memory/disk/network. Expand the row to see the
machine's **hostname and Tailscale IP**.

On the machine itself:

```bash
systemctl status server-monitor-agent       # should be active (running)
journalctl -u server-monitor-agent -f       # live agent logs
```

---

## Alternative: build the agent from source

If you'd rather compile than download from the hub (requires Go 1.26+):

```bash
cd distributed-server-monitoring-dashboard/server-monitor-go
go build -o /opt/server-monitor-agent/monitor-agent ./cmd/agent
/opt/server-monitor-agent/monitor-agent --name my-server-1 --server http://100.98.88.100:5000
```

To run it under systemd, reuse the unit that `install_agent.sh` writes
(`/etc/systemd/system/server-monitor-agent.service`) or run the script after
copying your built binary into place — it only rewrites the unit and restarts.

## Troubleshooting

| Symptom | Likely cause / fix |
|---|---|
| `403 Client not allowed` in agent logs | The `--name` doesn't match a registered client. Names are case-sensitive — re-check Step 2. |
| Stuck on *Pending* | Agent isn't reaching the hub: `curl http://<hub>:5000/healthz` from the machine; check tailscale status. |
| Shows *Stopped* | Agent stopped reporting >30s ago: `systemctl restart server-monitor-agent` and check `journalctl`. |
| Download fails in install script | The hub must run with `AGENTS_DIR` pointing at built binaries (`bash scripts/build.sh` on the hub, default `./dist`). |

## Removing a machine

Dashboard → Admin → trash icon on the row (removes the server and its
allow-list entry), then on the machine:

```bash
sudo systemctl disable --now server-monitor-agent
sudo rm -rf /opt/server-monitor-agent /etc/systemd/system/server-monitor-agent.service
sudo systemctl daemon-reload
```
