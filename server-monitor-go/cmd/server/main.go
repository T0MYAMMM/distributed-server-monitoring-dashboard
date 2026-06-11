// Command server is the monitoring backend: a REST + WebSocket API backed by
// SQLite. Run it on each network's hub host (reachable on its Tailscale IP)
// and point agents and the dashboard at it.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/thomasstefen/server-monitor/internal/api"
	"github.com/thomasstefen/server-monitor/internal/auth"
	"github.com/thomasstefen/server-monitor/internal/config"
	"github.com/thomasstefen/server-monitor/internal/hub"
	"github.com/thomasstefen/server-monitor/internal/monitor"
	"github.com/thomasstefen/server-monitor/internal/store"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.SetPrefix("[monitor] ")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	st, err := store.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	defer st.Close()
	log.Printf("database ready at %s", cfg.DatabasePath)

	h := hub.New()
	a := api.New(st, auth.New(cfg.SecretKey), h, cfg.AgentsDir)

	// Background staleness checker.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	staleAfter := time.Duration(cfg.StaleAfterSeconds) * time.Second
	go monitor.Run(ctx, st, a, 15*time.Second, staleAfter)

	srv := &http.Server{
		Addr:         cfg.Addr,
		Handler:      a.Handler(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 0, // 0: long-lived WebSocket connections must not time out
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("listening on %s", cfg.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	// Graceful shutdown on SIGINT/SIGTERM.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Println("shutting down...")

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutCancel()
	_ = srv.Shutdown(shutCtx)
}
