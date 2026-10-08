package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// RotatingFile is an io.WriteCloser that rolls the file over once it exceeds
// MaxBytes, keeping at most Keep previous files:
//
//	localmail.log  (current)
//	localmail.1.log (newest backup) … localmail.N.log (oldest)
//
// Rotation closes the file before renaming, which Windows requires.
type RotatingFile struct {
	path     string
	maxBytes int64
	keep     int

	mu   sync.Mutex
	file *os.File
	size int64
}

// OpenRotatingFile opens (or creates) path for appending.
func OpenRotatingFile(path string, maxBytes int64, keep int) (*RotatingFile, error) {
	if maxBytes <= 0 {
		return nil, fmt.Errorf("logging: maxBytes must be positive")
	}
	if keep < 0 {
		keep = 0
	}
	r := &RotatingFile{path: path, maxBytes: maxBytes, keep: keep}
	if err := r.open(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *RotatingFile) open() error {
	f, err := os.OpenFile(r.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("logging: open %s: %w", r.path, err)
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return fmt.Errorf("logging: stat %s: %w", r.path, err)
	}
	r.file, r.size = f, info.Size()
	return nil
}

// Write appends p, rotating first if p would push the file past the limit.
// A single record is never split across files.
func (r *RotatingFile) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.file == nil {
		return 0, os.ErrClosed
	}
	if r.size > 0 && r.size+int64(len(p)) > r.maxBytes {
		if err := r.rotate(); err != nil {
			// Keep logging into the current file rather than losing records.
			fmt.Fprintf(os.Stderr, "logging: rotate failed: %v\n", err)
		}
	}
	n, err := r.file.Write(p)
	r.size += int64(n)
	return n, err
}

func (r *RotatingFile) rotate() error {
	if err := r.file.Close(); err != nil {
		return err
	}
	r.file = nil

	if r.keep == 0 {
		_ = os.Remove(r.path)
	} else {
		_ = os.Remove(r.backupName(r.keep))
		for i := r.keep - 1; i >= 1; i-- {
			_ = os.Rename(r.backupName(i), r.backupName(i+1))
		}
		if err := os.Rename(r.path, r.backupName(1)); err != nil {
			// Re-open the original so writes continue.
			if oerr := r.open(); oerr != nil {
				return oerr
			}
			return err
		}
	}
	return r.open()
}

func (r *RotatingFile) backupName(i int) string {
	ext := filepath.Ext(r.path)
	return fmt.Sprintf("%s.%d%s", strings.TrimSuffix(r.path, ext), i, ext)
}

// Sync flushes the current file to disk.
func (r *RotatingFile) Sync() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.file == nil {
		return nil
	}
	return r.file.Sync()
}

// Close closes the current file. Further writes return os.ErrClosed.
func (r *RotatingFile) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.file == nil {
		return nil
	}
	err := r.file.Close()
	r.file = nil
	return err
}
