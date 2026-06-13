package sqlite

import "time"

// settings is a simple key/value table backing the in-app Settings page. It
// holds only operator overrides; defaults and env precedence live in the
// settings service, so an empty table means "all defaults".

// SettingsAll returns every stored setting override as a key→value map.
func (s *Store) SettingsAll() (map[string]string, error) {
	rows, err := s.db.Query(`SELECT key, value FROM settings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		out[k] = v
	}
	return out, rows.Err()
}

// SetSetting upserts one setting override.
func (s *Store) SetSetting(key, value string) error {
	now := time.Now().UTC().Format(timeLayout)
	_, err := s.db.Exec(`INSERT INTO settings (key, value, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		key, value, now)
	return err
}
