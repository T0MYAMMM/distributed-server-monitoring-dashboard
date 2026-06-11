#!/usr/bin/env bash
# build.sh — produce static release binaries into ./dist for all targets.
# The server runs on the hub (linux/amd64 here); agents cover the tailnet's
# Linux (amd64/arm64), Windows, and macOS hosts.
set -euo pipefail
cd "$(dirname "$0")/.."

mkdir -p dist
LDFLAGS="-s -w"

build() { # $1=os $2=arch $3=cmd $4=out
  echo "  ${1}/${2}: ${4}"
  GOOS="$1" GOARCH="$2" CGO_ENABLED=0 go build -trimpath -ldflags="$LDFLAGS" -o "dist/$4" "./cmd/$3"
}

echo "Building server..."
build linux amd64 server monitor-server-linux-amd64
build linux arm64 server monitor-server-linux-arm64

echo "Building agents..."
build linux   amd64 agent monitor-agent-linux-amd64
build linux   arm64 agent monitor-agent-linux-arm64
build windows amd64 agent monitor-agent-windows-amd64.exe
build darwin  arm64 agent monitor-agent-darwin-arm64
build darwin  amd64 agent monitor-agent-darwin-amd64

echo "Done. Artifacts in ./dist:"
ls -1 dist/
