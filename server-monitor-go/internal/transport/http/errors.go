package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/thomasstefen/server-monitor/internal/domain"
)

// writeJSON serializes v as a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("write json", "err", err)
	}
}

// writeError emits the flat legacy error body {"error": msg}. The /api/v1
// surface keeps this shape during the restructure; the nested
// {"error":{"code","message"}} envelope lands with the v1-consuming frontend.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// mapError maps a domain sentinel error to an HTTP status and message,
// preserving the legacy status codes the existing frontend and agents depend
// on. Non-domain errors are treated as internal server errors.
func mapError(err error) (int, string) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound, "Server not found"
	case errors.Is(err, domain.ErrNotAllowed):
		return http.StatusForbidden, "Client not allowed"
	case errors.Is(err, domain.ErrUnauthorized):
		return http.StatusUnauthorized, "Unauthorized"
	case errors.Is(err, domain.ErrConflict):
		return http.StatusBadRequest, "Already exists"
	case errors.Is(err, domain.ErrInvalidInput):
		return http.StatusBadRequest, "Invalid data"
	default:
		return http.StatusInternalServerError, err.Error()
	}
}

// fail renders err via the single mapping point, logging internal errors.
func (h *Handlers) fail(w http.ResponseWriter, err error) {
	status, msg := mapError(err)
	if status >= 500 {
		h.log.Error("internal error", "err", err)
	}
	writeError(w, status, msg)
}
