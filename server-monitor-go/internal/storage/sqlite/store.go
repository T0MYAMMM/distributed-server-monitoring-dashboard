// Package sqlite is the concrete persistence layer: a thin, well-typed wrapper
// around SQLite (modernc.org/sqlite, pure Go, no cgo) that owns the schema and
// all queries. It returns domain types and satisfies the repository interfaces
// the service layer defines. Server identity is keyed by name (the value an
// admin registers and the agent reports under); the public id is a stable
// md5(name) so the frontend can address rows.
//
// Operations such as AddClient and DeleteServer span the servers and
// allowed_clients tables transactionally, so they live on a single Store rather
// than being split across per-table repositories.
package sqlite

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/thomasstefen/server-monitor/internal/domain"

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

// Open connects to the SQLite database at path, runs versioned migrations, and
// resets any stale "running" rows left over from a previous process.
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	// SQLite permits a single writer; serialize to avoid "database is locked".
	db.SetMaxOpenConns(1)

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, err
	}
	// On startup any previously "running" server is unknown until its agent
	// reports again, so reset to stopped.
	if _, err := db.Exec(`UPDATE servers SET status = 'stopped' WHERE status = 'running'`); err != nil {
		return nil, fmt.Errorf("startup reset: %w", err)
	}
	return s, nil
}

// Close releases the database handle.
func (s *Store) Close() error { return s.db.Close() }

// ServerID derives the stable public id for a client name. It delegates to
// domain.ServerID so the id rule lives in one place.
func ServerID(name string) string { return domain.ServerID(name) }

// serverColumns is the canonical projection used by list/get queries.
const serverColumns = `id, name, type, location, ip_address, hostname,
	tailscale_ip, status, uptime, network_in, network_out, cpu, memory, disk,
	os_type, cpu_info, total_memory, total_disk, order_index, first_seen,
	last_update`

func scanServer(row interface{ Scan(...any) error }) (domain.Server, error) {
	var sv domain.Server
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
func (s *Store) ListServers() ([]domain.Server, error) {
	rows, err := s.db.Query(`SELECT ` + serverColumns + ` FROM servers
		ORDER BY order_index DESC, first_seen ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	servers := make([]domain.Server, 0)
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
func (s *Store) GetServer(id string) (domain.Server, bool, error) {
	row := s.db.QueryRow(`SELECT `+serverColumns+` FROM servers WHERE id = ?`, id)
	sv, err := scanServer(row)
	if err == sql.ErrNoRows {
		return domain.Server{}, false, nil
	}
	if err != nil {
		return domain.Server{}, false, err
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
func (s *Store) UpdateMetrics(in domain.Server) (changed bool, oldStatus domain.Status, err error) {
	var prev domain.Status
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
func (s *Store) SetStatus(id string, status domain.Status) error {
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
func (s *Store) ListClients() ([]domain.Client, error) {
	rows, err := s.db.Query(`SELECT name, created_at FROM allowed_clients ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	clients := make([]domain.Client, 0)
	for rows.Next() {
		var c domain.Client
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

// --- unknown agents (ingest observability) ---

// RecordUnknownAgent upserts a rejected ingest: it increments the seen count
// and refreshes the remote address and last-seen time for that name.
func (s *Store) RecordUnknownAgent(name, remoteAddr string, when time.Time) error {
	ts := when.UTC().Format(timeLayout)
	_, err := s.db.Exec(`INSERT INTO unknown_agents (name, remote_addr, last_seen, count)
		VALUES (?, ?, ?, 1)
		ON CONFLICT(name) DO UPDATE SET
			remote_addr = excluded.remote_addr,
			last_seen = excluded.last_seen,
			count = count + 1`,
		name, remoteAddr, ts)
	return err
}

// ListUnknownAgents returns rejected-ingest entries, most recently seen first.
func (s *Store) ListUnknownAgents() ([]domain.UnknownAgent, error) {
	rows, err := s.db.Query(`SELECT name, remote_addr, last_seen, count
		FROM unknown_agents ORDER BY last_seen DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.UnknownAgent, 0)
	for rows.Next() {
		var u domain.UnknownAgent
		if err := rows.Scan(&u.Name, &u.RemoteAddr, &u.LastSeen, &u.Count); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
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
