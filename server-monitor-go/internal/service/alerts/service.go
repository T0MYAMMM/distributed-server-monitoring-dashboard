// Package alerts holds the alerting foundation: it records alerts on status
// transitions and threshold breaches, delivers them through a Notifier, and
// serves the list/acknowledge API. It implements the servers.AlertSink
// interface so the servers service can emit without importing this package.
package alerts

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/thomasstefen/server-monitor/internal/domain"
)

// Repo is the alert persistence the service needs; sqlite.Store satisfies it.
type Repo interface {
	InsertAlert(a domain.Alert, when time.Time) (int64, error)
	ListAlerts(severity string, limit int) ([]domain.Alert, error)
	AcknowledgeAlert(id int64, when time.Time) (bool, error)
	UnacknowledgedThresholdExists(serverID string) (bool, error)
}

// Clock abstracts time for testable timestamps.
type Clock interface{ Now() time.Time }

// SystemClock is the production Clock.
type SystemClock struct{}

// Now returns the current time.
func (SystemClock) Now() time.Time { return time.Now() }

// Service implements the alert use cases and the servers.AlertSink hooks.
type Service struct {
	repo     Repo
	notifier Notifier
	clock    Clock
	log      *slog.Logger
	onEmit   func() // optional: refresh dashboards (e.g. WS snapshot) after an alert

	mu            sync.RWMutex
	diskThreshold float64
}

// New constructs the alert service. diskThreshold is the disk-usage percent that
// triggers a threshold alert; onEmit (may be nil) is invoked after each new
// alert so the caller can push a dashboard refresh.
func New(repo Repo, notifier Notifier, clock Clock, diskThreshold float64, onEmit func(), log *slog.Logger) *Service {
	if clock == nil {
		clock = SystemClock{}
	}
	if notifier == nil {
		notifier = NopNotifier{}
	}
	if log == nil {
		log = slog.Default()
	}
	return &Service{repo: repo, notifier: notifier, clock: clock, log: log, diskThreshold: diskThreshold, onEmit: onEmit}
}

// StatusChanged emits an alert for down (->stopped) and recovery
// (stopped->running) transitions. Other transitions (e.g. first connect from
// maintenance) are not alert-worthy.
func (s *Service) StatusChanged(serverID, serverName string, from, to domain.Status) {
	switch {
	case to == domain.StatusStopped:
		s.emit(domain.Alert{
			Type: domain.AlertStatusChange, ServerID: serverID, ServerName: serverName,
			Severity: domain.SeverityCritical,
			Message:  fmt.Sprintf("%s went down (no metrics)", serverName),
		})
	case to == domain.StatusRunning && from == domain.StatusStopped:
		s.emit(domain.Alert{
			Type: domain.AlertStatusChange, ServerID: serverID, ServerName: serverName,
			Severity: domain.SeverityInfo,
			Message:  fmt.Sprintf("%s recovered", serverName),
		})
	}
}

// Reported emits a threshold alert when disk usage exceeds the configured
// threshold, deduplicated so an open breach does not re-alert on every report.
func (s *Service) Reported(serverID, serverName string, disk float64) {
	s.mu.RLock()
	threshold := s.diskThreshold
	s.mu.RUnlock()
	if threshold <= 0 || disk <= threshold {
		return
	}
	exists, err := s.repo.UnacknowledgedThresholdExists(serverID)
	if err != nil {
		s.log.Error("threshold dedupe check", "err", err)
		return
	}
	if exists {
		return
	}
	s.emit(domain.Alert{
		Type: domain.AlertThreshold, ServerID: serverID, ServerName: serverName,
		Severity: domain.SeverityWarning,
		Message:  fmt.Sprintf("Disk %.0f%% on %s", disk, serverName),
	})
}

// emit persists the alert, notifies, and refreshes dashboards.
func (s *Service) emit(a domain.Alert) {
	now := s.clock.Now()
	id, err := s.repo.InsertAlert(a, now)
	if err != nil {
		s.log.Error("insert alert", "err", err)
		return
	}
	a.ID = id
	a.CreatedAt = now.UTC().Format("2006-01-02 15:04:05")
	s.log.Info("alert", "type", a.Type, "severity", a.Severity, "server", a.ServerName, "message", a.Message)
	s.notifier.Notify(a)
	if s.onEmit != nil {
		s.onEmit()
	}
}

// SetDiskThreshold updates the disk-usage percent that triggers a threshold
// alert, applied live (e.g. from the Settings page).
func (s *Service) SetDiskThreshold(v float64) {
	s.mu.Lock()
	s.diskThreshold = v
	s.mu.Unlock()
}

// List returns alerts, optionally filtered by severity and limited.
func (s *Service) List(severity string, limit int) ([]domain.Alert, error) {
	return s.repo.ListAlerts(severity, limit)
}

// Acknowledge marks an alert acknowledged, or returns domain.ErrNotFound if it
// does not exist or is already acknowledged.
func (s *Service) Acknowledge(id int64) error {
	ok, err := s.repo.AcknowledgeAlert(id, s.clock.Now())
	if err != nil {
		return err
	}
	if !ok {
		return domain.ErrNotFound
	}
	return nil
}
