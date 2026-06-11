package sqlite

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// TestMigrateFreshRecordsAllVersions verifies a brand-new database ends up with
// every migration recorded and the expected columns present.
func TestMigrateFreshRecordsAllVersions(t *testing.T) {
	st := newStore(t)

	applied, err := st.appliedVersions()
	if err != nil {
		t.Fatalf("appliedVersions: %v", err)
	}
	for _, m := range migrations {
		if !applied[m.version] {
			t.Errorf("migration %d (%s) not recorded on fresh DB", m.version, m.name)
		}
	}
	// hostname/tailscale_ip (added by migration 2) must be usable.
	if err := st.AddClient("web-1"); err != nil {
		t.Fatalf("AddClient: %v", err)
	}
	if _, ok, err := st.GetServer(ServerID("web-1")); err != nil || !ok {
		t.Fatalf("GetServer after migrate: ok=%v err=%v", ok, err)
	}
}

// TestMigrateBaselinesLegacyDatabase simulates an existing servers.db created by
// the pre-versioning code path (all columns present, no schema_migrations
// table) and verifies it upgrades in place without re-running the baseline DDL.
func TestMigrateBaselinesLegacyDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.db")

	// Build a legacy schema directly: servers already has hostname/tailscale_ip
	// and there is no schema_migrations table.
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open raw: %v", err)
	}
	// Defaults mirror what the legacy AddClient/UpdateMetrics paths always
	// populated, so a minimal INSERT below yields a fully non-null row.
	legacyDDL := []string{
		`CREATE TABLE servers (
			id TEXT PRIMARY KEY, name TEXT UNIQUE, type TEXT DEFAULT '',
			location TEXT DEFAULT '', ip_address TEXT DEFAULT '',
			status TEXT DEFAULT 'stopped', uptime INTEGER DEFAULT 0,
			network_in REAL DEFAULT 0, network_out REAL DEFAULT 0,
			cpu REAL DEFAULT 0, memory REAL DEFAULT 0, disk REAL DEFAULT 0,
			os_type TEXT DEFAULT '', cpu_info TEXT DEFAULT '',
			total_memory REAL DEFAULT 0, total_disk REAL DEFAULT 0,
			order_index INTEGER DEFAULT 0,
			first_seen TEXT DEFAULT (datetime('now')),
			last_update TEXT DEFAULT (datetime('now')),
			hostname TEXT DEFAULT '', tailscale_ip TEXT DEFAULT ''
		)`,
		`CREATE TABLE allowed_clients (
			id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT UNIQUE NOT NULL,
			created_at TEXT DEFAULT (datetime('now'))
		)`,
		`CREATE TABLE admin_auth (
			id INTEGER PRIMARY KEY AUTOINCREMENT, password_hash BLOB NOT NULL,
			is_initialized INTEGER DEFAULT 0
		)`,
		`INSERT INTO allowed_clients (name) VALUES ('legacy-1')`,
		`INSERT INTO servers (id, name, status) VALUES ('abc', 'legacy-1', 'running')`,
	}
	for _, stmt := range legacyDDL {
		if _, err := raw.Exec(stmt); err != nil {
			t.Fatalf("legacy DDL: %v", err)
		}
	}
	raw.Close()

	// Open through the migrator: it must baseline rather than fail on the
	// duplicate ADD COLUMN, preserve existing data, and reset the stale running
	// row to stopped.
	st, err := Open(path)
	if err != nil {
		t.Fatalf("Open legacy DB: %v", err)
	}
	defer st.Close()

	applied, err := st.appliedVersions()
	if err != nil {
		t.Fatalf("appliedVersions: %v", err)
	}
	if !applied[1] || !applied[2] {
		t.Errorf("legacy DB not baselined: applied=%v", applied)
	}
	// Existing allow-list entry survived.
	if ok, _ := st.IsClientAllowed("legacy-1"); !ok {
		t.Error("legacy allow-list entry lost across migration")
	}
	// Startup reset flipped the running row to stopped.
	if sv, ok, _ := st.GetServer("abc"); !ok || sv.Status != "stopped" {
		t.Errorf("startup reset failed: ok=%v status=%q", ok, sv.Status)
	}
}
