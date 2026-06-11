// Command server is the monitoring backend: a REST + WebSocket API backed by
// SQLite. Run it on each network's hub host (reachable on its Tailscale IP)
// and point agents and the dashboard at it.
//
// Wiring only: config -> deps -> router -> serve. The dependency graph is built
// explicitly here with constructor injection; no globals or init() side effects.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/thomasstefen/server-monitor/internal/auth"
	"github.com/thomasstefen/server-monitor/internal/config"
	authsvc "github.com/thomasstefen/server-monitor/internal/service/auth"
	"github.com/thomasstefen/server-monitor/internal/service/servers"
	"github.com/thomasstefen/server-monitor/internal/storage/sqlite"
	httpapi "github.com/thomasstefen/server-monitor/internal/transport/http"
	"github.com/thomasstefen/server-monitor/internal/transport/ws"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("config", "err", err)
		os.Exit(1)
	}

	db, err := sqlite.Open(cfg.DatabasePath)
	if err != nil {
		logger.Error("store", "err", err)
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("database ready", "path", cfg.DatabasePath)

	// Build the dependency graph explicitly.
	authPrimitives := auth.New(cfg.SecretKey)
	authService := authsvc.New(db, authPrimitives)
	serversService := servers.New(db, servers.SystemClock{}, logger)
	hub := ws.New()
	handlers := httpapi.New(serversService, authService, hub, cfg.AgentsDir, logger)

	// Background staleness sweeper; cancelled on shutdown.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	staleAfter := time.Duration(cfg.StaleAfterSeconds) * time.Second
	go serversService.RunSweeper(ctx, 15*time.Second, staleAfter, handlers.Broadcast)

	srv := &http.Server{
		Addr:         cfg.Addr,
		Handler:      handlers.Handler(logger),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 0, // 0: long-lived WebSocket connections must not time out
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("listening", "addr", cfg.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("listen", "err", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown on SIGINT/SIGTERM: stop the sweeper, drain the WS hub so
	// its handlers return, stop accepting, then close the database.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	logger.Info("shutting down")

	cancel()
	hub.CloseAll()

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutCancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		logger.Error("shutdown", "err", err)
	}
}
