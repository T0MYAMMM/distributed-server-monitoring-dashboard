//go:build windows

package main

import (
	"context"
	"os/signal"
	"syscall"

	"golang.org/x/sys/windows/svc"
)

// runService runs under the Windows Service Control Manager when launched as a
// service, and as a normal foreground process otherwise (e.g. when run from a
// console for debugging).
func runService(cfg agentConfig) {
	isService, err := svc.IsWindowsService()
	if err == nil && isService {
		_ = svc.Run("ServerMonitorAgent", &windowsService{cfg: cfg})
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	reportLoop(ctx, cfg)
}

// windowsService adapts the report loop to the SCM lifecycle, responding to
// stop/shutdown requests so the service starts and stops cleanly.
type windowsService struct {
	cfg agentConfig
}

func (s *windowsService) Execute(args []string, r <-chan svc.ChangeRequest, status chan<- svc.Status) (bool, uint32) {
	const accepted = svc.AcceptStop | svc.AcceptShutdown
	status <- svc.Status{State: svc.StartPending}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go reportLoop(ctx, s.cfg)

	status <- svc.Status{State: svc.Running, Accepts: accepted}
	for req := range r {
		switch req.Cmd {
		case svc.Interrogate:
			status <- req.CurrentStatus
		case svc.Stop, svc.Shutdown:
			status <- svc.Status{State: svc.StopPending}
			cancel()
			return false, 0
		}
	}
	return false, 0
}
