package alerts

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/thomasstefen/server-monitor/internal/domain"
)

// Notifier delivers an alert to an external channel. Implementations are
// best-effort and must not block the caller; the interface exists so ntfy,
// email, or Slack delivery can be added later without touching the service.
type Notifier interface {
	Notify(a domain.Alert)
}

// NopNotifier is the disabled notifier (no ALERT_WEBHOOK_URL configured).
type NopNotifier struct{}

// Notify does nothing.
func (NopNotifier) Notify(domain.Alert) {}

// WebhookNotifier POSTs the alert as JSON to a generic webhook URL.
type WebhookNotifier struct {
	url    string
	client *http.Client
	log    *slog.Logger
}

// NewNotifier returns a WebhookNotifier when url is set, otherwise the disabled
// NopNotifier so alerting is off by default.
func NewNotifier(url string, log *slog.Logger) Notifier {
	if url == "" {
		return NopNotifier{}
	}
	if log == nil {
		log = slog.Default()
	}
	return &WebhookNotifier{url: url, client: &http.Client{Timeout: 5 * time.Second}, log: log}
}

// Notify fires the webhook asynchronously so alert emission never blocks ingest
// or the sweep.
func (w *WebhookNotifier) Notify(a domain.Alert) {
	go func() {
		body, err := json.Marshal(a)
		if err != nil {
			w.log.Error("alert webhook marshal", "err", err)
			return
		}
		resp, err := w.client.Post(w.url, "application/json", bytes.NewReader(body))
		if err != nil {
			w.log.Error("alert webhook post", "err", err)
			return
		}
		resp.Body.Close()
	}()
}
