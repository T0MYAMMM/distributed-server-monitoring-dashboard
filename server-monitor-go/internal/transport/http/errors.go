package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

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

// writeError renders an error in the shape appropriate to the surface: the
// canonical /api/v1 paths use the nested {"error":{"code","message"}} envelope;
// legacy /api/... paths keep the flat {"error":"message"} shape that existing
// clients depend on.
func writeError(w http.ResponseWriter, r *http.Request, status int, msg string) {
	if strings.HasPrefix(r.URL.Path, "/api/v1/") {
		writeJSON(w, status, map[string]any{
			"error": map[string]string{"code": codeForStatus(status), "message": msg},
		})
		return
	}
	writeJSON(w, status, map[string]string{"error": msg})
}

func codeForStatus(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "bad_request"
	case http.StatusUnauthorized:
		return "unauthorized"
	case http.StatusForbidden:
		return "forbidden"
	case http.StatusNotFound:
		return "not_found"
	case http.StatusConflict:
		return "conflict"
	case http.StatusInternalServerError:
		return "internal"
	default:
		return "error"
	}
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
func (h *Handlers) fail(w http.ResponseWriter, r *http.Request, err error) {
	status, msg := mapError(err)
	if status >= 500 {
		h.log.Error("internal error", "err", err)
	}
	writeError(w, r, status, msg)
}
