package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/thomasstefen/server-monitor/internal/domain"
)

// ingestLogs accepts a batch of log lines from an agent. Allow-list gated like
// metrics ingest; no JWT (agents cannot send tokens). 503 when logs are
// disabled (no LOG_DATABASE_URL).
func (h *Handlers) ingestLogs(w http.ResponseWriter, r *http.Request) {
	if !h.logsEnabled() {
		writeError(w, r, http.StatusServiceUnavailable, "Logs are not enabled")
		return
	}
	var body struct {
		Server string           `json:"server"`
		Lines  []domain.LogLine `json:"lines"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Server == "" {
		writeError(w, r, http.StatusBadRequest, "Invalid data")
		return
	}
	allowed, err := h.servers.IsAllowed(body.Server)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	if !allowed {
		h.log.Warn("rejected log ingest", "name", body.Server, "remote", r.RemoteAddr)
		writeError(w, r, http.StatusForbidden, "Client not allowed")
		return
	}
	if err := h.logs.Ingest(r.Context(), body.Server, body.Lines); err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "success", "count": len(body.Lines)})
}

// queryLogs returns recent log lines for a server, filtered by level, search,
// time range, and source file.
func (h *Handlers) queryLogs(w http.ResponseWriter, r *http.Request) {
	if !h.logsEnabled() {
		writeError(w, r, http.StatusServiceUnavailable, "Logs are not enabled")
		return
	}
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	lines, err := h.logs.Query(r.Context(), domain.LogQuery{
		ServerID: r.PathValue("id"),
		Level:    q.Get("level"),
		Search:   q.Get("q"),
		Since:    q.Get("since"),
		Until:    q.Get("until"),
		File:     q.Get("file"),
		Limit:    limit,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, lines)
}

// streamLogs live-tails a server's logs over Server-Sent Events, polling the log
// store for rows newer than the last id it sent.
func (h *Handlers) streamLogs(w http.ResponseWriter, r *http.Request) {
	if !h.logsEnabled() {
		writeError(w, r, http.StatusServiceUnavailable, "Logs are not enabled")
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, r, http.StatusInternalServerError, "streaming unsupported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	id := r.PathValue("id")
	after, _ := strconv.ParseInt(r.URL.Query().Get("after"), 10, 64)
	ctx := r.Context()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		lines, err := h.logs.Query(ctx, domain.LogQuery{ServerID: id, AfterID: after, Limit: 200})
		if err == nil {
			for _, l := range lines {
				after = l.ID
				if b, mErr := json.Marshal(l); mErr == nil {
					fmt.Fprintf(w, "data: %s\n\n", b)
				}
			}
			flusher.Flush()
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (h *Handlers) logsEnabled() bool { return h.logs != nil && h.logs.Enabled() }
