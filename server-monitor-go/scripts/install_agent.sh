#!/usr/bin/env bash
#
# install_agent.sh — install the Go monitoring agent as a systemd service.
#
# The agent is a single static binary downloaded from the monitoring hub
# itself, so the only prerequisite on a monitored server is being on the
# tailnet (or otherwise able to reach the hub).
#
# Usage:
#   sudo ./install_agent.sh <node-name> <server-url>
#
# Examples:
#   sudo ./install_agent.sh web-1   http://100.98.88.100:5000
#   sudo ./install_agent.sh db-1    http://monitor-hub:5000
#
# <node-name> MUST match a client you registered in the dashboard
# (Admin → Add Client). Until it matches an allow-listed name, the hub
# rejects the agent's reports.
set -euo pipefail

GREEN='\033[0;32m'; RED='\033[0;31m'; YELLOW='\033[1;33m'; NC='\033[0m'
say() { echo -e "${2:-$GREEN}${1}${NC}"; }

if [[ $EUID -ne 0 ]]; then
  say "Please run as root (sudo)." "$RED"; exit 1
fi
if [[ $# -ne 2 ]]; then
  say "Usage: sudo $0 <node-name> <server-url>" "$RED"; exit 1
fi

NODE_NAME="${1//[\"\']/}"
SERVER_URL="${2%/}"   # strip trailing slash

# Map uname arch to the agent build we publish.
case "$(uname -m)" in
  x86_64|amd64)        ARCH=amd64 ;;
  aarch64|arm64)       ARCH=arm64 ;;
  *) say "Unsupported architecture: $(uname -m)" "$RED"; exit 1 ;;
esac

BIN_URL="${SERVER_URL}/download/monitor-agent-linux-${ARCH}"
INSTALL_DIR=/opt/server-monitor-agent
BIN_PATH="${INSTALL_DIR}/monitor-agent"
SERVICE=/etc/systemd/system/server-monitor-agent.service

say "Installing monitoring agent"
say "  node name : ${NODE_NAME}"
say "  hub       : ${SERVER_URL}"
say "  arch      : linux/${ARCH}"

mkdir -p "${INSTALL_DIR}"

say "Downloading agent from ${BIN_URL} ..."
if command -v curl >/dev/null 2>&1; then
  curl -fSL "${BIN_URL}" -o "${BIN_PATH}"
elif command -v wget >/dev/null 2>&1; then
  wget -q "${BIN_URL}" -O "${BIN_PATH}"
else
  say "Neither curl nor wget is available." "$RED"; exit 1
fi
chmod +x "${BIN_PATH}"

say "Writing systemd unit ${SERVICE} ..."
cat > "${SERVICE}" <<EOF
[Unit]
Description=Distributed Server Monitor Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=${BIN_PATH} --name "${NODE_NAME}" --server "${SERVER_URL}" --interval 2s
Restart=always
RestartSec=5
# Lightweight resource collection; run unprivileged where possible.
DynamicUser=yes

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable --now server-monitor-agent

say "Done. The agent is running and will report '${NODE_NAME}' to ${SERVER_URL}."
say "Check status: systemctl status server-monitor-agent" "$YELLOW"
say "Follow logs : journalctl -u server-monitor-agent -f" "$YELLOW"
