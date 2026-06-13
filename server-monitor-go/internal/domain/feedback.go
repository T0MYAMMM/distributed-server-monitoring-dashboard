package domain

// Feedback is one in-app submission: a category, a free-text message, and the
// page it was sent from. Stored for the admin and optionally forwarded to a
// webhook (e.g. Slack or an issue tracker).
type Feedback struct {
	ID        int64  `json:"id"`
	Category  string `json:"category"`
	Message   string `json:"message"`
	Page      string `json:"page"`
	CreatedAt string `json:"created_at"`
}

// Feedback categories.
const (
	FeedbackBug     = "bug"
	FeedbackIdea    = "idea"
	FeedbackPraise  = "praise"
	FeedbackGeneral = "general"
)
