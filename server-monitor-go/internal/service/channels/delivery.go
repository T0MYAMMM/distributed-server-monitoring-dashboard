package channels

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"strings"

	"github.com/thomasstefen/server-monitor/internal/domain"
)

// deliver routes an alert to the channel's destination using the type-specific
// payload. It is synchronous and returns an error on non-2xx or transport
// failure, so both the live notifier (wrapped in a goroutine) and the test-send
// can report the outcome.
func (s *Service) deliver(c domain.NotificationChannel, a domain.Alert) error {
	switch c.Type {
	case domain.ChannelSlack:
		return s.postJSON(c.Config["webhook_url"], map[string]string{"text": slackText(a)})
	case domain.ChannelDiscord:
		return s.postJSON(c.Config["webhook_url"], map[string]string{"content": slackText(a)})
	case domain.ChannelNtfy:
		return s.deliverNtfy(c, a)
	case domain.ChannelWebhook:
		return s.postJSON(c.Config["url"], a)
	case domain.ChannelPagerDuty:
		return s.deliverPagerDuty(c, a)
	case domain.ChannelEmail:
		return s.deliverEmail(c, a)
	default:
		return fmt.Errorf("unknown channel type %q", c.Type)
	}
}

// title renders a one-line headline for an alert.
func title(a domain.Alert) string {
	return fmt.Sprintf("[%s] %s", strings.ToUpper(a.Severity), a.Message)
}

func slackText(a domain.Alert) string {
	if a.ServerName != "" {
		return fmt.Sprintf("%s — %s", title(a), a.ServerName)
	}
	return title(a)
}

// postJSON POSTs v as JSON and treats any non-2xx as a delivery failure.
func (s *Service) postJSON(url string, v any) error {
	if url == "" {
		return fmt.Errorf("missing destination URL")
	}
	body, err := json.Marshal(v)
	if err != nil {
		return err
	}
	resp, err := s.client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("destination returned %s", resp.Status)
	}
	return nil
}

// deliverNtfy posts a plain-text message to an ntfy topic URL, mapping severity
// to ntfy priority and an optional bearer token for protected servers.
func (s *Service) deliverNtfy(c domain.NotificationChannel, a domain.Alert) error {
	url := c.Config["url"]
	if url == "" {
		return fmt.Errorf("missing ntfy topic URL")
	}
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(slackText(a)))
	if err != nil {
		return err
	}
	req.Header.Set("Title", "CloudGuard alert")
	req.Header.Set("Priority", ntfyPriority(a.Severity))
	switch a.Severity {
	case domain.SeverityCritical:
		req.Header.Set("Tags", "rotating_light")
	case domain.SeverityWarning:
		req.Header.Set("Tags", "warning")
	default:
		req.Header.Set("Tags", "white_check_mark")
	}
	if t := c.Config["token"]; t != "" {
		req.Header.Set("Authorization", "Bearer "+t)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("ntfy returned %s", resp.Status)
	}
	return nil
}

func ntfyPriority(sev string) string {
	switch sev {
	case domain.SeverityCritical:
		return "urgent"
	case domain.SeverityWarning:
		return "high"
	default:
		return "default"
	}
}

// deliverPagerDuty triggers an event through the PagerDuty Events API v2.
func (s *Service) deliverPagerDuty(c domain.NotificationChannel, a domain.Alert) error {
	key := c.Config["routing_key"]
	if key == "" {
		return fmt.Errorf("missing PagerDuty routing key")
	}
	source := a.ServerName
	if source == "" {
		source = "cloudguard"
	}
	payload := map[string]any{
		"routing_key":  key,
		"event_action": "trigger",
		"payload": map[string]any{
			"summary":  slackText(a),
			"source":   source,
			"severity": pagerSeverity(a.Severity),
		},
	}
	return s.postJSON("https://events.pagerduty.com/v2/enqueue", payload)
}

func pagerSeverity(sev string) string {
	switch sev {
	case domain.SeverityCritical:
		return "critical"
	case domain.SeverityWarning:
		return "warning"
	default:
		return "info"
	}
}

// deliverEmail sends a minimal RFC 5322 message over SMTP. Auth is used when a
// username is configured; otherwise an unauthenticated relay is assumed.
func (s *Service) deliverEmail(c domain.NotificationChannel, a domain.Alert) error {
	host, port := c.Config["host"], c.Config["port"]
	from, to := c.Config["from"], c.Config["to"]
	if host == "" || port == "" || from == "" || to == "" {
		return fmt.Errorf("email requires host, port, from and to")
	}
	recipients := splitList(to)
	subject := title(a)
	msg := strings.Join([]string{
		"From: " + from,
		"To: " + strings.Join(recipients, ", "),
		"Subject: CloudGuard — " + subject,
		"Content-Type: text/plain; charset=UTF-8",
		"",
		slackText(a),
	}, "\r\n")

	addr := host + ":" + port
	var auth smtp.Auth
	if u := c.Config["username"]; u != "" {
		auth = smtp.PlainAuth("", u, c.Config["password"], host)
	}
	return smtp.SendMail(addr, auth, from, recipients, []byte(msg))
}

func splitList(s string) []string {
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == ';' || r == ' ' })
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
