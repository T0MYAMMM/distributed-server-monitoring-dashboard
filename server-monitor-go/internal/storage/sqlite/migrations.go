package sqlite

import (
	"database/sql"
	"fmt"
)

// migration is one ordered, versioned schema change. Applied versions are
// recorded in schema_migrations so each migration runs exactly once and new
// feature tables are additive and traceable, replacing the previous ad-hoc
// "ALTER TABLE if the column is missing" approach.
type migration struct {
	version int
	name    string
	stmts   []string
}

// migrations is the full ordered history. Append new versions; never edit or
// reorder existing ones once shipped.
var migrations = []migration{
	{
		version: 1,
		name:    "initial schema",
		stmts: []string{
			`CREATE TABLE IF NOT EXISTS servers (
				id TEXT PRIMARY KEY,
				name TEXT UNIQUE,
				type TEXT,
				location TEXT,
				ip_address TEXT,
				status TEXT DEFAULT 'stopped',
				uptime INTEGER,
				network_in REAL,
				network_out REAL,
				cpu REAL,
				memory REAL,
				disk REAL,
				os_type TEXT,
				cpu_info TEXT,
				total_memory REAL,
				total_disk REAL,
				order_index INTEGER DEFAULT 0,
				first_seen TEXT DEFAULT (datetime('now')),
				last_update TEXT DEFAULT (datetime('now'))
			)`,
			`CREATE TABLE IF NOT EXISTS allowed_clients (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name TEXT UNIQUE NOT NULL,
				created_at TEXT DEFAULT (datetime('now'))
			)`,
			`CREATE INDEX IF NOT EXISTS idx_client_name ON allowed_clients(name)`,
			`CREATE TABLE IF NOT EXISTS admin_auth (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				password_hash BLOB NOT NULL,
				is_initialized INTEGER DEFAULT 0
			)`,
		},
	},
	{
		version: 2,
		name:    "add hostname and tailscale_ip",
		stmts: []string{
			`ALTER TABLE servers ADD COLUMN hostname TEXT DEFAULT ''`,
			`ALTER TABLE servers ADD COLUMN tailscale_ip TEXT DEFAULT ''`,
		},
	},
	{
		version: 3,
		name:    "metrics history (samples + 5m rollups)",
		stmts: []string{
			`CREATE TABLE metrics_samples (
				server_id TEXT NOT NULL,
				ts INTEGER NOT NULL,
				cpu REAL, mem REAL, disk REAL, net_in REAL, net_out REAL
			)`,
			`CREATE INDEX idx_metrics_samples_server_ts ON metrics_samples(server_id, ts)`,
			`CREATE TABLE metrics_rollup_5m (
				server_id TEXT NOT NULL,
				bucket INTEGER NOT NULL,
				cpu REAL, mem REAL, disk REAL, net_in REAL, net_out REAL,
				PRIMARY KEY (server_id, bucket)
			)`,
		},
	},
	{
		version: 4,
		name:    "unknown agents (rejected ingest observability)",
		stmts: []string{
			`CREATE TABLE unknown_agents (
				name TEXT PRIMARY KEY,
				remote_addr TEXT DEFAULT '',
				last_seen TEXT DEFAULT '',
				count INTEGER DEFAULT 0
			)`,
		},
	},
	{
		version: 5,
		name:    "alerts",
		stmts: []string{
			`CREATE TABLE alerts (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				type TEXT NOT NULL,
				server_id TEXT DEFAULT '',
				server_name TEXT DEFAULT '',
				severity TEXT DEFAULT 'info',
				message TEXT DEFAULT '',
				created_at TEXT DEFAULT (datetime('now')),
				acknowledged_at TEXT DEFAULT ''
			)`,
			`CREATE INDEX idx_alerts_created ON alerts(created_at)`,
		},
	},
	{
		version: 6,
		name:    "settings (env-backed config editable in-app)",
		stmts: []string{
			`CREATE TABLE settings (
				key TEXT PRIMARY KEY,
				value TEXT NOT NULL DEFAULT '',
				updated_at TEXT DEFAULT (datetime('now'))
			)`,
		},
	},
	{
		version: 7,
		name:    "notification channels (alert delivery targets)",
		stmts: []string{
			`CREATE TABLE notification_channels (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				type TEXT NOT NULL,
				name TEXT NOT NULL DEFAULT '',
				config TEXT NOT NULL DEFAULT '{}',
				enabled INTEGER NOT NULL DEFAULT 1,
				last_status TEXT NOT NULL DEFAULT '',
				last_error TEXT NOT NULL DEFAULT '',
				last_delivery TEXT NOT NULL DEFAULT '',
				created_at TEXT DEFAULT (datetime('now'))
			)`,
		},
	},
	{
		version: 8,
		name:    "feedback (in-app submissions)",
		stmts: []string{
			`CREATE TABLE feedback (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				category TEXT NOT NULL DEFAULT 'general',
				message TEXT NOT NULL DEFAULT '',
				page TEXT NOT NULL DEFAULT '',
				created_at TEXT DEFAULT (datetime('now'))
			)`,
			`CREATE INDEX idx_feedback_created ON feedback(created_at)`,
		},
	},
}

// legacyBaselineVersion is the schema version produced by the pre-versioning
// ad-hoc init+migrate path (servers/allowed_clients/admin_auth plus the
// hostname/tailscale_ip columns). A database created before schema_migrations
// existed is stamped at this version without re-running those migrations, so an
// existing servers.db upgrades in place with no manual steps.
const legacyBaselineVersion = 2

// migrate brings the database up to the latest schema version.
func (s *Store) migrate() error {
	if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		applied_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	applied, err := s.appliedVersions()
	if err != nil {
		return err
	}

	// Baseline a pre-versioning database: if there are no recorded migrations
	// but a servers table already exists, the schema came from the legacy path.
	// Record the baseline versions as applied instead of re-running their DDL
	// (which would fail on the duplicate ADD COLUMN).
	if len(applied) == 0 {
		legacy, err := s.tableExists("servers")
		if err != nil {
			return err
		}
		if legacy {
			for _, m := range migrations {
				if m.version > legacyBaselineVersion {
					break
				}
				if err := s.recordMigration(s.db, m); err != nil {
					return err
				}
				applied[m.version] = true
			}
		}
	}

	for _, m := range migrations {
		if applied[m.version] {
			continue
		}
		tx, err := s.db.Begin()
		if err != nil {
			return err
		}
		for _, stmt := range m.stmts {
			if _, err := tx.Exec(stmt); err != nil {
				tx.Rollback()
				return fmt.Errorf("migration %d (%s): %w", m.version, m.name, err)
			}
		}
		if err := s.recordMigration(tx, m); err != nil {
			tx.Rollback()
			return fmt.Errorf("record migration %d: %w", m.version, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %d: %w", m.version, err)
		}
	}
	return nil
}

// execer is satisfied by both *sql.DB and *sql.Tx so recordMigration can run
// inside or outside a transaction.
type execer interface {
	Exec(query string, args ...any) (sql.Result, error)
}

func (s *Store) recordMigration(e execer, m migration) error {
	_, err := e.Exec(`INSERT INTO schema_migrations (version, name) VALUES (?, ?)`,
		m.version, m.name)
	return err
}

func (s *Store) appliedVersions() (map[int]bool, error) {
	rows, err := s.db.Query(`SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[int]bool)
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out[v] = true
	}
	return out, rows.Err()
}

func (s *Store) tableExists(name string) (bool, error) {
	var found string
	err := s.db.QueryRow(
		`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, name).Scan(&found)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}
