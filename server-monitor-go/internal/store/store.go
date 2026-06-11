// Package store is the persistence layer: a thin, well-typed wrapper around
// SQLite that owns the schema and all queries. Server identity is keyed by
// name (the value an admin registers and the agent reports under); the public
// id is a stable md5(name) so the frontend can address rows.
package store

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/thomasstefen/server-monitor/internal/models"

	_ "modernc.org/sqlite" // pure-Go SQLite driver (no cgo), keeps a static binary
)

// timeLayout is the timestamp format stored in TEXT columns. It is
// lexicographically sortable and parseable, which keeps staleness comparisons
// simple and correct.
const timeLayout = "2006-01-02 15:04:05"

// Store wraps a SQLite database handle.
type Store struct {
	db *sql.DB
}

// Open connects to the SQLite database at path and initializes the schema.
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	// SQLite permits a single writer; serialize to avoid "database is locked".
	db.SetMaxOpenConns(1)

	s := &Store{db: db}
	if err := s.init(); err != nil {
		return nil, err
	}
	return s, nil
}

// Close releases the database handle.
func (s *Store) Close() error { return s.db.Close() }

// ServerID derives the stable public id for a client name.
func ServerID(name string) string {
	sum := md5.Sum([]byte(name))
	return hex.EncodeToString(sum[:])
}

func (s *Store) init() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS servers (
			id TEXT PRIMARY KEY,
			name TEXT UNIQUE,
			type TEXT,
			location TEXT,
			ip_address TEXT,
			hostname TEXT DEFAULT '',
			tailscale_ip TEXT DEFAULT '',
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
		// On startup any previously "running" server is unknown until its
		// agent reports again, so reset to stopped.
		`UPDATE servers SET status = 'stopped' WHERE status = 'running'`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("init schema: %w", err)
		}
	}
	return s.migrate()
}

// migrate brings an existing database up to the current schema by adding any
// columns introduced after it was first created. ADD COLUMN is idempotent here
// because we only add columns absent from the live table.
func (s *Store) migrate() error {
	existing, err := s.columns("servers")
	if err != nil {
		return err
	}
	adds := map[string]string{
		"hostname":     `ALTER TABLE servers ADD COLUMN hostname TEXT DEFAULT ''`,
		"tailscale_ip": `ALTER TABLE servers ADD COLUMN tailscale_ip TEXT DEFAULT ''`,
	}
	for col, ddl := range adds {
		if _, ok := existing[col]; ok {
			continue
		}
		if _, err := s.db.Exec(ddl); err != nil {
			return fmt.Errorf("migrate add %s: %w", col, err)
		}
	}
	return nil
}

// columns returns the set of column names for a table.
func (s *Store) columns(table string) (map[string]struct{}, error) {
	rows, err := s.db.Query(`SELECT name FROM pragma_table_info(?)`, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cols := make(map[string]struct{})
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		cols[name] = struct{}{}
	}
	return cols, rows.Err()
}

// serverColumns is the canonical projection used by list/get queries.
const serverColumns = `id, name, type, location, ip_address, hostname,
	tailscale_ip, status, uptime, network_in, network_out, cpu, memory, disk,
	os_type, cpu_info, total_memory, total_disk, order_index, first_seen,
	last_update`

func scanServer(row interface{ Scan(...any) error }) (models.Server, error) {
	var sv models.Server
	err := row.Scan(
		&sv.ID, &sv.Name, &sv.Type, &sv.Location, &sv.IPAddress, &sv.Hostname,
		&sv.TailscaleIP, &sv.Status, &sv.Uptime, &sv.NetworkIn, &sv.NetworkOut,
		&sv.CPU, &sv.Memory, &sv.Disk, &sv.OSType, &sv.CPUInfo, &sv.TotalMemory,
		&sv.TotalDisk, &sv.OrderIndex, &sv.FirstSeen, &sv.LastUpdate,
	)
	return sv, err
}

// ListServers returns every server ordered for display (highest order_index
// first, then oldest first as a stable tiebreaker).
func (s *Store) ListServers() ([]models.Server, error) {
	rows, err := s.db.Query(`SELECT ` + serverColumns + ` FROM servers
		ORDER BY order_index DESC, first_seen ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	servers := make([]models.Server, 0)
	for rows.Next() {
		sv, err := scanServer(rows)
		if err != nil {
			return nil, err
		}
		servers = append(servers, sv)
	}
	return servers, rows.Err()
}

// GetServer fetches a single server by its public id.
func (s *Store) GetServer(id string) (models.Server, bool, error) {
	row := s.db.QueryRow(`SELECT `+serverColumns+` FROM servers WHERE id = ?`, id)
	sv, err := scanServer(row)
	if err == sql.ErrNoRows {
		return models.Server{}, false, nil
	}
	if err != nil {
		return models.Server{}, false, err
	}
	return sv, true, nil
}

// IsClientAllowed reports whether a client name is on the allow-list.
func (s *Store) IsClientAllowed(name string) (bool, error) {
	var n string
	err := s.db.QueryRow(`SELECT name FROM allowed_clients WHERE name = ?`, name).Scan(&n)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

// UpdateMetrics applies an agent report to the matching server row (matched by
// name) and marks it running. Rows are only updated, never inserted here: a
// server must first be registered via AddClient. Returns false if no allowed
// row matched. The status transition (if any) is returned for logging.
func (s *Store) UpdateMetrics(in models.Server) (changed bool, oldStatus string, err error) {
	var prev string
	err = s.db.QueryRow(`SELECT status FROM servers WHERE name = ?`, in.Name).Scan(&prev)
	if err == sql.ErrNoRows {
		return false, "", nil
	}
	if err != nil {
		return false, "", err
	}

	now := time.Now().UTC().Format(timeLayout)
	res, err := s.db.Exec(`UPDATE servers SET
			type = ?, location = ?, ip_address = ?, hostname = ?,
			tailscale_ip = ?, status = 'running', uptime = ?, network_in = ?,
			network_out = ?, cpu = ?, memory = ?, disk = ?, os_type = ?,
			cpu_info = ?, total_memory = ?, total_disk = ?, last_update = ?
		WHERE name = ?`,
		nz(in.Type, "Unknown"), nz(in.Location, "UN"), nz(in.IPAddress, "127.0.0.1"),
		nz(in.Hostname, "N/A"), nz(in.TailscaleIP, "N/A"),
		in.Uptime, in.NetworkIn, in.NetworkOut, in.CPU, in.Memory, in.Disk,
		nz(in.OSType, "Unknown"), nz(in.CPUInfo, "N/A"), in.TotalMemory,
		in.TotalDisk, now, in.Name,
	)
	if err != nil {
		return false, prev, err
	}
	n, _ := res.RowsAffected()
	return n > 0, prev, nil
}

// Heartbeat refreshes last_update and marks a server running unless it is in
// maintenance. Used by the lightweight heartbeat endpoint.
func (s *Store) Heartbeat(id string) error {
	now := time.Now().UTC().Format(timeLayout)
	_, err := s.db.Exec(`UPDATE servers SET last_update = ?, status = 'running'
		WHERE id = ? AND status != 'maintenance'`, now, id)
	return err
}

// SetStatus forces a server's status by id.
func (s *Store) SetStatus(id, status string) error {
	now := time.Now().UTC().Format(timeLayout)
	_, err := s.db.Exec(`UPDATE servers SET status = ?, last_update = ? WHERE id = ?`,
		status, now, id)
	return err
}

// SetOrder updates a server's display order index.
func (s *Store) SetOrder(id string, order int) error {
	_, err := s.db.Exec(`UPDATE servers SET order_index = ? WHERE id = ?`, order, id)
	return err
}

// AddClient registers a name on the allow-list and creates its initial server
// row in maintenance ("Pending") state, replacing any prior records.
func (s *Store) AddClient(name string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM allowed_clients WHERE name = ?`, name); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM servers WHERE name = ?`, name); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO allowed_clients (name) VALUES (?)`, name); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO servers
		(id, name, type, location, ip_address, status, uptime, network_in,
		 network_out, cpu, memory, disk, os_type, cpu_info, total_memory,
		 total_disk, order_index)
		VALUES (?, ?, 'VPS', 'Pending', 'N/A', 'maintenance', 0, 0, 0, 0, 0, 0,
		        'Linux', 'N/A', 0, 0, 0)`,
		ServerID(name), name); err != nil {
		return err
	}
	return tx.Commit()
}

// ClientExists reports whether a client name is already registered.
func (s *Store) ClientExists(name string) (bool, error) {
	return s.IsClientAllowed(name)
}

// ListClients returns the allow-list entries.
func (s *Store) ListClients() ([]models.Client, error) {
	rows, err := s.db.Query(`SELECT name, created_at FROM allowed_clients ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	clients := make([]models.Client, 0)
	for rows.Next() {
		var c models.Client
		if err := rows.Scan(&c.Name, &c.CreatedAt); err != nil {
			return nil, err
		}
		clients = append(clients, c)
	}
	return clients, rows.Err()
}

// DeleteServer removes a server and its allow-list entry by id.
func (s *Store) DeleteServer(id string) (bool, error) {
	var name string
	err := s.db.QueryRow(`SELECT name FROM servers WHERE id = ?`, id).Scan(&name)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM servers WHERE id = ?`, id); err != nil {
		return false, err
	}
	if _, err := tx.Exec(`DELETE FROM allowed_clients WHERE name = ?`, name); err != nil {
		return false, err
	}
	return true, tx.Commit()
}

// MarkStaleStopped flips servers that have stopped reporting (running but
// silent for longer than staleAfter) to "stopped", returning the names that
// changed so the caller can log and broadcast.
func (s *Store) MarkStaleStopped(staleAfter time.Duration) ([]string, error) {
	cutoff := time.Now().UTC().Add(-staleAfter).Format(timeLayout)
	rows, err := s.db.Query(`SELECT id, name FROM servers
		WHERE status = 'running' AND last_update < ?`, cutoff)
	if err != nil {
		return nil, err
	}
	type idName struct{ id, name string }
	var stale []idName
	for rows.Next() {
		var v idName
		if err := rows.Scan(&v.id, &v.name); err != nil {
			rows.Close()
			return nil, err
		}
		stale = append(stale, v)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	changed := make([]string, 0, len(stale))
	for _, v := range stale {
		if _, err := s.db.Exec(`UPDATE servers SET status = 'stopped'
			WHERE id = ? AND status = 'running'`, v.id); err != nil {
			return changed, err
		}
		changed = append(changed, v.name)
	}
	return changed, nil
}

// --- admin auth ---

// IsInitialized reports whether an admin password has been set.
func (s *Store) IsInitialized() (bool, error) {
	var x int
	err := s.db.QueryRow(`SELECT is_initialized FROM admin_auth WHERE is_initialized = 1`).Scan(&x)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

// SetPasswordHash stores the admin password hash, creating or updating the row.
func (s *Store) SetPasswordHash(hash []byte) error {
	var id int
	err := s.db.QueryRow(`SELECT id FROM admin_auth LIMIT 1`).Scan(&id)
	if err == sql.ErrNoRows {
		_, err = s.db.Exec(`INSERT INTO admin_auth (password_hash, is_initialized)
			VALUES (?, 1)`, hash)
		return err
	}
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`UPDATE admin_auth SET password_hash = ?, is_initialized = 1
		WHERE id = ?`, hash, id)
	return err
}

// PasswordHash returns the stored admin password hash.
func (s *Store) PasswordHash() ([]byte, bool, error) {
	var hash []byte
	err := s.db.QueryRow(`SELECT password_hash FROM admin_auth
		WHERE is_initialized = 1`).Scan(&hash)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return hash, true, nil
}

// nz returns fallback when v is empty.
func nz(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}
