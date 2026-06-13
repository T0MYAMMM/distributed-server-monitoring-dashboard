package sqlite

import (
	"time"

	"github.com/thomasstefen/server-monitor/internal/domain"
)

// InsertFeedback stores a submission and returns its id.
func (s *Store) InsertFeedback(f domain.Feedback, when time.Time) (int64, error) {
	res, err := s.db.Exec(`INSERT INTO feedback (category, message, page, created_at)
		VALUES (?, ?, ?, ?)`,
		f.Category, f.Message, f.Page, when.UTC().Format(timeLayout))
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// ListFeedback returns submissions most recent first (0 = no limit).
func (s *Store) ListFeedback(limit int) ([]domain.Feedback, error) {
	q := `SELECT id, category, message, page, created_at FROM feedback ORDER BY id DESC`
	args := []any{}
	if limit > 0 {
		q += ` LIMIT ?`
		args = append(args, limit)
	}
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.Feedback, 0)
	for rows.Next() {
		var f domain.Feedback
		if err := rows.Scan(&f.ID, &f.Category, &f.Message, &f.Page, &f.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}
