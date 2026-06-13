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
	alertssvc "github.com/thomasstefen/server-monitor/internal/service/alerts"
	authsvc "github.com/thomasstefen/server-monitor/internal/service/auth"
	channelssvc "github.com/thomasstefen/server-monitor/internal/service/channels"
	feedbacksvc "github.com/thomasstefen/server-monitor/internal/service/feedback"
	logssvc "github.com/thomasstefen/server-monitor/internal/service/logs"
	metricssvc "github.com/thomasstefen/server-monitor/internal/service/metrics"
	"github.com/thomasstefen/server-monitor/internal/service/servers"
	settingssvc "github.com/thomasstefen/server-monitor/internal/service/settings"
	"github.com/thomasstefen/server-monitor/internal/storage/postgres"
	"github.com/thomasstefen/server-monitor/internal/storage/sqlite"
	httpapi "github.com/thomasstefen/server-monitor/internal/transport/http"
	"github.com/thomasstefen/server-monitor/internal/transport/ws"
)

// version is the build version reported in Settings → About.
const version = "1.0.0"

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
	metricsService := metricssvc.New(db, metricssvc.SystemClock{}, logger)
	hub := ws.New()

	// Settings: env-backed values made editable in-app (env still wins). Loaded
	// before the services that consume them so stored overrides apply at boot.
	settingsService, err := settingssvc.New(db, settingssvc.Defaults{
		InstanceName:  "CloudGuard",
		DiskThreshold: cfg.AlertDiskThreshold,
		StaleAfter:    cfg.StaleAfterSeconds,
	}, logger)
	if err != nil {
		logger.Error("settings", "err", err)
		os.Exit(1)
	}

	// Notification channels: outbound alert targets managed on the Integrations
	// page. They implement alerts.Notifier and are combined with the legacy env
	// webhook so alerts fan out to both.
	channelsService := channelssvc.New(db, logger)
	notifier := alertssvc.NewMultiNotifier(
		alertssvc.NewNotifier(cfg.AlertWebhookURL, logger),
		channelsService,
	)

	// Alerts: a sink wired into the servers service so transitions and threshold
	// breaches emit alerts. onEmit refreshes dashboards via the WS snapshot.
	var handlers *httpapi.Handlers
	alertsService := alertssvc.New(db, notifier, alertssvc.SystemClock{}, cfg.AlertDiskThreshold,
		func() { handlers.Broadcast() }, logger)
	serversService.SetAlertSink(alertsService)

	// Apply live-readable settings now and on every change (disk threshold).
	settingsService.OnApply(func() {
		alertsService.SetDiskThreshold(settingsService.DiskThreshold())
	})
	settingsService.ApplyNow()

	feedbackService := feedbacksvc.New(db, cfg.FeedbackWebhookURL, logger)

	// External log store (Postgres, e.g. the home-db server). Optional: when
	// LOG_DATABASE_URL is unset, the logs feature is disabled and the core
	// monitoring stays entirely on the hub's SQLite.
	var logStore logssvc.Store
	if cfg.LogDatabaseURL != "" {
		openCtx, openCancel := context.WithTimeout(context.Background(), 10*time.Second)
		ps, err := postgres.Open(openCtx, cfg.LogDatabaseURL)
		openCancel()
		if err != nil {
			logger.Error("log database unavailable; logs disabled", "err", err)
		} else {
			logStore = ps
			defer ps.Close()
			logger.Info("log database connected")
		}
	}
	logsService := logssvc.New(logStore, logger)

	handlers = httpapi.New(httpapi.Deps{
		Servers:   serversService,
		Auth:      authService,
		Metrics:   metricsService,
		Alerts:    alertsService,
		Logs:      logsService,
		Settings:  settingsService,
		Channels:  channelsService,
		Feedback:  feedbackService,
		Hub:       hub,
		AgentsDir: cfg.AgentsDir,
		Version:   version,
		Log:       logger,
	})

	// Background jobs; cancelled on shutdown.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	staleAfter := time.Duration(settingsService.StaleAfterSeconds()) * time.Second
	go serversService.RunSweeper(ctx, 15*time.Second, staleAfter, handlers.Broadcast)
	go metricsService.RunCompactor(ctx, 5*time.Minute)

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
