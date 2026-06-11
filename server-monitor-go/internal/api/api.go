// Package api wires HTTP routes to the store, auth, and broadcast hub. It
// exposes the same REST contract the Next.js frontend and agents expect, plus
// a WebSocket endpoint for real-time dashboard updates.
package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"

	"github.com/thomasstefen/server-monitor/internal/auth"
	"github.com/thomasstefen/server-monitor/internal/domain"
	"github.com/thomasstefen/server-monitor/internal/hub"
	"github.com/thomasstefen/server-monitor/internal/store"
)

// maskedIP is shown to unauthenticated viewers instead of real addresses.
const maskedIP = "***.***.***.**"

// API holds the dependencies shared by all handlers.
type API struct {
	store     *store.Store
	auth      *auth.Auth
	hub       *hub.Hub
	agentsDir string
}

// New constructs an API. agentsDir holds prebuilt agent binaries to serve at
// /download/<file>; pass "" to disable the download endpoint.
func New(s *store.Store, a *auth.Auth, h *hub.Hub, agentsDir string) *API {
	return &API{store: s, auth: a, hub: h, agentsDir: agentsDir}
}

// Handler builds the HTTP handler with all routes mounted under /api and CORS
// applied. Uses the Go 1.22+ method-aware ServeMux, so no router dependency.
func (a *API) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/servers", a.listServers)
	mux.HandleFunc("POST /api/servers/update", a.updateServer)
	mux.HandleFunc("GET /api/servers/{id}", a.getServer)
	mux.HandleFunc("DELETE /api/servers/{id}", a.deleteServer)
	mux.HandleFunc("PUT /api/servers/{id}/status", a.setStatus)
	mux.HandleFunc("PUT /api/servers/{id}/order", a.setOrder)
	mux.HandleFunc("POST /api/servers/{id}/heartbeat", a.heartbeat)

	mux.HandleFunc("GET /api/clients", a.listClients)
	mux.HandleFunc("POST /api/clients", a.addClient)

	mux.HandleFunc("GET /api/auth/status", a.authStatus)
	mux.HandleFunc("POST /api/auth/initialize", a.initialize)
	mux.HandleFunc("POST /api/auth/login", a.login)
	mux.HandleFunc("POST /api/auth/reset-password", a.resetPassword)

	mux.HandleFunc("GET /api/ws/dashboard", a.dashboardWS)

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Serve prebuilt agent binaries so tailnet hosts can self-install.
	if a.agentsDir != "" {
		fs := http.FileServer(http.Dir(a.agentsDir))
		mux.Handle("GET /download/", http.StripPrefix("/download/", fs))
	}

	return withCORS(mux)
}

// --- servers ---

func (a *API) listServers(w http.ResponseWriter, r *http.Request) {
	servers, err := a.store.ListServers()
	if err != nil {
		serverError(w, err)
		return
	}
	if !a.isAuthed(r) {
		for i := range servers {
			servers[i].IPAddress = maskedIP
		}
	}
	writeJSON(w, http.StatusOK, servers)
}

func (a *API) getServer(w http.ResponseWriter, r *http.Request) {
	sv, ok, err := a.store.GetServer(r.PathValue("id"))
	if err != nil {
		serverError(w, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "Server not found")
		return
	}
	if !a.isAuthed(r) {
		sv.IPAddress = maskedIP
	}
	writeJSON(w, http.StatusOK, sv)
}

// updateServer ingests an agent metrics report (REST path). Only allow-listed
// clients are accepted; on success the fresh snapshot is broadcast to
// dashboards for real-time updates.
func (a *API) updateServer(w http.ResponseWriter, r *http.Request) {
	var in domain.Server
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid data")
		return
	}
	if in.Name == "" {
		writeError(w, http.StatusBadRequest, "Invalid data")
		return
	}

	allowed, err := a.store.IsClientAllowed(in.Name)
	if err != nil {
		serverError(w, err)
		return
	}
	if !allowed {
		writeError(w, http.StatusForbidden, "Client not allowed")
		return
	}

	changed, old, err := a.store.UpdateMetrics(in)
	if err != nil {
		serverError(w, err)
		return
	}
	if !changed {
		writeError(w, http.StatusNotFound, "Server not found")
		return
	}
	if old != domain.StatusRunning {
		log.Printf("server %q status: %s -> running", in.Name, old)
	}

	a.broadcastSnapshot()
	writeJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func (a *API) deleteServer(w http.ResponseWriter, r *http.Request) {
	ok, err := a.store.DeleteServer(r.PathValue("id"))
	if err != nil {
		serverError(w, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "Server not found")
		return
	}
	a.broadcastSnapshot()
	writeJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func (a *API) setStatus(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Status domain.Status `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid data")
		return
	}
	switch body.Status {
	case domain.StatusRunning, domain.StatusStopped, domain.StatusMaintenance:
	default:
		writeError(w, http.StatusBadRequest, "Invalid status")
		return
	}
	if err := a.store.SetStatus(r.PathValue("id"), body.Status); err != nil {
		serverError(w, err)
		return
	}
	sv, ok, err := a.store.GetServer(r.PathValue("id"))
	if err != nil {
		serverError(w, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "Server not found")
		return
	}
	a.broadcastSnapshot()
	writeJSON(w, http.StatusOK, sv)
}

func (a *API) setOrder(w http.ResponseWriter, r *http.Request) {
	var body struct {
		OrderIndex *int `json:"order_index"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.OrderIndex == nil {
		writeError(w, http.StatusBadRequest, "Order index is required")
		return
	}
	if err := a.store.SetOrder(r.PathValue("id"), *body.OrderIndex); err != nil {
		serverError(w, err)
		return
	}
	a.broadcastSnapshot()
	writeJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func (a *API) heartbeat(w http.ResponseWriter, r *http.Request) {
	if err := a.store.Heartbeat(r.PathValue("id")); err != nil {
		serverError(w, err)
		return
	}
	a.broadcastSnapshot()
	writeJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// --- clients ---

func (a *API) listClients(w http.ResponseWriter, r *http.Request) {
	clients, err := a.store.ListClients()
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, clients)
}

func (a *API) addClient(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Name) == "" {
		writeError(w, http.StatusBadRequest, "Client name is required")
		return
	}
	name := strings.TrimSpace(body.Name)

	exists, err := a.store.ClientExists(name)
	if err != nil {
		serverError(w, err)
		return
	}
	if exists {
		writeError(w, http.StatusBadRequest, "Client already exists")
		return
	}
	if err := a.store.AddClient(name); err != nil {
		serverError(w, err)
		return
	}
	a.broadcastSnapshot()
	writeJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// --- auth ---

func (a *API) authStatus(w http.ResponseWriter, r *http.Request) {
	init, err := a.store.IsInitialized()
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"initialized": init})
}

func (a *API) initialize(w http.ResponseWriter, r *http.Request) {
	init, err := a.store.IsInitialized()
	if err != nil {
		serverError(w, err)
		return
	}
	if init {
		writeError(w, http.StatusBadRequest, "Already initialized")
		return
	}
	var body struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Password == "" {
		writeError(w, http.StatusBadRequest, "Password is required")
		return
	}
	hash, err := a.auth.Hash(body.Password)
	if err != nil {
		serverError(w, err)
		return
	}
	if err := a.store.SetPasswordHash(hash); err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (a *API) login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Password == "" {
		writeError(w, http.StatusBadRequest, "Password is required")
		return
	}
	hash, ok, err := a.store.PasswordHash()
	if err != nil {
		serverError(w, err)
		return
	}
	if !ok || !a.auth.Check(hash, body.Password) {
		writeError(w, http.StatusUnauthorized, "Invalid password")
		return
	}
	token, err := a.auth.IssueToken()
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}

func (a *API) resetPassword(w http.ResponseWriter, r *http.Request) {
	var body struct {
		OldPassword string `json:"oldPassword"`
		NewPassword string `json:"newPassword"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil ||
		body.OldPassword == "" || body.NewPassword == "" {
		writeError(w, http.StatusBadRequest, "Both old and new passwords are required")
		return
	}
	hash, ok, err := a.store.PasswordHash()
	if err != nil {
		serverError(w, err)
		return
	}
	if !ok || !a.auth.Check(hash, body.OldPassword) {
		writeError(w, http.StatusUnauthorized, "Current password is incorrect")
		return
	}
	newHash, err := a.auth.Hash(body.NewPassword)
	if err != nil {
		serverError(w, err)
		return
	}
	if err := a.store.SetPasswordHash(newHash); err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// --- websocket ---

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // trusted tailnet
}

// dashboardWS upgrades the connection, sends the current snapshot immediately,
// then keeps the socket registered so it receives broadcasts until it closes.
func (a *API) dashboardWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	if servers, err := a.store.ListServers(); err == nil {
		// Dashboard sockets are unauthenticated; mask IPs.
		for i := range servers {
			servers[i].IPAddress = maskedIP
		}
		if b, err := json.Marshal(servers); err == nil {
			_ = conn.WriteMessage(websocket.TextMessage, b)
		}
	}
	a.hub.Add(conn) // blocks until the connection closes
}

// BroadcastSnapshot pushes the current (IP-masked) server list to dashboards.
// Exported for use by the background monitor.
func (a *API) BroadcastSnapshot() { a.broadcastSnapshot() }

// broadcastSnapshot pushes the current (IP-masked) server list to dashboards.
func (a *API) broadcastSnapshot() {
	if a.hub.Count() == 0 {
		return
	}
	servers, err := a.store.ListServers()
	if err != nil {
		return
	}
	for i := range servers {
		servers[i].IPAddress = maskedIP
	}
	if b, err := json.Marshal(servers); err == nil {
		a.hub.Broadcast(b)
	}
}

// isAuthed reports whether the request carries a valid admin bearer token.
func (a *API) isAuthed(r *http.Request) bool {
	h := r.Header.Get("Authorization")
	parts := strings.SplitN(h, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return false
	}
	return a.auth.Valid(parts[1])
}
