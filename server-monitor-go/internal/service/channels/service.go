// Package channels manages notification channels — the outbound alert delivery
// targets configured on the Integrations page. It implements alerts.Notifier so
// the alerts service can fan an alert out to every enabled channel without
// importing this package, and exposes CRUD + a synchronous test-send for the
// API. Secret config values are masked on read and preserved on blank update.
package channels

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/thomasstefen/server-monitor/internal/domain"
)

// Repo is the channel persistence the service needs; sqlite.Store satisfies it.
type Repo interface {
	InsertChannel(c domain.NotificationChannel) (int64, error)
	ListChannels() ([]domain.NotificationChannel, error)
	GetChannel(id int64) (domain.NotificationChannel, bool, error)
	EnabledChannels() ([]domain.NotificationChannel, error)
	UpdateChannel(c domain.NotificationChannel) error
	DeleteChannel(id int64) (bool, error)
	MarkChannelDelivery(id int64, status, errMsg string, when time.Time) error
}

// Service implements channel use cases and alert fan-out.
type Service struct {
	repo   Repo
	client *http.Client
	log    *slog.Logger
}

// New constructs the channels service. log may be nil.
func New(repo Repo, log *slog.Logger) *Service {
	if log == nil {
		log = slog.Default()
	}
	return &Service{repo: repo, client: &http.Client{Timeout: 8 * time.Second}, log: log}
}

// secretKeys are config fields never returned in clear text.
var secretKeys = map[string]bool{
	"webhook_url": true, "url": true, "token": true,
	"password": true, "routing_key": true,
}

func isSecret(k string) bool { return secretKeys[k] }

// validTypes are the supported channel types and their required config keys.
var requiredKeys = map[string][]string{
	domain.ChannelSlack:     {"webhook_url"},
	domain.ChannelDiscord:   {"webhook_url"},
	domain.ChannelNtfy:      {"url"},
	domain.ChannelWebhook:   {"url"},
	domain.ChannelPagerDuty: {"routing_key"},
	domain.ChannelEmail:     {"host", "port", "from", "to"},
}

// mask returns a copy of the channel safe to serialize: secret values blanked,
// with secrets_set listing which secrets hold a value.
func mask(c domain.NotificationChannel) domain.NotificationChannel {
	out := c
	out.Config = map[string]string{}
	out.SecretsSet = nil
	for k, v := range c.Config {
		if isSecret(k) {
			if v != "" {
				out.SecretsSet = append(out.SecretsSet, k)
			}
			out.Config[k] = ""
			continue
		}
		out.Config[k] = v
	}
	return out
}

// List returns all channels with secrets masked.
func (s *Service) List() ([]domain.NotificationChannel, error) {
	all, err := s.repo.ListChannels()
	if err != nil {
		return nil, err
	}
	for i := range all {
		all[i] = mask(all[i])
	}
	return all, nil
}

// Add validates and creates a channel, returning the masked record.
func (s *Service) Add(c domain.NotificationChannel) (domain.NotificationChannel, error) {
	if err := validate(c); err != nil {
		return domain.NotificationChannel{}, err
	}
	if c.Name == "" {
		c.Name = defaultName(c.Type)
	}
	id, err := s.repo.InsertChannel(c)
	if err != nil {
		return domain.NotificationChannel{}, err
	}
	saved, _, err := s.repo.GetChannel(id)
	if err != nil {
		return domain.NotificationChannel{}, err
	}
	return mask(saved), nil
}

// Update merges changes into an existing channel. Blank secret values are
// preserved from the stored record so the UI need not resend them.
func (s *Service) Update(id int64, name string, config map[string]string, enabled bool) (domain.NotificationChannel, error) {
	cur, ok, err := s.repo.GetChannel(id)
	if err != nil {
		return domain.NotificationChannel{}, err
	}
	if !ok {
		return domain.NotificationChannel{}, domain.ErrNotFound
	}
	merged := map[string]string{}
	for k, v := range config {
		if isSecret(k) && v == "" {
			merged[k] = cur.Config[k] // keep existing secret
			continue
		}
		merged[k] = v
	}
	cur.Name = name
	cur.Config = merged
	cur.Enabled = enabled
	if err := validate(cur); err != nil {
		return domain.NotificationChannel{}, err
	}
	if err := s.repo.UpdateChannel(cur); err != nil {
		return domain.NotificationChannel{}, err
	}
	return mask(cur), nil
}

// Remove deletes a channel.
func (s *Service) Remove(id int64) error {
	ok, err := s.repo.DeleteChannel(id)
	if err != nil {
		return err
	}
	if !ok {
		return domain.ErrNotFound
	}
	return nil
}

// Test delivers a sample alert to one channel synchronously and records the
// outcome, returning any delivery error so the UI can show success/failure.
func (s *Service) Test(id int64) error {
	c, ok, err := s.repo.GetChannel(id)
	if err != nil {
		return err
	}
	if !ok {
		return domain.ErrNotFound
	}
	sample := domain.Alert{
		Type: domain.AlertStatusChange, Severity: domain.SeverityInfo,
		ServerName: "cloudguard", Message: "Test notification from CloudGuard — delivery is working.",
		CreatedAt: time.Now().UTC().Format("2006-01-02 15:04:05"),
	}
	derr := s.deliver(c, sample)
	s.record(c.ID, derr)
	return derr
}

// Notify implements alerts.Notifier: deliver an alert to every enabled channel,
// asynchronously so alert emission never blocks.
func (s *Service) Notify(a domain.Alert) {
	go func() {
		list, err := s.repo.EnabledChannels()
		if err != nil {
			s.log.Error("load channels for notify", "err", err)
			return
		}
		for _, c := range list {
			err := s.deliver(c, a)
			s.record(c.ID, err)
			if err != nil {
				s.log.Error("channel delivery", "type", c.Type, "id", c.ID, "err", err)
			}
		}
	}()
}

func (s *Service) record(id int64, err error) {
	status, msg := "ok", ""
	if err != nil {
		status, msg = "error", err.Error()
	}
	if rerr := s.repo.MarkChannelDelivery(id, status, msg, time.Now()); rerr != nil {
		s.log.Error("mark channel delivery", "err", rerr)
	}
}

// validate checks the type is known and required keys are present.
func validate(c domain.NotificationChannel) error {
	req, ok := requiredKeys[c.Type]
	if !ok {
		return fmt.Errorf("%w: unknown channel type %q", domain.ErrInvalidInput, c.Type)
	}
	for _, k := range req {
		if strings.TrimSpace(c.Config[k]) == "" {
			return fmt.Errorf("%w: %s requires %q", domain.ErrInvalidInput, c.Type, k)
		}
	}
	return nil
}

func defaultName(t string) string {
	switch t {
	case domain.ChannelSlack:
		return "Slack"
	case domain.ChannelDiscord:
		return "Discord"
	case domain.ChannelNtfy:
		return "ntfy"
	case domain.ChannelWebhook:
		return "Webhook"
	case domain.ChannelPagerDuty:
		return "PagerDuty"
	case domain.ChannelEmail:
		return "Email"
	default:
		return t
	}
}
