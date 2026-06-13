// Package logs holds the log use cases: ingest batches shipped by agents and
// query/tail them for the dashboard. Storage is an external database (Postgres)
// injected via a consumer-defined Store interface; the feature is enabled only
// when that store is configured.
package logs

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/thomasstefen/server-monitor/internal/domain"
)

// Store is the external log persistence the service needs; the postgres Store
// satisfies it.
type Store interface {
	InsertLogs(ctx context.Context, serverID, server string, lines []domain.LogLine) error
	QueryLogs(ctx context.Context, q domain.LogQuery) ([]domain.LogLine, error)
	Modules(ctx context.Context, serverID string) ([]string, error)
	LogVolume(ctx context.Context, serverID string, from, to time.Time, bucketSecs int) ([]domain.LogVolumePoint, error)
	TopModules(ctx context.Context, serverID string, from, to time.Time, limit int) ([]domain.ModuleStat, error)
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

// Modules returns the distinct module (app) names for a server.
func (s *Service) Modules(ctx context.Context, serverID string) ([]string, error) {
	return s.store.Modules(ctx, serverID)
}

// analyticsBuckets is the target number of time buckets for the volume series.
const analyticsBuckets = 48

// rangeWindow maps an API range token to [from, to) and a bucket width.
func rangeWindow(rng string) (from, to time.Time, bucketSecs int) {
	secs := int64(24 * 3600)
	switch rng {
	case "24h":
		secs = 24 * 3600
	case "7d":
		secs = 7 * 24 * 3600
	case "30d":
		secs = 30 * 24 * 3600
	case "90d":
		secs = 90 * 24 * 3600
	}
	to = time.Now().UTC()
	from = to.Add(-time.Duration(secs) * time.Second)
	bucketSecs = int(secs / analyticsBuckets)
	if bucketSecs < 60 {
		bucketSecs = 60
	}
	return from, to, bucketSecs
}

// Volume returns the per-bucket log volume by level over the range. serverID ""
// spans the fleet.
func (s *Service) Volume(ctx context.Context, serverID, rng string) ([]domain.LogVolumePoint, error) {
	from, to, bucket := rangeWindow(rng)
	return s.store.LogVolume(ctx, serverID, from, to, bucket)
}

// TopModules returns the busiest modules (with error counts) over the range.
func (s *Service) TopModules(ctx context.Context, serverID, rng string, limit int) ([]domain.ModuleStat, error) {
	from, to, _ := rangeWindow(rng)
	return s.store.TopModules(ctx, serverID, from, to, limit)
}
