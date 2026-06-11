// Package httpapi wires HTTP routes to the service layer. Handlers stay thin:
// decode the request, call a service, encode the result, and translate domain
// errors through the single mapping point in errors.go. It serves the legacy
// /api/... contract and the canonical /api/v1/... surface from the same
// handlers.
package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/gorilla/websocket"

	"github.com/thomasstefen/server-monitor/internal/domain"
	"github.com/thomasstefen/server-monitor/internal/masking"
	authsvc "github.com/thomasstefen/server-monitor/internal/service/auth"
	metricssvc "github.com/thomasstefen/server-monitor/internal/service/metrics"
	"github.com/thomasstefen/server-monitor/internal/service/servers"
	"github.com/thomasstefen/server-monitor/internal/transport/http/middleware"
	"github.com/thomasstefen/server-monitor/internal/transport/ws"
)

// Handlers holds the dependencies shared by all HTTP handlers.
type Handlers struct {
	servers   *servers.Service
	auth      *authsvc.Service
	metrics   *metricssvc.Service
	hub       *ws.Hub
	agentsDir string
	log       *slog.Logger
}

// New constructs the HTTP handlers. agentsDir holds prebuilt agent binaries to
// serve at /download/<file>; pass "" to disable the download endpoint.
func New(srv *servers.Service, a *authsvc.Service, m *metricssvc.Service, hub *ws.Hub, agentsDir string, log *slog.Logger) *Handlers {
	if log == nil {
		log = slog.Default()
	}
	return &Handlers{servers: srv, auth: a, metrics: m, hub: hub, agentsDir: agentsDir, log: log}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // trusted tailnet
}

// --- servers ---

func (h *Handlers) listServers(w http.ResponseWriter, r *http.Request) {
	list, err := h.servers.List()
	if err != nil {
		h.fail(w, err)
		return
	}
	if !h.isAuthed(r) {
		masking.All(list)
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *Handlers) getServer(w http.ResponseWriter, r *http.Request) {
	sv, err := h.servers.Get(r.PathValue("id"))
	if err != nil {
		h.fail(w, err)
		return
	}
	if !h.isAuthed(r) {
		masking.One(&sv)
	}
	writeJSON(w, http.StatusOK, sv)
}

// serverMetrics returns a downsampled time series for one server over the
// requested range (1h|6h|24h|7d), capped server-side at 500 points.
func (h *Handlers) serverMetrics(w http.ResponseWriter, r *http.Request) {
	series, err := h.metrics.History(r.PathValue("id"), r.URL.Query().Get("range"))
	if err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, series)
}

// metricsSummary returns fleet KPIs and previous-window deltas for the
// dashboard cards.
func (h *Handlers) metricsSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := h.metrics.Summary(r.URL.Query().Get("range"))
	if err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

// updateServer ingests an agent metrics report. Rejected (unregistered) reports
// are logged with the offending name and remote address before returning 403.
func (h *Handlers) updateServer(w http.ResponseWriter, r *http.Request) {
	var in domain.Server
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid data")
		return
	}
	if err := h.servers.Ingest(in); err != nil {
		if err == domain.ErrNotAllowed {
			h.log.Warn("rejected ingest report", "name", in.Name, "remote", r.RemoteAddr)
		}
		h.fail(w, err)
		return
	}
	// Record time-series history for charts/trends; non-fatal on failure.
	if h.metrics != nil {
		if err := h.metrics.Record(in); err != nil {
			h.log.Error("record metric sample", "name", in.Name, "err", err)
		}
	}
	h.broadcast()
	writeJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func (h *Handlers) deleteServer(w http.ResponseWriter, r *http.Request) {
	if err := h.servers.Delete(r.PathValue("id")); err != nil {
		h.fail(w, err)
		return
	}
	h.broadcast()
	writeJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func (h *Handlers) setStatus(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Status domain.Status `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid data")
		return
	}
	sv, err := h.servers.ForceStatus(r.PathValue("id"), body.Status)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.broadcast()
	writeJSON(w, http.StatusOK, sv)
}

func (h *Handlers) setOrder(w http.ResponseWriter, r *http.Request) {
	var body struct {
		OrderIndex *int `json:"order_index"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.OrderIndex == nil {
		writeError(w, http.StatusBadRequest, "Order index is required")
		return
	}
	if err := h.servers.SetOrder(r.PathValue("id"), *body.OrderIndex); err != nil {
		h.fail(w, err)
		return
	}
	h.broadcast()
	writeJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func (h *Handlers) heartbeat(w http.ResponseWriter, r *http.Request) {
	if err := h.servers.Heartbeat(r.PathValue("id")); err != nil {
		h.fail(w, err)
		return
	}
	h.broadcast()
	writeJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// --- clients ---

func (h *Handlers) listClients(w http.ResponseWriter, r *http.Request) {
	clients, err := h.servers.ListClients()
	if err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, clients)
}

func (h *Handlers) addClient(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Client name is required")
		return
	}
	if err := h.servers.AddClient(body.Name); err != nil {
		if err == domain.ErrInvalidInput {
			writeError(w, http.StatusBadRequest, "Client name is required")
			return
		}
		if err == domain.ErrConflict {
			writeError(w, http.StatusBadRequest, "Client already exists")
			return
		}
		h.fail(w, err)
		return
	}
	h.broadcast()
	writeJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// --- auth ---

func (h *Handlers) authStatus(w http.ResponseWriter, r *http.Request) {
	init, err := h.auth.Initialized()
	if err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"initialized": init})
}

func (h *Handlers) initialize(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Password is required")
		return
	}
	if err := h.auth.Initialize(body.Password); err != nil {
		if err == domain.ErrConflict {
			writeError(w, http.StatusBadRequest, "Already initialized")
			return
		}
		if err == domain.ErrInvalidInput {
			writeError(w, http.StatusBadRequest, "Password is required")
			return
		}
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (h *Handlers) login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Password is required")
		return
	}
	token, err := h.auth.Login(body.Password)
	if err != nil {
		if err == domain.ErrInvalidInput {
			writeError(w, http.StatusBadRequest, "Password is required")
			return
		}
		if err == domain.ErrUnauthorized {
			writeError(w, http.StatusUnauthorized, "Invalid password")
			return
		}
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}

func (h *Handlers) resetPassword(w http.ResponseWriter, r *http.Request) {
	var body struct {
		OldPassword string `json:"oldPassword"`
		NewPassword string `json:"newPassword"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Both old and new passwords are required")
		return
	}
	if err := h.auth.ResetPassword(body.OldPassword, body.NewPassword); err != nil {
		if err == domain.ErrInvalidInput {
			writeError(w, http.StatusBadRequest, "Both old and new passwords are required")
			return
		}
		if err == domain.ErrUnauthorized {
			writeError(w, http.StatusUnauthorized, "Current password is incorrect")
			return
		}
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// --- websocket ---

// dashboardWS upgrades the connection, sends the current (IP-masked) snapshot
// immediately, then keeps the socket registered so it receives broadcasts until
// it closes.
func (h *Handlers) dashboardWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	if list, err := h.servers.List(); err == nil {
		masking.All(list) // dashboard sockets are unauthenticated; mask IPs
		if b, err := json.Marshal(list); err == nil {
			_ = conn.WriteMessage(websocket.TextMessage, b)
		}
	}
	h.hub.Add(conn) // blocks until the connection closes
}

// broadcast pushes the current (IP-masked) server list to dashboards.
func (h *Handlers) broadcast() {
	if h.hub.Count() == 0 {
		return
	}
	list, err := h.servers.List()
	if err != nil {
		h.log.Error("broadcast snapshot", "err", err)
		return
	}
	masking.All(list)
	if b, err := json.Marshal(list); err == nil {
		h.hub.Broadcast(b)
	}
}

// Broadcast pushes a fresh snapshot to dashboards. Exported for the staleness
// sweeper's onChange callback.
func (h *Handlers) Broadcast() { h.broadcast() }

// isAuthed reports whether the request carries a valid admin bearer token.
func (h *Handlers) isAuthed(r *http.Request) bool {
	return h.auth.ValidToken(middleware.BearerToken(r))
}
