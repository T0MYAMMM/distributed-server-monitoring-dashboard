//go:build !windows

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// runService runs the report loop in the foreground, stopping cleanly on
// SIGINT/SIGTERM. On Unix the agent is supervised by systemd, which handles
// restart and boot-start, so no service framework is needed here.
func runService(cfg agentConfig) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	reportLoop(ctx, cfg)
	_ = os.Stdout.Sync()
}
