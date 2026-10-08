// Package config resolves where LocalMail keeps its data on disk.
//
// Default (installed) mode stores everything under %LOCALAPPDATA%\LocalMail.
// Local rather than Roaming AppData is deliberate: the mail database can grow
// large and must not be synced by roaming profiles.
//
// Portable mode is enabled when a marker file (brand.PortableMarker) exists
// next to the executable; data is then kept in a "data" folder beside it.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"localmail/internal/brand"
)

// Paths lists every location LocalMail reads from or writes to.
type Paths struct {
	Root     string // base data directory
	Database string // SQLite database file
	Logs     string // rotating log files
	Certs    string // generated / user-supplied TLS certificates
	Temp     string // scratch files (open-in-browser, open attachment)
	Portable bool
}

// ResolveOptions carries the environment-dependent inputs of Resolve so the
// function stays pure and testable.
type ResolveOptions struct {
	// ExePath is the absolute path of the running executable.
	ExePath string
	// LocalAppData is the value of %LOCALAPPDATA%. When empty, os.UserCacheDir
	// semantics are expected to have been applied by the caller.
	LocalAppData string
	// Override, when non-empty, forces the data root (e.g. --data-dir flag).
	Override string
	// Dev selects a separate data directory so development builds never touch
	// the mailbox of an installed copy.
	Dev bool
	// FileExists reports whether a regular file exists; defaults to os.Stat.
	FileExists func(path string) bool
}

// ErrNoDataDir is returned when no usable data directory can be determined.
var ErrNoDataDir = errors.New("config: unable to determine a data directory")

// Resolve computes the data paths without touching the filesystem (apart from
// the portable-marker existence check).
func Resolve(opts ResolveOptions) (Paths, error) {
	exists := opts.FileExists
	if exists == nil {
		exists = fileExists
	}

	var (
		root     string
		portable bool
	)
	switch {
	case opts.Override != "":
		root = opts.Override
	case opts.ExePath != "" && exists(filepath.Join(filepath.Dir(opts.ExePath), brand.PortableMarker)):
		root = filepath.Join(filepath.Dir(opts.ExePath), "data")
		portable = true
	case opts.LocalAppData != "":
		name := brand.DataDirName
		if opts.Dev {
			name += "-Dev"
		}
		root = filepath.Join(opts.LocalAppData, name)
	default:
		return Paths{}, ErrNoDataDir
	}

	root, err := filepath.Abs(root)
	if err != nil {
		return Paths{}, fmt.Errorf("config: resolve data root: %w", err)
	}

	return Paths{
		Root:     root,
		Database: filepath.Join(root, brand.ExecutableName+".db"),
		Logs:     filepath.Join(root, "logs"),
		Certs:    filepath.Join(root, "certs"),
		Temp:     filepath.Join(root, "tmp"),
		Portable: portable,
	}, nil
}

// ResolveFromEnvironment calls Resolve with values taken from the running
// process. override may be empty.
func ResolveFromEnvironment(override string, dev bool) (Paths, error) {
	exe, err := os.Executable()
	if err != nil {
		exe = ""
	} else if resolved, rerr := filepath.EvalSymlinks(exe); rerr == nil {
		exe = resolved
	}

	local := os.Getenv("LOCALAPPDATA")
	if local == "" {
		// Non-Windows development hosts: fall back to the per-user cache dir.
		if dir, cerr := os.UserCacheDir(); cerr == nil {
			local = dir
		}
	}

	return Resolve(ResolveOptions{ExePath: exe, LocalAppData: local, Override: override, Dev: dev})
}

// EnsureDirs creates every directory in p.
func (p Paths) EnsureDirs() error {
	for _, dir := range []string{p.Root, p.Logs, p.Certs, p.Temp} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("config: create %s: %w", dir, err)
		}
	}
	return nil
}

// CleanTemp removes files left in the temp directory by earlier sessions.
// Failures are ignored: a file may still be open in another application.
func (p Paths) CleanTemp() {
	entries, err := os.ReadDir(p.Temp)
	if err != nil {
		return
	}
	for _, e := range entries {
		_ = os.RemoveAll(filepath.Join(p.Temp, e.Name()))
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}
