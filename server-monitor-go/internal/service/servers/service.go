// Package servers holds the server-lifecycle business logic: ingest
// accept/reject, status transitions, ordering, deletion, and the staleness
// sweep. It depends only on the domain types and a Repo interface it defines
// itself (consumer-defined boundary), so it is unit-testable with a fake repo
// and a fake clock.
package servers

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/thomasstefen/server-monitor/internal/domain"
)

// Repo is the persistence the servers service needs. The sqlite Store satisfies
// it structurally; tests provide a fake.
type Repo interface {
	ListServers() ([]domain.Server, error)
	GetServer(id string) (domain.Server, bool, error)
	IsClientAllowed(name string) (bool, error)
	UpdateMetrics(in domain.Server) (changed bool, oldStatus domain.Status, err error)
	DeleteServer(id string) (bool, error)
	SetStatus(id string, status domain.Status) error
	SetOrder(id string, order int) error
	Heartbeat(id string) error
	AddClient(name string) error
	ClientExists(name string) (bool, error)
	ListClients() ([]domain.Client, error)
	MarkStaleStopped(staleAfter time.Duration) ([]string, error)
	RecordUnknownAgent(name, remoteAddr string, when time.Time) error
	ListUnknownAgents() ([]domain.UnknownAgent, error)
}

// Clock abstracts time so the sweep loop is testable without real delays.
type Clock interface{ Now() time.Time }

// AlertSink receives lifecycle events worth alerting on. It is implemented by
// the alerts service and injected at wiring time; the servers service depends
// only on this interface, not on the alerts package.
type AlertSink interface {
	StatusChanged(serverID, serverName string, from, to domain.Status)
	Reported(serverID, serverName string, disk float64)
}

// SystemClock is the production Clock.
type SystemClock struct{}

// Now returns the current time.
func (SystemClock) Now() time.Time { return time.Now() }

// Service implements the server-lifecycle use cases.
type Service struct {
	repo  Repo
	clock Clock
	log   *slog.Logger
	sink  AlertSink
}

// New constructs a Service. clock and log may be nil for sensible defaults.
func New(repo Repo, clock Clock, log *slog.Logger) *Service {
	if clock == nil {
		clock = SystemClock{}
	}
	if log == nil {
		log = slog.Default()
	}
	return &Service{repo: repo, clock: clock, log: log}
}

// SetAlertSink injects the alert sink at wiring time (optional). When unset, no
// alerts are emitted and behavior is unchanged.
func (s *Service) SetAlertSink(sink AlertSink) { s.sink = sink }

// List returns all servers in display order.
func (s *Service) List() ([]domain.Server, error) { return s.repo.ListServers() }

// Get returns one server or domain.ErrNotFound.
func (s *Service) Get(id string) (domain.Server, error) {
	sv, ok, err := s.repo.GetServer(id)
	if err != nil {
		return domain.Server{}, err
	}
	if !ok {
		return domain.Server{}, domain.ErrNotFound
	}
	return sv, nil
}

// Ingest applies an agent report. It returns domain.ErrInvalidInput for an
// empty name, domain.ErrNotAllowed when the name is not allow-listed (the 403
// case), and domain.ErrNotFound when no server row matched.
func (s *Service) Ingest(in domain.Server) error {
	if in.Name == "" {
		return domain.ErrInvalidInput
	}
	allowed, err := s.repo.IsClientAllowed(in.Name)
	if err != nil {
		return err
	}
	if !allowed {
		return domain.ErrNotAllowed
	}
	changed, old, err := s.repo.UpdateMetrics(in)
	if err != nil {
		return err
	}
	if !changed {
		return domain.ErrNotFound
	}
	if old != domain.StatusRunning {
		s.log.Info("server status transition", "name", in.Name, "from", old, "to", domain.StatusRunning)
	}
	if s.sink != nil {
		id := domain.ServerID(in.Name)
		if old == domain.StatusStopped {
			s.sink.StatusChanged(id, in.Name, domain.StatusStopped, domain.StatusRunning)
		}
		s.sink.Reported(id, in.Name, in.Disk)
	}
	return nil
}

// Delete removes a server and its allow-list entry, or domain.ErrNotFound.
func (s *Service) Delete(id string) error {
	ok, err := s.repo.DeleteServer(id)
	if err != nil {
		return err
	}
	if !ok {
		return domain.ErrNotFound
	}
	return nil
}

// ForceStatus sets a server's status to a validated value and returns the
// updated record.
func (s *Service) ForceStatus(id string, status domain.Status) (domain.Server, error) {
	if !status.Valid() {
		return domain.Server{}, domain.ErrInvalidInput
	}
	if err := s.repo.SetStatus(id, status); err != nil {
		return domain.Server{}, err
	}
	return s.Get(id)
}

// SetOrder updates a server's display order index.
func (s *Service) SetOrder(id string, order int) error { return s.repo.SetOrder(id, order) }

// Heartbeat records a lightweight liveness ping.
func (s *Service) Heartbeat(id string) error { return s.repo.Heartbeat(id) }

// AddClient registers a name on the allow-list. Empty -> ErrInvalidInput,
// already-registered -> ErrConflict.
func (s *Service) AddClient(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.ErrInvalidInput
	}
	exists, err := s.repo.ClientExists(name)
	if err != nil {
		return err
	}
	if exists {
		return domain.ErrConflict
	}
	return s.repo.AddClient(name)
}

// ListClients returns the allow-list entries.
func (s *Service) ListClients() ([]domain.Client, error) { return s.repo.ListClients() }

// IsAllowed reports whether a name is on the allow-list (used to gate log
// ingest the same way metrics ingest is gated).
func (s *Service) IsAllowed(name string) (bool, error) { return s.repo.IsClientAllowed(name) }

// RecordUnknownAgent notes a rejected ingest for admin observability.
func (s *Service) RecordUnknownAgent(name, remoteAddr string) error {
	return s.repo.RecordUnknownAgent(name, remoteAddr, s.clock.Now())
}

// UnknownAgents returns recently rejected agent names for the admin panel.
func (s *Service) UnknownAgents() ([]domain.UnknownAgent, error) {
	return s.repo.ListUnknownAgents()
}

// SweepStale flips servers silent for longer than staleAfter to stopped,
// returning the names that changed so the caller can broadcast.
func (s *Service) SweepStale(staleAfter time.Duration) ([]string, error) {
	changed, err := s.repo.MarkStaleStopped(staleAfter)
	if err != nil {
		return nil, err
	}
	for _, name := range changed {
		s.log.Info("server status transition", "name", name, "from", domain.StatusRunning, "to", domain.StatusStopped, "reason", "stale")
		if s.sink != nil {
			s.sink.StatusChanged(domain.ServerID(name), name, domain.StatusRunning, domain.StatusStopped)
		}
	}
	return changed, nil
}

// RunSweeper runs SweepStale every interval until ctx is cancelled, invoking
// onChange when any server's status flipped (so the caller can broadcast).
func (s *Service) RunSweeper(ctx context.Context, interval, staleAfter time.Duration, onChange func()) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	s.log.Debug("staleness sweeper started", "interval", interval, "stale_after", staleAfter, "at", s.clock.Now())
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			changed, err := s.SweepStale(staleAfter)
			if err != nil {
				s.log.Error("staleness sweep failed", "err", err)
				continue
			}
			if len(changed) > 0 && onChange != nil {
				onChange()
			}
		}
	}
}
