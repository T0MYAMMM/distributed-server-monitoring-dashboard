// Package postgres is the external log store: a pgx-backed (pure Go, no cgo)
// Postgres implementation for high-volume log lines, kept separate from the
// hub's SQLite so the core monitoring stays a zero-dependency single binary
// while logs live on a dedicated database (e.g. the home-db server). It is only
// constructed when LOG_DATABASE_URL is configured.
package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/thomasstefen/server-monitor/internal/domain"
)

const tsLayout = time.RFC3339

// Store wraps a Postgres connection pool.
type Store struct {
	pool *pgxpool.Pool
}

// Open connects to the log database and ensures the schema exists.
func Open(ctx context.Context, url string) (*Store, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("connect log db: %w", err)
	}
	s := &Store{pool: pool}
	if err := s.ensureSchema(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return s, nil
}

// Close releases the pool.
func (s *Store) Close() { s.pool.Close() }

func (s *Store) ensureSchema(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS logs (
			id BIGSERIAL PRIMARY KEY,
			server_id TEXT NOT NULL,
			server TEXT NOT NULL,
			ts TIMESTAMPTZ NOT NULL,
			level TEXT NOT NULL DEFAULT 'INFO',
			module TEXT NOT NULL DEFAULT '',
			message TEXT NOT NULL DEFAULT '',
			source_file TEXT NOT NULL DEFAULT '',
			received_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_logs_server_id_id ON logs(server_id, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_logs_ts ON logs(ts DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_logs_server_module ON logs(server_id, module)`,
	}
	for _, stmt := range stmts {
		if _, err := s.pool.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("ensure log schema: %w", err)
		}
	}
	return nil
}

// InsertLogs appends a batch of log lines for a server.
func (s *Store) InsertLogs(ctx context.Context, serverID, server string, lines []domain.LogLine) error {
	if len(lines) == 0 {
		return nil
	}
	rows := make([][]any, 0, len(lines))
	for _, l := range lines {
		ts := parseTs(l.Ts)
		rows = append(rows, []any{
			serverID, server, ts, nz(l.Level, domain.LogInfo), l.Module, l.Message, l.SourceFile,
		})
	}
	_, err := s.pool.CopyFrom(ctx,
		pgx.Identifier{"logs"},
		[]string{"server_id", "server", "ts", "level", "module", "message", "source_file"},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return fmt.Errorf("insert logs: %w", err)
	}
	return nil
}

// QueryLogs returns log lines matching the filter. Normal queries return newest
// first (capped by Limit); tail queries (AfterID > 0) return rows in ascending
// id order so the stream can advance its cursor.
func (s *Store) QueryLogs(ctx context.Context, q domain.LogQuery) ([]domain.LogLine, error) {
	var where []string
	var args []any
	add := func(cond string, val any) {
		args = append(args, val)
		where = append(where, fmt.Sprintf(cond, len(args)))
	}
	add("server_id = $%d", q.ServerID)
	if q.Level != "" {
		add("level = $%d", strings.ToUpper(q.Level))
	}
	if len(q.Modules) > 0 {
		add("module = ANY($%d)", q.Modules)
	}
	if q.Search != "" {
		// Keyword grep on the message only; module has its own filter.
		add("message ILIKE $%d", "%"+q.Search+"%")
	}
	if q.File != "" {
		add("source_file = $%d", q.File)
	}
	if q.Since != "" {
		add("ts >= $%d", parseTs(q.Since))
	}
	if q.Until != "" {
		add("ts <= $%d", parseTs(q.Until))
	}
	order := "ORDER BY id DESC"
	if q.AfterID > 0 {
		add("id > $%d", q.AfterID)
		order = "ORDER BY id ASC"
	}
	limit := q.Limit
	if limit <= 0 || limit > 2000 {
		limit = 500
	}
	args = append(args, limit)
	sql := fmt.Sprintf(
		`SELECT id, server, ts, level, module, message, source_file FROM logs WHERE %s %s LIMIT $%d`,
		strings.Join(where, " AND "), order, len(args),
	)

	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query logs: %w", err)
	}
	defer rows.Close()
	out := make([]domain.LogLine, 0)
	for rows.Next() {
		var l domain.LogLine
		var ts time.Time
		if err := rows.Scan(&l.ID, &l.Server, &ts, &l.Level, &l.Module, &l.Message, &l.SourceFile); err != nil {
			return nil, err
		}
		l.Ts = ts.UTC().Format(tsLayout)
		out = append(out, l)
	}
	return out, rows.Err()
}

// Modules returns the distinct module (app) names seen for a server, so the UI
// can offer a per-node module filter.
func (s *Store) Modules(ctx context.Context, serverID string) ([]string, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT DISTINCT module FROM logs WHERE server_id = $1 AND module <> '' ORDER BY module LIMIT 200`,
		serverID)
	if err != nil {
		return nil, fmt.Errorf("query modules: %w", err)
	}
	defer rows.Close()
	out := make([]string, 0)
	for rows.Next() {
		var m string
		if err := rows.Scan(&m); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// LogVolume returns per-bucket log counts split by level over [from, to).
// serverID "" aggregates across the whole fleet; bucketSecs sets the bucket
// width in seconds.
func (s *Store) LogVolume(ctx context.Context, serverID string, from, to time.Time, bucketSecs int) ([]domain.LogVolumePoint, error) {
	if bucketSecs < 1 {
		bucketSecs = 3600
	}
	args := []any{bucketSecs, from.UTC(), to.UTC()}
	where := "ts >= $2 AND ts < $3"
	if serverID != "" {
		args = append(args, serverID)
		where += fmt.Sprintf(" AND server_id = $%d", len(args))
	}
	sql := `SELECT (floor(extract(epoch from ts)/$1)*$1)::bigint AS bucket,
			SUM(CASE WHEN level='DEBUG' THEN 1 ELSE 0 END),
			SUM(CASE WHEN level='INFO'  THEN 1 ELSE 0 END),
			SUM(CASE WHEN level='WARN'  THEN 1 ELSE 0 END),
			SUM(CASE WHEN level='ERROR' THEN 1 ELSE 0 END)
		FROM logs WHERE ` + where + `
		GROUP BY bucket ORDER BY bucket`
	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("log volume: %w", err)
	}
	defer rows.Close()
	out := make([]domain.LogVolumePoint, 0)
	for rows.Next() {
		var p domain.LogVolumePoint
		if err := rows.Scan(&p.Ts, &p.Debug, &p.Info, &p.Warn, &p.Error); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// TopModules returns the busiest modules over [from, to) with their error
// counts, for the "top error sources" view. serverID "" spans the fleet.
func (s *Store) TopModules(ctx context.Context, serverID string, from, to time.Time, limit int) ([]domain.ModuleStat, error) {
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	args := []any{from.UTC(), to.UTC()}
	where := "ts >= $1 AND ts < $2 AND module <> ''"
	if serverID != "" {
		args = append(args, serverID)
		where += fmt.Sprintf(" AND server_id = $%d", len(args))
	}
	args = append(args, limit)
	sql := `SELECT module, COUNT(*) AS total,
			SUM(CASE WHEN level='ERROR' THEN 1 ELSE 0 END) AS errors
		FROM logs WHERE ` + where + `
		GROUP BY module ORDER BY total DESC LIMIT $` + fmt.Sprint(len(args))
	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("top modules: %w", err)
	}
	defer rows.Close()
	out := make([]domain.ModuleStat, 0)
	for rows.Next() {
		var m domain.ModuleStat
		if err := rows.Scan(&m.Module, &m.Total, &m.Errors); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func parseTs(s string) time.Time {
	if s == "" {
		return time.Now().UTC()
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC()
		}
	}
	return time.Now().UTC()
}

func nz(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}
