// Package httpapi_test holds characterization tests written in Phase 0 of the
// refactor (relocated from internal/api in Phase 1). They pin the *current*
// HTTP contract so the restructure can be proven behavior-preserving. Where an
// assertion encodes a behavior the refactor will deliberately change later, it
// is called out in a comment.
package httpapi_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/thomasstefen/server-monitor/internal/auth"
	"github.com/thomasstefen/server-monitor/internal/domain"
	alertssvc "github.com/thomasstefen/server-monitor/internal/service/alerts"
	authsvc "github.com/thomasstefen/server-monitor/internal/service/auth"
	metricssvc "github.com/thomasstefen/server-monitor/internal/service/metrics"
	"github.com/thomasstefen/server-monitor/internal/service/servers"
	"github.com/thomasstefen/server-monitor/internal/storage/sqlite"
	httpapi "github.com/thomasstefen/server-monitor/internal/transport/http"
	"github.com/thomasstefen/server-monitor/internal/transport/ws"
)

const testSecret = "characterization-test-secret"

func setupAPI(t *testing.T) (srv *httptest.Server, st *sqlite.Store, token string) {
	t.Helper()
	st, err := sqlite.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("sqlite.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	au := auth.New([]byte(testSecret))
	tok, err := au.IssueToken()
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	serversSvc := servers.New(st, servers.SystemClock{}, slog.Default())
	alertsSvc := alertssvc.New(st, nil, alertssvc.SystemClock{}, 90, nil, slog.Default())
	serversSvc.SetAlertSink(alertsSvc)
	h := httpapi.New(
		serversSvc,
		authsvc.New(st, au),
		metricssvc.New(st, metricssvc.SystemClock{}, slog.Default()),
		alertsSvc,
		ws.New(), "", slog.Default(),
	)
	srv = httptest.NewServer(h.Handler(slog.Default()))
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

	id := sqlite.ServerID("web-1")
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
	id := sqlite.ServerID("web-1")

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
	id := sqlite.ServerID("web-1")

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
	id := sqlite.ServerID("web-1")

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

// TestV1MutationAuth locks decision D3: mutating /api/v1 routes require a valid
// bearer token, while the legacy /api/ equivalents stay open.
func TestV1MutationAuth(t *testing.T) {
	srv, _, token := setupAPI(t)
	registerClient(t, srv.URL, "web-1")
	id := sqlite.ServerID("web-1")

	// v1 mutation without a token is rejected.
	if resp, _ := do(t, http.MethodDelete, srv.URL+"/api/v1/servers/"+id, nil, ""); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("v1 delete without token: got %d want 401", resp.StatusCode)
	}
	// v1 mutation with a valid token succeeds.
	if resp, _ := do(t, http.MethodDelete, srv.URL+"/api/v1/servers/"+id, nil, token); resp.StatusCode != http.StatusOK {
		t.Fatalf("v1 delete with token: got %d want 200", resp.StatusCode)
	}

	// Legacy mutation remains open (no token).
	registerClient(t, srv.URL, "web-2")
	id2 := sqlite.ServerID("web-2")
	if resp, _ := do(t, http.MethodDelete, srv.URL+"/api/servers/"+id2, nil, ""); resp.StatusCode != http.StatusOK {
		t.Fatalf("legacy delete without token: got %d want 200", resp.StatusCode)
	}
}

// TestV1ReadAliasMatchesLegacy confirms the canonical v1 read surface returns
// the same masked shape as the legacy path.
func TestV1ReadAliasMatchesLegacy(t *testing.T) {
	srv, _, _ := setupAPI(t)
	registerClient(t, srv.URL, "web-1")

	resp, body := do(t, http.MethodGet, srv.URL+"/api/v1/servers", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("v1 list: %d", resp.StatusCode)
	}
	var list []domain.Server
	if err := json.Unmarshal(body, &list); err != nil {
		t.Fatalf("decode v1 list: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("v1 list len = %d want 1", len(list))
	}
}

// TestMetricsEndpoints covers the B1 surface: an accepted report records a
// sample, the per-server history endpoint returns a series, and the fleet
// summary reports counts.
func TestMetricsEndpoints(t *testing.T) {
	srv, _, _ := setupAPI(t)
	registerClient(t, srv.URL, "web-1")
	do(t, http.MethodPost, srv.URL+"/api/servers/update",
		domain.Server{Name: "web-1", CPU: 55, Memory: 40, Disk: 30}, "")
	id := sqlite.ServerID("web-1")

	// Per-server history.
	resp, body := do(t, http.MethodGet, srv.URL+"/api/v1/servers/"+id+"/metrics?range=1h", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("metrics history: %d", resp.StatusCode)
	}
	var series []domain.MetricSample
	if err := json.Unmarshal(body, &series); err != nil {
		t.Fatalf("decode series: %v", err)
	}
	if len(series) == 0 {
		t.Error("expected at least one recorded sample in history")
	}

	// Fleet summary.
	resp, body = do(t, http.MethodGet, srv.URL+"/api/v1/metrics/summary?range=24h", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("metrics summary: %d", resp.StatusCode)
	}
	var sum domain.FleetSummary
	if err := json.Unmarshal(body, &sum); err != nil {
		t.Fatalf("decode summary: %v", err)
	}
	if sum.TotalServers != 1 {
		t.Errorf("summary total_servers = %d want 1", sum.TotalServers)
	}
}

// TestUnknownAgents covers B2: a rejected report is recorded and surfaced via
// the JWT-protected admin endpoint.
func TestUnknownAgents(t *testing.T) {
	srv, _, token := setupAPI(t)

	// Two rejected reports under the same unknown name.
	do(t, http.MethodPost, srv.URL+"/api/servers/update", domain.Server{Name: "mistyped"}, "")
	do(t, http.MethodPost, srv.URL+"/api/servers/update", domain.Server{Name: "mistyped"}, "")

	// Endpoint requires auth.
	if resp, _ := do(t, http.MethodGet, srv.URL+"/api/v1/admin/unknown-agents", nil, ""); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unknown-agents without token: got %d want 401", resp.StatusCode)
	}

	resp, body := do(t, http.MethodGet, srv.URL+"/api/v1/admin/unknown-agents", nil, token)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unknown-agents: %d", resp.StatusCode)
	}
	var agents []domain.UnknownAgent
	if err := json.Unmarshal(body, &agents); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(agents) != 1 || agents[0].Name != "mistyped" || agents[0].Count != 2 {
		t.Errorf("agents = %+v want one 'mistyped' with count 2", agents)
	}
}

// TestAlertsEndpoints covers B3: a disk-threshold breach on ingest creates an
// alert, the list endpoint returns it, and acknowledge requires auth.
func TestAlertsEndpoints(t *testing.T) {
	srv, _, token := setupAPI(t)
	registerClient(t, srv.URL, "db-1")

	// Report with disk over the 90% threshold -> threshold alert.
	do(t, http.MethodPost, srv.URL+"/api/servers/update",
		domain.Server{Name: "db-1", Disk: 95}, "")

	resp, body := do(t, http.MethodGet, srv.URL+"/api/v1/alerts", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list alerts: %d", resp.StatusCode)
	}
	var list []domain.Alert
	if err := json.Unmarshal(body, &list); err != nil {
		t.Fatalf("decode alerts: %v", err)
	}
	if len(list) != 1 || list[0].Type != domain.AlertThreshold || list[0].ServerName != "db-1" {
		t.Fatalf("alerts = %+v want one threshold alert for db-1", list)
	}
	if list[0].AcknowledgedAt != "" {
		t.Errorf("new alert should be unacknowledged")
	}
	id := list[0].ID

	// Acknowledge requires auth.
	path := srv.URL + "/api/v1/alerts/" + strconv.FormatInt(id, 10) + "/acknowledge"
	if resp, _ := do(t, http.MethodPost, path, nil, ""); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("ack without token: got %d want 401", resp.StatusCode)
	}
	if resp, _ := do(t, http.MethodPost, path, nil, token); resp.StatusCode != http.StatusOK {
		t.Fatalf("ack with token: got %d want 200", resp.StatusCode)
	}

	// Now acknowledged.
	_, body = do(t, http.MethodGet, srv.URL+"/api/v1/alerts", nil, "")
	json.Unmarshal(body, &list)
	if len(list) != 1 || list[0].AcknowledgedAt == "" {
		t.Errorf("alert should be acknowledged: %+v", list)
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
