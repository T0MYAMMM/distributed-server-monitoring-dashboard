package metrics_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/thomasstefen/server-monitor/internal/domain"
	metricssvc "github.com/thomasstefen/server-monitor/internal/service/metrics"
	"github.com/thomasstefen/server-monitor/internal/storage/sqlite"
)

type fakeClock struct{ t time.Time }

func (f *fakeClock) Now() time.Time { return f.t }

func newStore(t *testing.T) *sqlite.Store {
	t.Helper()
	st, err := sqlite.Open(filepath.Join(t.TempDir(), "m.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func TestHistoryDownsamplesAndCaps(t *testing.T) {
	st := newStore(t)
	base := time.Unix(1_700_000_000, 0).UTC()
	now := base.Unix()
	id := domain.ServerID("web-1")

	// 1000 raw samples at 2s spacing (2000s span, inside the 1h window).
	for i := 0; i < 1000; i++ {
		if err := st.InsertSample(id, domain.MetricSample{Ts: now - int64(i*2), CPU: float64(i % 100)}); err != nil {
			t.Fatalf("InsertSample: %v", err)
		}
	}

	svc := metricssvc.New(st, &fakeClock{base}, nil)
	hist, err := svc.History(id, "1h")
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	if len(hist) == 0 || len(hist) > 500 {
		t.Fatalf("history len = %d, want 1..500", len(hist))
	}
	for i := 1; i < len(hist); i++ {
		if hist[i].Ts < hist[i-1].Ts {
			t.Fatalf("history not sorted ascending at %d", i)
		}
	}
}

func TestSummaryComputesDeltas(t *testing.T) {
	st := newStore(t)
	base := time.Unix(1_700_000_000, 0).UTC()
	now := base.Unix()
	id := domain.ServerID("web-1")

	if err := st.AddClient("web-1"); err != nil {
		t.Fatalf("AddClient: %v", err)
	}
	if err := st.SetStatus(id, domain.StatusRunning); err != nil {
		t.Fatalf("SetStatus: %v", err)
	}

	// Current window (last hour): cpu ~80. Previous window: cpu ~20.
	st.InsertSample(id, domain.MetricSample{Ts: now - 100, CPU: 80, Memory: 50})
	st.InsertSample(id, domain.MetricSample{Ts: now - 200, CPU: 80, Memory: 50})
	st.InsertSample(id, domain.MetricSample{Ts: now - 3700, CPU: 20, Memory: 30})
	st.InsertSample(id, domain.MetricSample{Ts: now - 3800, CPU: 20, Memory: 30})

	svc := metricssvc.New(st, &fakeClock{base}, nil)
	sum, err := svc.Summary("1h")
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	if sum.TotalServers != 1 || sum.ActiveServers != 1 {
		t.Errorf("counts: total=%d active=%d want 1/1", sum.TotalServers, sum.ActiveServers)
	}
	if sum.CPU.Value != 80 {
		t.Errorf("cpu value = %v want 80", sum.CPU.Value)
	}
	if sum.CPU.Delta != 60 { // 80 current - 20 previous
		t.Errorf("cpu delta = %v want 60", sum.CPU.Delta)
	}
}

func TestCompactRollsUpAndPrunesRaw(t *testing.T) {
	st := newStore(t)
	base := time.Unix(1_700_000_000, 0).UTC()
	now := base.Unix()
	id := domain.ServerID("web-1")

	oldTs := now - 50*3600 // 50h ago, beyond the 48h raw retention
	st.InsertSample(id, domain.MetricSample{Ts: oldTs, CPU: 42})
	st.InsertSample(id, domain.MetricSample{Ts: now - 60, CPU: 10}) // recent, retained

	svc := metricssvc.New(st, &fakeClock{base}, nil)
	if err := svc.Compact(); err != nil {
		t.Fatalf("Compact: %v", err)
	}

	// Old raw pruned.
	raw, _ := st.RawSeries(id, oldTs-300, oldTs+300, 300)
	if len(raw) != 0 {
		t.Errorf("old raw not pruned: %v", raw)
	}
	// But a rollup bucket survives for the old data.
	roll, _ := st.RollupSeries(id, oldTs-300, oldTs+600, 300)
	if len(roll) == 0 {
		t.Errorf("expected a rollup bucket for the pruned old sample")
	}
	// Recent raw retained.
	recent, _ := st.RawSeries(id, now-300, now+1, 300)
	if len(recent) == 0 {
		t.Errorf("recent raw should be retained")
	}
}
