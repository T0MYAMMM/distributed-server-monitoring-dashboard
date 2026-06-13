// Package feedback handles in-app feedback: it stores each submission and,
// when a webhook is configured, forwards it asynchronously so the message
// reaches somewhere real (Slack, an issue tracker) without blocking the request.
package feedback

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/thomasstefen/server-monitor/internal/domain"
)

// Repo is the feedback persistence the service needs; sqlite.Store satisfies it.
type Repo interface {
	InsertFeedback(f domain.Feedback, when time.Time) (int64, error)
	ListFeedback(limit int) ([]domain.Feedback, error)
}

// Service implements the feedback use cases.
type Service struct {
	repo       Repo
	webhookURL string
	client     *http.Client
	log        *slog.Logger
}

var validCategory = map[string]bool{
	domain.FeedbackBug: true, domain.FeedbackIdea: true,
	domain.FeedbackPraise: true, domain.FeedbackGeneral: true,
}

// New constructs the feedback service. webhookURL may be empty (store only).
func New(repo Repo, webhookURL string, log *slog.Logger) *Service {
	if log == nil {
		log = slog.Default()
	}
	return &Service{repo: repo, webhookURL: webhookURL,
		client: &http.Client{Timeout: 5 * time.Second}, log: log}
}

// Submit validates and stores a feedback item, then forwards it to the webhook
// if configured. Returns domain.ErrInvalidInput for an empty message.
func (s *Service) Submit(category, message, page string) (domain.Feedback, error) {
	message = strings.TrimSpace(message)
	if message == "" {
		return domain.Feedback{}, domain.ErrInvalidInput
	}
	if !validCategory[category] {
		category = domain.FeedbackGeneral
	}
	now := time.Now()
	f := domain.Feedback{Category: category, Message: message, Page: page}
	id, err := s.repo.InsertFeedback(f, now)
	if err != nil {
		return domain.Feedback{}, err
	}
	f.ID = id
	f.CreatedAt = now.UTC().Format("2006-01-02 15:04:05")
	s.forward(f)
	return f, nil
}

// List returns recent submissions for the admin.
func (s *Service) List(limit int) ([]domain.Feedback, error) {
	return s.repo.ListFeedback(limit)
}

// forward posts the feedback to the configured webhook asynchronously.
func (s *Service) forward(f domain.Feedback) {
	if s.webhookURL == "" {
		return
	}
	go func() {
		payload := map[string]string{
			"text": "CloudGuard feedback [" + f.Category + "]: " + f.Message,
		}
		body, err := json.Marshal(payload)
		if err != nil {
			s.log.Error("feedback webhook marshal", "err", err)
			return
		}
		resp, err := s.client.Post(s.webhookURL, "application/json", bytes.NewReader(body))
		if err != nil {
			s.log.Error("feedback webhook post", "err", err)
			return
		}
		resp.Body.Close()
	}()
}
