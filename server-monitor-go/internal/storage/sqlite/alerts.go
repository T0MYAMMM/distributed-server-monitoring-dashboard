package sqlite

import (
	"database/sql"
	"time"

	"github.com/thomasstefen/server-monitor/internal/domain"
)

// InsertAlert stores an alert and returns its assigned id.
func (s *Store) InsertAlert(a domain.Alert, when time.Time) (int64, error) {
	res, err := s.db.Exec(`INSERT INTO alerts
		(type, server_id, server_name, severity, message, created_at, acknowledged_at)
		VALUES (?, ?, ?, ?, ?, ?, '')`,
		a.Type, a.ServerID, a.ServerName, a.Severity, a.Message,
		when.UTC().Format(timeLayout))
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// ListAlerts returns alerts most recent first, optionally filtered by severity
// and limited (0 = no limit).
func (s *Store) ListAlerts(severity string, limit int) ([]domain.Alert, error) {
	q := `SELECT id, type, server_id, server_name, severity, message,
		created_at, acknowledged_at FROM alerts`
	args := []any{}
	if severity != "" {
		q += ` WHERE severity = ?`
		args = append(args, severity)
	}
	q += ` ORDER BY id DESC`
	if limit > 0 {
		q += ` LIMIT ?`
		args = append(args, limit)
	}
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.Alert, 0)
	for rows.Next() {
		var a domain.Alert
		if err := rows.Scan(&a.ID, &a.Type, &a.ServerID, &a.ServerName,
			&a.Severity, &a.Message, &a.CreatedAt, &a.AcknowledgedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// AcknowledgeAlert stamps an alert acknowledged; returns false if id is unknown
// or already acknowledged.
func (s *Store) AcknowledgeAlert(id int64, when time.Time) (bool, error) {
	res, err := s.db.Exec(`UPDATE alerts SET acknowledged_at = ?
		WHERE id = ? AND acknowledged_at = ''`,
		when.UTC().Format(timeLayout), id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// UnacknowledgedThresholdExists reports whether an open threshold alert already
// exists for a server, so repeated breaches do not spam new alerts every report.
func (s *Store) UnacknowledgedThresholdExists(serverID string) (bool, error) {
	var x int
	err := s.db.QueryRow(`SELECT 1 FROM alerts
		WHERE server_id = ? AND type = ? AND acknowledged_at = '' LIMIT 1`,
		serverID, domain.AlertThreshold).Scan(&x)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}
