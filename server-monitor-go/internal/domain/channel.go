package domain

// NotificationChannel is an outbound alert delivery target (Slack, Discord,
// ntfy, a generic webhook, PagerDuty, or email). Config holds the type-specific
// fields (URLs, tokens, SMTP details); secret values are masked on read and
// preserved on update when sent blank.
type NotificationChannel struct {
	ID           int64             `json:"id"`
	Type         string            `json:"type"`
	Name         string            `json:"name"`
	Config       map[string]string `json:"config"`
	Enabled      bool              `json:"enabled"`
	SecretsSet   []string          `json:"secrets_set,omitempty"` // which secret keys hold a value
	LastStatus   string            `json:"last_status"`           // "" | "ok" | "error"
	LastError    string            `json:"last_error"`
	LastDelivery string            `json:"last_delivery"`
	CreatedAt    string            `json:"created_at"`
}

// Notification channel types.
const (
	ChannelSlack     = "slack"
	ChannelDiscord   = "discord"
	ChannelNtfy      = "ntfy"
	ChannelWebhook   = "webhook"
	ChannelPagerDuty = "pagerduty"
	ChannelEmail     = "email"
)
