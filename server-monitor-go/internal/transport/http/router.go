package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/thomasstefen/server-monitor/internal/transport/http/middleware"
)

// Handler builds the HTTP handler: every legacy /api/... path plus the
// canonical /api/v1/... surface, wrapped with the middleware chain. Legacy
// paths remain open for backward compatibility (deployed agents and the current
// frontend); the v1 mutating routes additionally require a valid bearer token.
func (h *Handlers) Handler(log *slog.Logger) http.Handler {
	mux := http.NewServeMux()

	// Reads, ingest, and auth endpoints: identical on both surfaces, open.
	// Ingest stays open because deployed agents cannot send a token.
	for _, p := range []string{"/api", "/api/v1"} {
		mux.HandleFunc("GET "+p+"/servers", h.listServers)
		mux.HandleFunc("GET "+p+"/servers/{id}", h.getServer)
		mux.HandleFunc("POST "+p+"/servers/update", h.updateServer)
		mux.HandleFunc("POST "+p+"/servers/{id}/heartbeat", h.heartbeat)
		mux.HandleFunc("GET "+p+"/clients", h.listClients)
		mux.HandleFunc("GET "+p+"/auth/status", h.authStatus)
		mux.HandleFunc("POST "+p+"/auth/initialize", h.initialize)
		mux.HandleFunc("POST "+p+"/auth/login", h.login)
		mux.HandleFunc("POST "+p+"/auth/reset-password", h.resetPassword)
		mux.HandleFunc("GET "+p+"/ws/dashboard", h.dashboardWS)
	}

	// Admin mutations. Legacy paths stay open (the tailnet is the boundary);
	// the v1 equivalents require auth (decision D3).
	mux.HandleFunc("DELETE /api/servers/{id}", h.deleteServer)
	mux.HandleFunc("PUT /api/servers/{id}/status", h.setStatus)
	mux.HandleFunc("PUT /api/servers/{id}/order", h.setOrder)
	mux.HandleFunc("POST /api/clients", h.addClient)

	requireAuth := middleware.RequireAuth(h.auth)
	mux.Handle("DELETE /api/v1/servers/{id}", requireAuth(http.HandlerFunc(h.deleteServer)))
	mux.Handle("PUT /api/v1/servers/{id}/status", requireAuth(http.HandlerFunc(h.setStatus)))
	mux.Handle("PUT /api/v1/servers/{id}/order", requireAuth(http.HandlerFunc(h.setOrder)))
	mux.Handle("POST /api/v1/clients", requireAuth(http.HandlerFunc(h.addClient)))
	mux.Handle("GET /api/v1/admin/unknown-agents", requireAuth(http.HandlerFunc(h.unknownAgents)))

	// Metrics history and fleet summary: new in v1 only, public reads.
	mux.HandleFunc("GET /api/v1/servers/{id}/metrics", h.serverMetrics)
	mux.HandleFunc("GET /api/v1/metrics/summary", h.metricsSummary)

	// Alerts: list is a public read (dashboard bell); acknowledge requires auth.
	mux.HandleFunc("GET /api/v1/alerts", h.listAlerts)
	mux.Handle("POST /api/v1/alerts/{id}/acknowledge", requireAuth(http.HandlerFunc(h.acknowledgeAlert)))

	mux.HandleFunc("GET /healthz", h.healthz)

	// Serve prebuilt agent binaries so tailnet hosts can self-install.
	if h.agentsDir != "" {
		fs := http.FileServer(http.Dir(h.agentsDir))
		mux.Handle("GET /download/", http.StripPrefix("/download/", fs))
	}

	// Outermost to innermost: request ID, logging, panic recovery, CORS.
	return middleware.RequestID(
		middleware.Logger(log)(
			middleware.Recover(log)(
				middleware.CORS(mux))))
}

func (h *Handlers) healthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
