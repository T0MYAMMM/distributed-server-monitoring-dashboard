package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"runtime"
	"strconv"

	"github.com/thomasstefen/server-monitor/internal/domain"
)

// --- settings ---

// aboutInfo gathers read-only facts for the Settings "About" panel.
func (h *Handlers) aboutInfo() map[string]string {
	logs := "disabled"
	if h.logsEnabled() {
		logs = "connected"
	}
	version := h.version
	if version == "" {
		version = "dev"
	}
	name := "CloudGuard"
	if h.settings != nil {
		name = h.settings.InstanceName()
	}
	return map[string]string{
		"version":       version,
		"go":            runtime.Version(),
		"log_database":  logs,
		"instance_name": name,
	}
}

// getSettings returns the editable settings plus About facts. Public read so the
// login screen can pick up the instance name and default theme.
func (h *Handlers) getSettings(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.settings.Doc(h.aboutInfo()))
}

// updateSettings persists setting overrides and applies the live-readable ones.
// Auth enforced by the route.
func (h *Handlers) updateSettings(w http.ResponseWriter, r *http.Request) {
	var in map[string]string
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, r, http.StatusBadRequest, "Invalid data")
		return
	}
	if err := h.settings.Update(in); err != nil {
		// Update errors are validation/env-lock messages meant for the operator.
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, h.settings.Doc(h.aboutInfo()))
}

// --- notification channels ---

func (h *Handlers) listChannels(w http.ResponseWriter, r *http.Request) {
	list, err := h.channels.List()
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *Handlers) addChannel(w http.ResponseWriter, r *http.Request) {
	var body domain.NotificationChannel
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, http.StatusBadRequest, "Invalid data")
		return
	}
	ch, err := h.channels.Add(body)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			writeError(w, r, http.StatusBadRequest, err.Error())
			return
		}
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, ch)
}

func (h *Handlers) updateChannel(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "Invalid channel id")
		return
	}
	var body struct {
		Name    string            `json:"name"`
		Config  map[string]string `json:"config"`
		Enabled bool              `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, http.StatusBadRequest, "Invalid data")
		return
	}
	ch, err := h.channels.Update(id, body.Name, body.Config, body.Enabled)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			writeError(w, r, http.StatusBadRequest, err.Error())
			return
		}
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, ch)
}

func (h *Handlers) deleteChannel(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "Invalid channel id")
		return
	}
	if err := h.channels.Remove(id); err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// testChannel delivers a sample alert and reports the outcome inline (a failed
// delivery is a 200 with ok:false, not a server error, so the UI can show it).
func (h *Handlers) testChannel(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "Invalid channel id")
		return
	}
	if err := h.channels.Test(id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			h.fail(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// --- feedback ---

func (h *Handlers) submitFeedback(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Category string `json:"category"`
		Message  string `json:"message"`
		Page     string `json:"page"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, r, http.StatusBadRequest, "Invalid data")
		return
	}
	f, err := h.feedback.Submit(body.Category, body.Message, body.Page)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			writeError(w, r, http.StatusBadRequest, "A message is required")
			return
		}
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, f)
}

func (h *Handlers) listFeedback(w http.ResponseWriter, r *http.Request) {
	limit := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	list, err := h.feedback.List(limit)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// --- analytics ---

// analyticsServers returns per-server uptime + capacity stats over the range.
func (h *Handlers) analyticsServers(w http.ResponseWriter, r *http.Request) {
	stats, err := h.metrics.ServerStats(r.URL.Query().Get("range"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

// analyticsLogVolume returns the per-bucket log volume by level over the range.
func (h *Handlers) analyticsLogVolume(w http.ResponseWriter, r *http.Request) {
	if !h.logsEnabled() {
		writeError(w, r, http.StatusServiceUnavailable, "Logs are not enabled")
		return
	}
	vol, err := h.logs.Volume(r.Context(), r.URL.Query().Get("server"), r.URL.Query().Get("range"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, vol)
}

// analyticsTopModules returns the busiest modules (with error counts).
func (h *Handlers) analyticsTopModules(w http.ResponseWriter, r *http.Request) {
	if !h.logsEnabled() {
		writeError(w, r, http.StatusServiceUnavailable, "Logs are not enabled")
		return
	}
	limit := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		limit, _ = strconv.Atoi(v)
	}
	mods, err := h.logs.TopModules(r.Context(), r.URL.Query().Get("server"), r.URL.Query().Get("range"), limit)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, mods)
}
