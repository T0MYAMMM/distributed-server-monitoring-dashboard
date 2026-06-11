package alerts

import (
	"errors"
	"testing"
	"time"

	"github.com/thomasstefen/server-monitor/internal/domain"
)

type fakeRepo struct {
	inserted        []domain.Alert
	thresholdExists bool
	ackResult       bool
}

func (f *fakeRepo) InsertAlert(a domain.Alert, when time.Time) (int64, error) {
	f.inserted = append(f.inserted, a)
	return int64(len(f.inserted)), nil
}
func (f *fakeRepo) ListAlerts(severity string, limit int) ([]domain.Alert, error) {
	return f.inserted, nil
}
func (f *fakeRepo) AcknowledgeAlert(id int64, when time.Time) (bool, error) { return f.ackResult, nil }
func (f *fakeRepo) UnacknowledgedThresholdExists(serverID string) (bool, error) {
	return f.thresholdExists, nil
}

type captureNotifier struct{ count int }

func (c *captureNotifier) Notify(domain.Alert) { c.count++ }

func TestStatusChangeAlerts(t *testing.T) {
	tests := []struct {
		name     string
		from, to domain.Status
		wantEmit bool
		wantSev  string
	}{
		{"down", domain.StatusRunning, domain.StatusStopped, true, domain.SeverityCritical},
		{"recovery", domain.StatusStopped, domain.StatusRunning, true, domain.SeverityInfo},
		{"first connect", domain.StatusMaintenance, domain.StatusRunning, false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{}
			notif := &captureNotifier{}
			emitted := 0
			svc := New(repo, notif, SystemClock{}, 90, func() { emitted++ }, nil)
			svc.StatusChanged("id1", "web-1", tt.from, tt.to)

			if tt.wantEmit {
				if len(repo.inserted) != 1 {
					t.Fatalf("inserted %d want 1", len(repo.inserted))
				}
				if repo.inserted[0].Severity != tt.wantSev {
					t.Errorf("severity = %q want %q", repo.inserted[0].Severity, tt.wantSev)
				}
				if notif.count != 1 || emitted != 1 {
					t.Errorf("notify=%d onEmit=%d want 1/1", notif.count, emitted)
				}
			} else if len(repo.inserted) != 0 {
				t.Errorf("expected no alert, got %d", len(repo.inserted))
			}
		})
	}
}

func TestThresholdAlertDedup(t *testing.T) {
	repo := &fakeRepo{}
	svc := New(repo, &captureNotifier{}, SystemClock{}, 90, nil, nil)

	// Below threshold: nothing.
	svc.Reported("id1", "db-1", 80)
	if len(repo.inserted) != 0 {
		t.Fatalf("below threshold emitted %d", len(repo.inserted))
	}
	// Above threshold, no open alert: one emission.
	svc.Reported("id1", "db-1", 95)
	if len(repo.inserted) != 1 || repo.inserted[0].Type != domain.AlertThreshold {
		t.Fatalf("threshold breach: inserted %+v", repo.inserted)
	}
	// Above threshold again with an open alert: deduped.
	repo.thresholdExists = true
	svc.Reported("id1", "db-1", 96)
	if len(repo.inserted) != 1 {
		t.Errorf("dedupe failed: inserted %d want 1", len(repo.inserted))
	}
}

func TestAcknowledge(t *testing.T) {
	repo := &fakeRepo{ackResult: true}
	svc := New(repo, nil, SystemClock{}, 90, nil, nil)
	if err := svc.Acknowledge(1); err != nil {
		t.Errorf("ack known: %v", err)
	}
	repo.ackResult = false
	if err := svc.Acknowledge(999); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("ack unknown err = %v want ErrNotFound", err)
	}
}
