// Characterization tests (Phase 0) for the persistence layer: the status
// lifecycle and the staleness sweep. They pin current behavior so the Phase 1
// move (including the planned Clock injection) stays behavior-preserving.
package store

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/thomasstefen/server-monitor/internal/models"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	st, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func TestAddClientCreatesPendingRow(t *testing.T) {
	st := newStore(t)
	if err := st.AddClient("web-1"); err != nil {
		t.Fatalf("AddClient: %v", err)
	}
	sv, ok, err := st.GetServer(ServerID("web-1"))
	if err != nil || !ok {
		t.Fatalf("GetServer: ok=%v err=%v", ok, err)
	}
	if sv.Status != models.StatusMaintenance {
		t.Errorf("new client status = %q want maintenance", sv.Status)
	}
	if sv.Location != "Pending" {
		t.Errorf("new client location = %q want Pending", sv.Location)
	}
	if allowed, _ := st.IsClientAllowed("web-1"); !allowed {
		t.Error("client should be allow-listed after AddClient")
	}
}

func TestUpdateMetricsMarksRunning(t *testing.T) {
	st := newStore(t)
	_ = st.AddClient("web-1")

	changed, old, err := st.UpdateMetrics(models.Server{Name: "web-1", CPU: 10})
	if err != nil {
		t.Fatalf("UpdateMetrics: %v", err)
	}
	if !changed {
		t.Fatal("UpdateMetrics changed = false want true")
	}
	if old != models.StatusMaintenance {
		t.Errorf("previous status = %q want maintenance", old)
	}
	sv, _, _ := st.GetServer(ServerID("web-1"))
	if sv.Status != models.StatusRunning {
		t.Errorf("status after update = %q want running", sv.Status)
	}

	// Unregistered/no-row name does not insert.
	changed, _, err = st.UpdateMetrics(models.Server{Name: "ghost"})
	if err != nil {
		t.Fatalf("UpdateMetrics ghost: %v", err)
	}
	if changed {
		t.Error("UpdateMetrics for unknown name should not change anything")
	}
}

// TestMarkStaleStopped exercises the sweep deterministically by controlling the
// staleAfter window rather than the wall clock: a negative window forces the
// cutoff into the future so a just-updated row is considered stale; a large
// positive window leaves it fresh.
func TestMarkStaleStopped(t *testing.T) {
	st := newStore(t)
	_ = st.AddClient("web-1")
	if _, _, err := st.UpdateMetrics(models.Server{Name: "web-1"}); err != nil {
		t.Fatalf("UpdateMetrics: %v", err)
	}

	// Fresh: a 1h window leaves the running row untouched.
	changed, err := st.MarkStaleStopped(1 * time.Hour)
	if err != nil {
		t.Fatalf("MarkStaleStopped fresh: %v", err)
	}
	if len(changed) != 0 {
		t.Fatalf("fresh sweep changed %v want none", changed)
	}
	if sv, _, _ := st.GetServer(ServerID("web-1")); sv.Status != models.StatusRunning {
		t.Errorf("status = %q want running after fresh sweep", sv.Status)
	}

	// Stale: a negative window pushes the cutoff past now, so the row flips.
	changed, err = st.MarkStaleStopped(-1 * time.Hour)
	if err != nil {
		t.Fatalf("MarkStaleStopped stale: %v", err)
	}
	if len(changed) != 1 || changed[0] != "web-1" {
		t.Fatalf("stale sweep changed %v want [web-1]", changed)
	}
	if sv, _, _ := st.GetServer(ServerID("web-1")); sv.Status != models.StatusStopped {
		t.Errorf("status = %q want stopped after stale sweep", sv.Status)
	}

	// Idempotent: already-stopped rows are not reported again.
	changed, _ = st.MarkStaleStopped(-1 * time.Hour)
	if len(changed) != 0 {
		t.Errorf("second sweep changed %v want none (already stopped)", changed)
	}
}

func TestHeartbeatSkipsMaintenance(t *testing.T) {
	st := newStore(t)
	_ = st.AddClient("web-1")
	id := ServerID("web-1")

	// In maintenance, heartbeat must not flip to running.
	if err := st.Heartbeat(id); err != nil {
		t.Fatalf("Heartbeat: %v", err)
	}
	if sv, _, _ := st.GetServer(id); sv.Status != models.StatusMaintenance {
		t.Errorf("status = %q want maintenance (heartbeat must skip it)", sv.Status)
	}

	// From stopped, heartbeat marks running.
	_ = st.SetStatus(id, models.StatusStopped)
	if err := st.Heartbeat(id); err != nil {
		t.Fatalf("Heartbeat: %v", err)
	}
	if sv, _, _ := st.GetServer(id); sv.Status != models.StatusRunning {
		t.Errorf("status = %q want running after heartbeat", sv.Status)
	}
}

func TestDeleteServerRemovesAllowList(t *testing.T) {
	st := newStore(t)
	_ = st.AddClient("web-1")
	id := ServerID("web-1")

	ok, err := st.DeleteServer(id)
	if err != nil || !ok {
		t.Fatalf("DeleteServer: ok=%v err=%v", ok, err)
	}
	if _, ok, _ := st.GetServer(id); ok {
		t.Error("server row should be gone after delete")
	}
	if allowed, _ := st.IsClientAllowed("web-1"); allowed {
		t.Error("allow-list entry should be gone after delete")
	}
	// Deleting a missing id reports not-found.
	if ok, _ := st.DeleteServer("does-not-exist"); ok {
		t.Error("deleting unknown id should report false")
	}
}
