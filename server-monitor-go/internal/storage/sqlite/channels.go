package sqlite

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/thomasstefen/server-monitor/internal/domain"
)

// notification_channels persistence. Config is stored as a JSON object string;
// the service owns secret masking and type-specific validation.

func scanChannel(row interface{ Scan(...any) error }) (domain.NotificationChannel, error) {
	var c domain.NotificationChannel
	var cfg string
	var enabled int
	if err := row.Scan(&c.ID, &c.Type, &c.Name, &cfg, &enabled,
		&c.LastStatus, &c.LastError, &c.LastDelivery, &c.CreatedAt); err != nil {
		return c, err
	}
	c.Enabled = enabled != 0
	c.Config = map[string]string{}
	_ = json.Unmarshal([]byte(cfg), &c.Config)
	return c, nil
}

const channelColumns = `id, type, name, config, enabled, last_status, last_error, last_delivery, created_at`

// InsertChannel creates a channel and returns its id.
func (s *Store) InsertChannel(c domain.NotificationChannel) (int64, error) {
	cfg, err := json.Marshal(c.Config)
	if err != nil {
		return 0, err
	}
	enabled := 0
	if c.Enabled {
		enabled = 1
	}
	res, err := s.db.Exec(`INSERT INTO notification_channels
		(type, name, config, enabled) VALUES (?, ?, ?, ?)`,
		c.Type, c.Name, string(cfg), enabled)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// ListChannels returns all channels, newest first.
func (s *Store) ListChannels() ([]domain.NotificationChannel, error) {
	rows, err := s.db.Query(`SELECT ` + channelColumns + ` FROM notification_channels ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.NotificationChannel, 0)
	for rows.Next() {
		c, err := scanChannel(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// GetChannel fetches one channel by id.
func (s *Store) GetChannel(id int64) (domain.NotificationChannel, bool, error) {
	row := s.db.QueryRow(`SELECT `+channelColumns+` FROM notification_channels WHERE id = ?`, id)
	c, err := scanChannel(row)
	if err == sql.ErrNoRows {
		return domain.NotificationChannel{}, false, nil
	}
	if err != nil {
		return domain.NotificationChannel{}, false, err
	}
	return c, true, nil
}

// EnabledChannels returns only channels that are turned on (delivery targets).
func (s *Store) EnabledChannels() ([]domain.NotificationChannel, error) {
	all, err := s.ListChannels()
	if err != nil {
		return nil, err
	}
	out := make([]domain.NotificationChannel, 0, len(all))
	for _, c := range all {
		if c.Enabled {
			out = append(out, c)
		}
	}
	return out, nil
}

// UpdateChannel replaces a channel's name, config, and enabled flag.
func (s *Store) UpdateChannel(c domain.NotificationChannel) error {
	cfg, err := json.Marshal(c.Config)
	if err != nil {
		return err
	}
	enabled := 0
	if c.Enabled {
		enabled = 1
	}
	res, err := s.db.Exec(`UPDATE notification_channels
		SET name = ?, config = ?, enabled = ? WHERE id = ?`,
		c.Name, string(cfg), enabled, c.ID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// DeleteChannel removes a channel by id.
func (s *Store) DeleteChannel(id int64) (bool, error) {
	res, err := s.db.Exec(`DELETE FROM notification_channels WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// MarkChannelDelivery records the outcome of the most recent delivery attempt.
func (s *Store) MarkChannelDelivery(id int64, status, errMsg string, when time.Time) error {
	_, err := s.db.Exec(`UPDATE notification_channels
		SET last_status = ?, last_error = ?, last_delivery = ? WHERE id = ?`,
		status, errMsg, when.UTC().Format(timeLayout), id)
	return err
}
