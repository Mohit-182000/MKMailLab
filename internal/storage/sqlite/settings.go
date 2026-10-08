package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"localmail/internal/domain"
)

// GetSetting returns the raw value stored under key.
func (db *DB) GetSetting(ctx context.Context, key string) (string, error) {
	var v string
	err := db.reader.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = ?`, key).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", domain.ErrNotFound
	}
	return v, err
}

// PutSetting stores value under key, replacing any previous value.
func (db *DB) PutSetting(ctx context.Context, key, value string) error {
	_, err := db.writer.ExecContext(ctx, `INSERT INTO settings (key, value, updated_at) VALUES (?, ?, ?)
		ON CONFLICT (key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		key, value, db.now().UnixMilli())
	return err
}
