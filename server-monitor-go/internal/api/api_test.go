// Package api_test holds characterization tests written in Phase 0 of the
// refactor. They pin the *current* HTTP contract so the Phase 1 restructure can
// be proven behavior-preserving. Where an assertion encodes a behavior the
// refactor will deliberately change later, it is called out in a comment.
package api_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/thomasstefen/server-monitor/internal/api"
	"github.com/thomasstefen/server-monitor/internal/auth"
	"github.com/thomasstefen/server-monitor/internal/domain"
	"github.com/thomasstefen/server-monitor/internal/hub"
	"github.com/thomasstefen/server-monitor/internal/store"
)

const testSecret = "characterization-test-secret"

func setupAPI(t *testing.T) (srv *httptest.Server, st *store.Store, token string) {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	au := auth.New([]byte(testSecret))
	tok, err := au.IssueToken()
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	srv = httptest.NewServer(api.New(st, au, hub.New(), "").Handler())
	t.Cleanup(srv.Close)
	return srv, st, tok
}

// do performs an HTTP request with an optional JSON body and bearer token.
func do(t *testing.T, method, url string, body any, token string) (*http.Response, []byte) {
	t.Helper()
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, url, r)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do %s %s: %v", method, url, err)
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	return resp, out
}

func registerClient(t *testing.T, base, name string) {
	t.Helper()
	resp, body := do(t, http.MethodPost, base+"/api/clients", map[string]string{"name": name}, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("register %q: status %d body %s", name, resp.StatusCode, body)
	}
}

func TestIngestAcceptAndReject(t *testing.T) {
	srv, _, token := setupAPI(t)

	// Reject: unregistered name -> 403.
	if resp, _ := do(t, http.MethodPost, srv.URL+"/api/servers/update",
		domain.Server{Name: "ghost"}, ""); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("unregistered ingest: got %d want 403", resp.StatusCode)
	}

	// Reject: empty name -> 400.
	if resp, _ := do(t, http.MethodPost, srv.URL+"/api/servers/update",
		domain.Server{Name: ""}, ""); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("empty-name ingest: got %d want 400", resp.StatusCode)
	}

	// Accept: registered name -> 200, row goes running, nz() defaults applied.
	registerClient(t, srv.URL, "web-1")
	resp, _ := do(t, http.MethodPost, srv.URL+"/api/servers/update", domain.Server{
		Name: "web-1", CPU: 42.5, Memory: 60, // Type/Location/IPAddress left empty
	}, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("registered ingest: got %d want 200", resp.StatusCode)
	}

	id := store.ServerID("web-1")
	sv := getServer(t, srv.URL, id, token) // authed -> real IP visible
	if sv.Status != domain.StatusRunning {
		t.Errorf("status = %q want running", sv.Status)
	}
	if sv.CPU != 42.5 {
		t.Errorf("cpu = %v want 42.5", sv.CPU)
	}
	// nz() backfills.
	if sv.Type != "Unknown" || sv.Location != "UN" || sv.IPAddress != "127.0.0.1" {
		t.Errorf("nz defaults not applied: type=%q location=%q ip=%q", sv.Type, sv.Location, sv.IPAddress)
	}
}

func TestMaskingListAndGet(t *testing.T) {
	srv, _, token := setupAPI(t)
	registerClient(t, srv.URL, "web-1")
	do(t, http.MethodPost, srv.URL+"/api/servers/update", domain.Server{
		Name: "web-1", IPAddress: "203.0.113.7", TailscaleIP: "100.64.0.5", Hostname: "web1.local",
	}, "")
	id := store.ServerID("web-1")

	// Anonymous: public ip masked.
	anon := getServer(t, srv.URL, id, "")
	if anon.IPAddress != "***.***.***.**" {
		t.Errorf("anon ip = %q want masked", anon.IPAddress)
	}
	// CHARACTERIZATION OF CURRENT BEHAVIOR (decision D5 will flip these two):
	// today tailscale_ip and hostname are NOT masked for anonymous viewers.
	if anon.TailscaleIP != "100.64.0.5" {
		t.Errorf("anon tailscale_ip = %q; current behavior is unmasked", anon.TailscaleIP)
	}
	if anon.Hostname != "web1.local" {
		t.Errorf("anon hostname = %q; current behavior is unmasked", anon.Hostname)
	}

	// Admin: real public ip visible.
	if admin := getServer(t, srv.URL, id, token); admin.IPAddress != "203.0.113.7" {
		t.Errorf("admin ip = %q want real", admin.IPAddress)
	}

	// List endpoint masks the same way.
	resp, body := do(t, http.MethodGet, srv.URL+"/api/servers", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list: %d", resp.StatusCode)
	}
	var list []domain.Server
	if err := json.Unmarshal(body, &list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(list) != 1 || list[0].IPAddress != "***.***.***.**" {
		t.Errorf("list anon masking wrong: %+v", list)
	}
}

func TestAuthFlow(t *testing.T) {
	srv, _, _ := setupAPI(t)

	status := func() bool {
		_, body := do(t, http.MethodGet, srv.URL+"/api/auth/status", nil, "")
		var v struct {
			Initialized bool `json:"initialized"`
		}
		json.Unmarshal(body, &v)
		return v.Initialized
	}

	if status() {
		t.Fatal("fresh store should be uninitialized")
	}
	if resp, _ := do(t, http.MethodPost, srv.URL+"/api/auth/initialize",
		map[string]string{"password": "pw1"}, ""); resp.StatusCode != http.StatusOK {
		t.Fatalf("initialize: %d", resp.StatusCode)
	}
	if !status() {
		t.Fatal("should be initialized after init")
	}
	// Re-initialize is rejected.
	if resp, _ := do(t, http.MethodPost, srv.URL+"/api/auth/initialize",
		map[string]string{"password": "pw1"}, ""); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("re-initialize: got %d want 400", resp.StatusCode)
	}
	// Wrong password.
	if resp, _ := do(t, http.MethodPost, srv.URL+"/api/auth/login",
		map[string]string{"password": "nope"}, ""); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("bad login: got %d want 401", resp.StatusCode)
	}
	// Correct password yields a token.
	resp, body := do(t, http.MethodPost, srv.URL+"/api/auth/login",
		map[string]string{"password": "pw1"}, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login: %d", resp.StatusCode)
	}
	var tok struct {
		Token string `json:"token"`
	}
	json.Unmarshal(body, &tok)
	if tok.Token == "" {
		t.Fatal("login returned empty token")
	}
	// Reset password: wrong old rejected, correct accepted.
	if resp, _ := do(t, http.MethodPost, srv.URL+"/api/auth/reset-password",
		map[string]string{"oldPassword": "wrong", "newPassword": "pw2"}, ""); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("reset wrong-old: got %d want 401", resp.StatusCode)
	}
	if resp, _ := do(t, http.MethodPost, srv.URL+"/api/auth/reset-password",
		map[string]string{"oldPassword": "pw1", "newPassword": "pw2"}, ""); resp.StatusCode != http.StatusOK {
		t.Fatalf("reset: %d", resp.StatusCode)
	}
	if resp, _ := do(t, http.MethodPost, srv.URL+"/api/auth/login",
		map[string]string{"password": "pw2"}, ""); resp.StatusCode != http.StatusOK {
		t.Fatalf("login with new password: %d", resp.StatusCode)
	}
}

func TestForceStatusAndOrder(t *testing.T) {
	srv, _, _ := setupAPI(t)
	registerClient(t, srv.URL, "web-1")
	id := store.ServerID("web-1")

	// Valid status.
	resp, body := do(t, http.MethodPut, srv.URL+"/api/servers/"+id+"/status",
		map[string]string{"status": "stopped"}, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("set status: %d %s", resp.StatusCode, body)
	}
	var sv domain.Server
	json.Unmarshal(body, &sv)
	if sv.Status != domain.StatusStopped {
		t.Errorf("status = %q want stopped", sv.Status)
	}
	// Invalid status enum.
	if resp, _ := do(t, http.MethodPut, srv.URL+"/api/servers/"+id+"/status",
		map[string]string{"status": "banana"}, ""); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid status: got %d want 400", resp.StatusCode)
	}
	// Order requires order_index.
	if resp, _ := do(t, http.MethodPut, srv.URL+"/api/servers/"+id+"/order",
		map[string]int{"order_index": 7}, ""); resp.StatusCode != http.StatusOK {
		t.Fatalf("set order: %d", resp.StatusCode)
	}
	if resp, _ := do(t, http.MethodPut, srv.URL+"/api/servers/"+id+"/order",
		map[string]string{}, ""); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("missing order_index: got %d want 400", resp.StatusCode)
	}
}

func TestDeleteRemovesServerAndAllowList(t *testing.T) {
	srv, _, _ := setupAPI(t)
	registerClient(t, srv.URL, "web-1")
	id := store.ServerID("web-1")

	if resp, _ := do(t, http.MethodDelete, srv.URL+"/api/servers/"+id, nil, ""); resp.StatusCode != http.StatusOK {
		t.Fatalf("delete: %d", resp.StatusCode)
	}
	if resp, _ := do(t, http.MethodGet, srv.URL+"/api/servers/"+id, nil, ""); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("get after delete: got %d want 404", resp.StatusCode)
	}
	// Allow-list entry removed too: ingest now rejected.
	if resp, _ := do(t, http.MethodPost, srv.URL+"/api/servers/update",
		domain.Server{Name: "web-1"}, ""); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("ingest after delete: got %d want 403", resp.StatusCode)
	}
}

func TestClients(t *testing.T) {
	srv, _, _ := setupAPI(t)
	registerClient(t, srv.URL, "web-1")

	// Duplicate registration is rejected.
	if resp, _ := do(t, http.MethodPost, srv.URL+"/api/clients",
		map[string]string{"name": "web-1"}, ""); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("duplicate client: got %d want 400", resp.StatusCode)
	}
	// List contains the client.
	_, body := do(t, http.MethodGet, srv.URL+"/api/clients", nil, "")
	var clients []domain.Client
	json.Unmarshal(body, &clients)
	if len(clients) != 1 || clients[0].Name != "web-1" {
		t.Errorf("clients = %+v want [web-1]", clients)
	}
}

func TestCORSPreflightAndHealthz(t *testing.T) {
	srv, _, _ := setupAPI(t)

	resp, _ := do(t, http.MethodOptions, srv.URL+"/api/servers", nil, "")
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("OPTIONS: got %d want 204", resp.StatusCode)
	}
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("CORS origin = %q want *", got)
	}

	resp, body := do(t, http.MethodGet, srv.URL+"/healthz", nil, "")
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), `"ok"`) {
		t.Errorf("healthz: %d %s", resp.StatusCode, body)
	}
}

func TestWebSocketSnapshotAndBroadcast(t *testing.T) {
	srv, _, _ := setupAPI(t)
	registerClient(t, srv.URL, "web-1")

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/api/ws/dashboard"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, http.Header{})
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	// First frame: current snapshot (IP-masked).
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, msg, err := conn.ReadMessage(); err != nil {
		t.Fatalf("read snapshot: %v", err)
	} else {
		var snap []domain.Server
		if err := json.Unmarshal(msg, &snap); err != nil {
			t.Fatalf("decode snapshot: %v", err)
		}
	}

	// An accepted report triggers a broadcast frame carrying the update.
	do(t, http.MethodPost, srv.URL+"/api/servers/update",
		domain.Server{Name: "web-1", IPAddress: "203.0.113.7", CPU: 99}, "")

	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read broadcast: %v", err)
	}
	var pushed []domain.Server
	if err := json.Unmarshal(msg, &pushed); err != nil {
		t.Fatalf("decode broadcast: %v", err)
	}
	if len(pushed) != 1 || pushed[0].CPU != 99 {
		t.Errorf("broadcast payload = %+v want web-1 cpu=99", pushed)
	}
	// WS frames are always masked, even though no auth is possible on the socket.
	if pushed[0].IPAddress != "***.***.***.**" {
		t.Errorf("ws ip = %q want masked", pushed[0].IPAddress)
	}
}

// getServer fetches and decodes a single server, failing the test on non-200.
func getServer(t *testing.T, base, id, token string) domain.Server {
	t.Helper()
	resp, body := do(t, http.MethodGet, base+"/api/servers/"+url.PathEscape(id), nil, token)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get server %s: status %d body %s", id, resp.StatusCode, body)
	}
	var sv domain.Server
	if err := json.Unmarshal(body, &sv); err != nil {
		t.Fatalf("decode server: %v", err)
	}
	return sv
}
