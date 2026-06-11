// Package monitor runs the background job that detects silent servers and
// flips them from "running" to "stopped", broadcasting the change to dashboards.
package monitor

import (
	"context"
	"log"
	"time"

	"github.com/thomasstefen/server-monitor/internal/storage/sqlite"
)

// Broadcaster is the subset of the hub the monitor needs to notify dashboards.
type Broadcaster interface {
	BroadcastSnapshot()
}

// Run checks every `interval` for servers that have been silent longer than
// `staleAfter` and marks them stopped. It returns when ctx is cancelled.
func Run(ctx context.Context, s *sqlite.Store, b Broadcaster, interval, staleAfter time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			changed, err := s.MarkStaleStopped(staleAfter)
			if err != nil {
				log.Printf("monitor: %v", err)
				continue
			}
			for _, name := range changed {
				log.Printf("server %q status: running -> stopped (stale)", name)
			}
			if len(changed) > 0 {
				b.BroadcastSnapshot()
			}
		}
	}
}
