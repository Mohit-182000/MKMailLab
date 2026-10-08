package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"localmail/internal/brand"
)

// LegacyMigration reports what MigrateLegacyData did.
type LegacyMigration struct {
	Moved bool   // data was moved from the legacy directory
	From  string // legacy directory
	To    string // new directory
}

// MigrateLegacyData moves the data directory of the app's previous name
// (%LOCALAPPDATA%\LocalMail) to the current one and renames the database
// files, so captured emails and settings survive the rebrand.
//
// It only runs when the new directory does not exist yet and the legacy one
// does. If the move fails (e.g. the old app is still running and holds the
// database open) an error is returned and nothing is changed; the caller
// starts with a fresh directory and the legacy data stays where it was.
func MigrateLegacyData(localAppData, newRoot string) (LegacyMigration, error) {
	res := LegacyMigration{From: filepath.Join(localAppData, brand.LegacyDataDirName), To: newRoot}
	if localAppData == "" || filepath.Clean(res.From) == filepath.Clean(newRoot) {
		return res, nil
	}
	if _, err := os.Stat(newRoot); err == nil || !errors.Is(err, os.ErrNotExist) {
		return res, nil // new directory already in use
	}
	if info, err := os.Stat(res.From); err != nil || !info.IsDir() {
		return res, nil // nothing to migrate
	}

	if err := os.Rename(res.From, newRoot); err != nil {
		return res, fmt.Errorf("move %s to %s: %w", res.From, newRoot, err)
	}
	res.Moved = true

	// Database files carry the old executable name.
	for _, suffix := range []string{".db", ".db-wal", ".db-shm"} {
		oldPath := filepath.Join(newRoot, brand.LegacyExecutableName+suffix)
		newPath := filepath.Join(newRoot, brand.ExecutableName+suffix)
		if _, err := os.Stat(oldPath); err != nil {
			continue
		}
		if err := os.Rename(oldPath, newPath); err != nil {
			return res, fmt.Errorf("rename %s: %w", oldPath, err)
		}
	}
	return res, nil
}
