// Package logging builds the application's structured logger.
//
// Every record fans out to:
//   - a size-rotated JSON file in the logs directory (for support/debugging),
//   - an in-memory Ring feeding the diagnostics page,
//   - stderr as human-readable text in development builds.
//
// Attributes with sensitive-looking keys (passwords, tokens, …) are redacted
// in every sink. Later phases add a database sink for warnings and errors.
package logging

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

// ComponentKey is the attribute key used to tag records with the subsystem
// that produced them ("smtp", "ingest", "storage", …).
const ComponentKey = "component"

// Options configures New.
type Options struct {
	Dir          string     // log directory; required
	FileName     string     // defaults to "localmail.log"
	MaxFileBytes int64      // defaults to 10 MiB
	KeepFiles    int        // rotated backups to keep; defaults to 5
	RingSize     int        // in-memory entries; defaults to 2000
	Level        slog.Level // initial minimum level
	Console      io.Writer  // optional human-readable sink (dev builds)
}

// Logging bundles the configured logger with its runtime controls.
type Logging struct {
	Logger *slog.Logger
	Ring   *Ring
	Level  *slog.LevelVar // adjustable at runtime from settings
	File   string         // absolute path of the active log file

	file *RotatingFile
}

// New creates the logger and its sinks.
func New(opts Options) (*Logging, error) {
	if opts.Dir == "" {
		return nil, errors.New("logging: Dir is required")
	}
	if opts.FileName == "" {
		opts.FileName = "localmail.log"
	}
	if opts.MaxFileBytes <= 0 {
		opts.MaxFileBytes = 10 << 20
	}
	if opts.KeepFiles == 0 {
		opts.KeepFiles = 5
	}
	if opts.RingSize <= 0 {
		opts.RingSize = 2000
	}

	if err := os.MkdirAll(opts.Dir, 0o700); err != nil {
		return nil, fmt.Errorf("logging: create dir: %w", err)
	}
	path := filepath.Join(opts.Dir, opts.FileName)
	file, err := OpenRotatingFile(path, opts.MaxFileBytes, opts.KeepFiles)
	if err != nil {
		return nil, err
	}

	level := new(slog.LevelVar)
	level.Set(opts.Level)
	ring := NewRing(opts.RingSize)

	handlerOpts := &slog.HandlerOptions{Level: level, ReplaceAttr: redactAttr}
	handlers := []slog.Handler{
		slog.NewJSONHandler(file, handlerOpts),
		ring.Handler(level),
	}
	if opts.Console != nil {
		handlers = append(handlers, slog.NewTextHandler(opts.Console, handlerOpts))
	}

	return &Logging{
		Logger: slog.New(slog.NewMultiHandler(handlers...)),
		Ring:   ring,
		Level:  level,
		File:   path,
		file:   file,
	}, nil
}

// Close flushes and closes the log file. The logger must not be used after.
func (l *Logging) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	_ = l.file.Sync()
	return l.file.Close()
}

// Component returns a child logger tagged with the given subsystem name.
func Component(l *slog.Logger, name string) *slog.Logger {
	return l.With(slog.String(ComponentKey, name))
}

// WithMinLevel returns a logger sharing l's sinks that additionally drops
// records below min. Used to quiet chatty third-party loggers (Wails logs
// every asset request at INFO) without lowering the app's own level.
func WithMinLevel(l *slog.Logger, min slog.Leveler) *slog.Logger {
	return slog.New(&minLevelHandler{inner: l.Handler(), min: min})
}

type minLevelHandler struct {
	inner slog.Handler
	min   slog.Leveler
}

func (h *minLevelHandler) Enabled(ctx context.Context, lvl slog.Level) bool {
	return lvl >= h.min.Level() && h.inner.Enabled(ctx, lvl)
}

func (h *minLevelHandler) Handle(ctx context.Context, r slog.Record) error {
	return h.inner.Handle(ctx, r)
}

func (h *minLevelHandler) WithAttrs(as []slog.Attr) slog.Handler {
	return &minLevelHandler{inner: h.inner.WithAttrs(as), min: h.min}
}

func (h *minLevelHandler) WithGroup(name string) slog.Handler {
	return &minLevelHandler{inner: h.inner.WithGroup(name), min: h.min}
}

// Discard returns a logger that drops everything; useful in tests.
func Discard() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}
