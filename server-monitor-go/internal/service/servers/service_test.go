package servers

import (
	"errors"
	"testing"
	"time"

	"github.com/thomasstefen/server-monitor/internal/domain"
)

// fakeRepo is an in-memory Repo for service unit tests. Behavior is controlled
// by per-method hooks so each test sets only what it needs.
type fakeRepo struct {
	allowed         map[string]bool
	servers         map[string]domain.Server
	updateOld       domain.Status
	updateChange    bool
	clientExists    bool
	staleNames      []string
	unknownRecorded int
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{allowed: map[string]bool{}, servers: map[string]domain.Server{}}
}

func (f *fakeRepo) ListServers() ([]domain.Server, error) { return nil, nil }
func (f *fakeRepo) GetServer(id string) (domain.Server, bool, error) {
	sv, ok := f.servers[id]
	return sv, ok, nil
}
func (f *fakeRepo) IsClientAllowed(name string) (bool, error) { return f.allowed[name], nil }
func (f *fakeRepo) UpdateMetrics(in domain.Server) (bool, domain.Status, error) {
	return f.updateChange, f.updateOld, nil
}
func (f *fakeRepo) DeleteServer(id string) (bool, error) {
	_, ok := f.servers[id]
	return ok, nil
}
func (f *fakeRepo) SetStatus(id string, status domain.Status) error {
	sv := f.servers[id]
	sv.ID, sv.Status = id, status
	f.servers[id] = sv
	return nil
}
func (f *fakeRepo) SetOrder(id string, order int) error                { return nil }
func (f *fakeRepo) Heartbeat(id string) error                          { return nil }
func (f *fakeRepo) AddClient(name string) error                        { return nil }
func (f *fakeRepo) ClientExists(name string) (bool, error)             { return f.clientExists, nil }
func (f *fakeRepo) ListClients() ([]domain.Client, error)              { return nil, nil }
func (f *fakeRepo) MarkStaleStopped(d time.Duration) ([]string, error) { return f.staleNames, nil }
func (f *fakeRepo) RecordUnknownAgent(name, remoteAddr string, when time.Time) error {
	f.unknownRecorded++
	return nil
}
func (f *fakeRepo) ListUnknownAgents() ([]domain.UnknownAgent, error) { return nil, nil }

func TestIngest(t *testing.T) {
	tests := []struct {
		name    string
		in      domain.Server
		allowed bool
		change  bool
		old     domain.Status
		want    error
	}{
		{"empty name", domain.Server{}, false, false, "", domain.ErrInvalidInput},
		{"not allow-listed", domain.Server{Name: "ghost"}, false, false, "", domain.ErrNotAllowed},
		{"no row", domain.Server{Name: "web-1"}, true, false, "", domain.ErrNotFound},
		{"accepted from maintenance", domain.Server{Name: "web-1"}, true, true, domain.StatusMaintenance, nil},
		{"accepted already running", domain.Server{Name: "web-1"}, true, true, domain.StatusRunning, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRepo()
			repo.allowed[tt.in.Name] = tt.allowed
			repo.updateChange = tt.change
			repo.updateOld = tt.old
			svc := New(repo, SystemClock{}, nil)
			if err := svc.Ingest(tt.in); !errors.Is(err, tt.want) {
				t.Errorf("Ingest err = %v want %v", err, tt.want)
			}
		})
	}
}

func TestAddClient(t *testing.T) {
	repo := newFakeRepo()
	svc := New(repo, nil, nil)

	if err := svc.AddClient("   "); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("blank name err = %v want ErrInvalidInput", err)
	}
	repo.clientExists = true
	if err := svc.AddClient("web-1"); !errors.Is(err, domain.ErrConflict) {
		t.Errorf("duplicate err = %v want ErrConflict", err)
	}
	repo.clientExists = false
	if err := svc.AddClient("web-1"); err != nil {
		t.Errorf("valid add err = %v want nil", err)
	}
}

func TestForceStatusValidation(t *testing.T) {
	repo := newFakeRepo()
	repo.servers["id1"] = domain.Server{ID: "id1"}
	svc := New(repo, nil, nil)

	if _, err := svc.ForceStatus("id1", "banana"); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("invalid status err = %v want ErrInvalidInput", err)
	}
	sv, err := svc.ForceStatus("id1", domain.StatusStopped)
	if err != nil {
		t.Fatalf("ForceStatus: %v", err)
	}
	if sv.Status != domain.StatusStopped {
		t.Errorf("status = %q want stopped", sv.Status)
	}
}

func TestSweepStale(t *testing.T) {
	repo := newFakeRepo()
	repo.staleNames = []string{"a", "b"}
	svc := New(repo, nil, nil)
	changed, err := svc.SweepStale(30 * time.Second)
	if err != nil {
		t.Fatalf("SweepStale: %v", err)
	}
	if len(changed) != 2 {
		t.Errorf("changed = %v want 2 names", changed)
	}
}
