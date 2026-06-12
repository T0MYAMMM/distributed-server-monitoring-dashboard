package postgres

import (
	"context"
	"os"
	"testing"

	"github.com/thomasstefen/server-monitor/internal/domain"
)

// These tests require a Postgres reachable at TEST_LOG_DATABASE_URL; they skip
// otherwise so the suite stays dependency-free in CI without Postgres.
func testStore(t *testing.T) *Store {
	t.Helper()
	url := os.Getenv("TEST_LOG_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_LOG_DATABASE_URL to run Postgres log-store tests")
	}
	st, err := Open(context.Background(), url)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(st.Close)
	// Start from a clean table for deterministic assertions.
	if _, err := st.pool.Exec(context.Background(), `TRUNCATE logs RESTART IDENTITY`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return st
}

func TestInsertAndQuery(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()
	id := domain.ServerID("web-1")

	lines := []domain.LogLine{
		{Ts: "2026-06-12T10:00:00Z", Level: "INFO", Module: "app", Message: "started", SourceFile: "/var/log/app.log"},
		{Ts: "2026-06-12T10:00:01Z", Level: "ERROR", Module: "db", Message: "connection refused", SourceFile: "/var/log/app.log"},
		{Ts: "2026-06-12T10:00:02Z", Level: "INFO", Module: "app", Message: "recovered", SourceFile: "/var/log/app.log"},
	}
	if err := st.InsertLogs(ctx, id, "web-1", lines); err != nil {
		t.Fatalf("InsertLogs: %v", err)
	}

	// All for the server, newest first.
	all, err := st.QueryLogs(ctx, domain.LogQuery{ServerID: id})
	if err != nil {
		t.Fatalf("QueryLogs: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("got %d lines want 3", len(all))
	}
	if all[0].Message != "recovered" {
		t.Errorf("newest-first ordering wrong: %q", all[0].Message)
	}

	// Level filter.
	errs, _ := st.QueryLogs(ctx, domain.LogQuery{ServerID: id, Level: "error"})
	if len(errs) != 1 || errs[0].Module != "db" {
		t.Errorf("level filter = %+v want one ERROR/db", errs)
	}

	// Search filter.
	found, _ := st.QueryLogs(ctx, domain.LogQuery{ServerID: id, Search: "refused"})
	if len(found) != 1 {
		t.Errorf("search = %d want 1", len(found))
	}

	// Tail: rows after the first id, ascending.
	tail, _ := st.QueryLogs(ctx, domain.LogQuery{ServerID: id, AfterID: all[len(all)-1].ID})
	if len(tail) != 2 || tail[0].ID >= tail[1].ID {
		t.Errorf("tail wrong: %+v", tail)
	}

	// Isolation by server.
	other, _ := st.QueryLogs(ctx, domain.LogQuery{ServerID: domain.ServerID("other")})
	if len(other) != 0 {
		t.Errorf("expected no rows for other server, got %d", len(other))
	}
}
