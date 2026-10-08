// Package bridge contains the thin, Wails-bound API surface exposed to the
// frontend. Bridge types validate input, call services and map results to
// DTOs; they hold no business logic of their own.
package bridge

import (
	"runtime"
	"time"

	"localmail/internal/brand"
	"localmail/internal/config"
	"localmail/internal/logging"
)

// AppInfo describes the running application for the About screen,
// diagnostics page and dashboard uptime.
type AppInfo struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	Commit       string `json:"commit"`
	Dev          bool   `json:"dev"`
	Portable     bool   `json:"portable"`
	DataDir      string `json:"dataDir"`
	DatabasePath string `json:"databasePath"`
	LogDir       string `json:"logDir"`
	LogFile      string `json:"logFile"`
	StartedAt    int64  `json:"startedAt"` // Unix milliseconds
	GoVersion    string `json:"goVersion"`
	Platform     string `json:"platform"`
}

// SystemService exposes application metadata and diagnostics to the UI.
type SystemService struct {
	paths     config.Paths
	logFile   string
	ring      *logging.Ring
	startedAt func() time.Time
}

// NewSystemService wires the service. startedAt reports lifecycle start time.
func NewSystemService(paths config.Paths, logFile string, ring *logging.Ring, startedAt func() time.Time) *SystemService {
	return &SystemService{paths: paths, logFile: logFile, ring: ring, startedAt: startedAt}
}

// GetAppInfo returns static and runtime information about the application.
func (s *SystemService) GetAppInfo() AppInfo {
	var started int64
	if t := s.startedAt(); !t.IsZero() {
		started = t.UnixMilli()
	}
	return AppInfo{
		Name:         brand.Name,
		Version:      brand.Version,
		Commit:       brand.Commit,
		Dev:          brand.Dev,
		Portable:     s.paths.Portable,
		DataDir:      s.paths.Root,
		DatabasePath: s.paths.Database,
		LogDir:       s.paths.Logs,
		LogFile:      s.logFile,
		StartedAt:    started,
		GoVersion:    runtime.Version(),
		Platform:     runtime.GOOS + "/" + runtime.GOARCH,
	}
}

// GetRecentLogs returns the in-memory log buffer, oldest first.
func (s *SystemService) GetRecentLogs() []logging.Entry {
	return s.ring.Snapshot()
}
