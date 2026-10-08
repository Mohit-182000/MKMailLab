// Package sqlite implements persistence on SQLite (pure-Go modernc driver).
//
// Two connection pools are used: a single-connection writer (SQLite allows
// one writer at a time; serialising in Go avoids SQLITE_BUSY churn) and a
// small reader pool that runs concurrently thanks to WAL mode.
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"time"

	_ "modernc.org/sqlite" // registers the "sqlite" driver
)

// DB owns the writer and reader pools.
type DB struct {
	path   string
	log    *slog.Logger
	now    func() time.Time
	writer *sql.DB
	reader *sql.DB
}

// Open opens (creating if needed) the database file at path and applies
// pending migrations.
func Open(ctx context.Context, path string, log *slog.Logger) (*DB, error) {
	q := url.Values{}
	for _, p := range []string{
		// auto_vacuum must precede the first table; it is a no-op afterwards.
		"auto_vacuum(INCREMENTAL)",
		"journal_mode(WAL)",
		"busy_timeout(5000)",
		"foreign_keys(1)",
		"synchronous(NORMAL)",
		"temp_store(MEMORY)",
	} {
		q.Add("_pragma", p)
	}
	base := path + "?" + q.Encode()

	writer, err := sql.Open("sqlite", base+"&_txlock=immediate")
	if err != nil {
		return nil, fmt.Errorf("sqlite: open writer: %w", err)
	}
	writer.SetMaxOpenConns(1)
	writer.SetMaxIdleConns(1)
	writer.SetConnMaxLifetime(0)

	reader, err := sql.Open("sqlite", base)
	if err != nil {
		_ = writer.Close()
		return nil, fmt.Errorf("sqlite: open reader: %w", err)
	}
	reader.SetMaxOpenConns(4)
	reader.SetMaxIdleConns(4)

	db := &DB{path: path, log: log, now: time.Now, writer: writer, reader: reader}
	if err := writer.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("sqlite: connect %s: %w", path, err)
	}
	if err := db.migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

// Close closes both pools.
func (db *DB) Close() error {
	return errors.Join(db.reader.Close(), db.writer.Close())
}

// Path returns the database file path.
func (db *DB) Path() string { return db.path }

// SetClock overrides the time source (tests).
func (db *DB) SetClock(now func() time.Time) { db.now = now }

// withTx runs fn in a write transaction, rolling back on error or panic.
func (db *DB) withTx(ctx context.Context, fn func(tx *sql.Tx) error) (err error) {
	tx, err := db.writer.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("sqlite: begin: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	if err = fn(tx); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("sqlite: commit: %w", err)
	}
	return nil
}
