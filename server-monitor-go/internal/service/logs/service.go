// Package logs holds the log use cases: ingest batches shipped by agents and
// query/tail them for the dashboard. Storage is an external database (Postgres)
// injected via a consumer-defined Store interface; the feature is enabled only
// when that store is configured.
package logs

import (
	"context"
	"log/slog"
	"strings"

	"github.com/thomasstefen/server-monitor/internal/domain"
)

// Store is the external log persistence the service needs; the postgres Store
// satisfies it.
type Store interface {
	InsertLogs(ctx context.Context, serverID, server string, lines []domain.LogLine) error
	QueryLogs(ctx context.Context, q domain.LogQuery) ([]domain.LogLine, error)
	Close()
}

// Service implements the log use cases.
type Service struct {
	store Store
	log   *slog.Logger
}

// New constructs the log service. store may be nil, which disables the feature.
func New(store Store, log *slog.Logger) *Service {
	if log == nil {
		log = slog.Default()
	}
	return &Service{store: store, log: log}
}

// Enabled reports whether a log store is configured.
func (s *Service) Enabled() bool { return s.store != nil }

// Ingest stores a batch of log lines for a server, normalizing levels.
func (s *Service) Ingest(ctx context.Context, server string, lines []domain.LogLine) error {
	if len(lines) == 0 {
		return nil
	}
	for i := range lines {
		if lines[i].Level == "" {
			lines[i].Level = domain.LogInfo
		} else {
			lines[i].Level = strings.ToUpper(lines[i].Level)
		}
	}
	return s.store.InsertLogs(ctx, domain.ServerID(server), server, lines)
}

// Query returns log lines matching the filter.
func (s *Service) Query(ctx context.Context, q domain.LogQuery) ([]domain.LogLine, error) {
	return s.store.QueryLogs(ctx, q)
}
