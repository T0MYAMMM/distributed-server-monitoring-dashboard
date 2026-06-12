// Command agent runs on each monitored server. It samples local resource
// usage and POSTs it to the backend on a fixed interval. A single static
// binary with no runtime dependencies, so it deploys cleanly across the
// tailnet. On Windows it runs correctly under the Service Control Manager.
//
// Usage:
//
//	agent --name web-1 --server http://100.98.88.100:5000 --interval 2s
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/thomasstefen/server-monitor/internal/logtail"
	"github.com/thomasstefen/server-monitor/internal/metrics"
)

// agentConfig holds resolved runtime settings.
type agentConfig struct {
	nodeName   string
	serverBase string
	endpoint   string
	interval   time.Duration
	logPaths   []string
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.SetPrefix("[agent] ")

	hostname, _ := os.Hostname()
	name := flag.String("name", hostname, "node name (must match the client registered in the dashboard)")
	server := flag.String("server", envOr("MONITOR_SERVER", "http://localhost:5000"), "backend base URL")
	interval := flag.Duration("interval", 2*time.Second, "reporting interval")
	logs := flag.String("logs", envOr("MONITOR_LOGS", ""), "comma-separated log file paths to tail and ship to the hub")
	flag.Parse()

	base := strings.TrimRight(*server, "/")
	cfg := agentConfig{
		nodeName:   strings.Trim(strings.TrimSpace(*name), `"'`),
		serverBase: base,
		endpoint:   base + "/api/servers/update",
		interval:   *interval,
		logPaths:   splitPaths(*logs),
	}

	// runService dispatches to the OS-appropriate entry point (a plain
	// foreground loop everywhere, plus Windows SCM integration on Windows).
	runService(cfg)
}

// splitPaths parses a comma-separated path list, trimming blanks.
func splitPaths(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// reportLoop samples and reports metrics until ctx is cancelled. When log paths
// are configured it also runs the log tailer alongside.
func reportLoop(ctx context.Context, cfg agentConfig) {
	log.Printf("node=%q server=%s interval=%s", cfg.nodeName, cfg.endpoint, cfg.interval)

	if len(cfg.logPaths) > 0 {
		go logtail.New(cfg.serverBase, cfg.nodeName, cfg.logPaths).Run(ctx, cfg.interval)
	}

	collector := metrics.NewCollector(cfg.nodeName)
	client := &http.Client{Timeout: 8 * time.Second}

	ticker := time.NewTicker(cfg.interval)
	defer ticker.Stop()
	for {
		report(client, cfg.endpoint, collector)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// report samples metrics and posts them, with a short retry on failure.
func report(client *http.Client, endpoint string, c *metrics.Collector) {
	body, err := json.Marshal(c.Sample())
	if err != nil {
		log.Printf("marshal: %v", err)
		return
	}

	const maxRetries = 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		resp, err := client.Post(endpoint, "application/json", bytes.NewReader(body))
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
			log.Printf("post failed (attempt %d/%d): status %d", attempt, maxRetries, resp.StatusCode)
		} else {
			log.Printf("post failed (attempt %d/%d): %v", attempt, maxRetries, err)
		}
		if attempt < maxRetries {
			time.Sleep(time.Second)
		}
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
