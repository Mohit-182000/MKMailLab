package bridge

import (
	"log/slog"
	"testing"
	"time"

	"localmail/internal/brand"
	"localmail/internal/config"
	"localmail/internal/logging"
)

func TestSystemServiceGetAppInfo(t *testing.T) {
	paths := config.Paths{Root: `C:\data`, Database: `C:\data\localmail.db`, Logs: `C:\data\logs`, Portable: true}
	ring := logging.NewRing(10)
	started := time.UnixMilli(1_700_000_000_000)

	svc := NewSystemService(paths, `C:\data\logs\localmail.log`, ring, func() time.Time { return started })
	info := svc.GetAppInfo()

	if info.Name != brand.Name || info.Version != brand.Version {
		t.Errorf("unexpected identity: %+v", info)
	}
	if !info.Portable || info.DataDir != paths.Root || info.DatabasePath != paths.Database {
		t.Errorf("paths not propagated: %+v", info)
	}
	if info.StartedAt != started.UnixMilli() {
		t.Errorf("StartedAt = %d", info.StartedAt)
	}

	notStarted := NewSystemService(paths, "", ring, func() time.Time { return time.Time{} })
	if notStarted.GetAppInfo().StartedAt != 0 {
		t.Error("zero start time should map to 0")
	}

	slog.New(ring.Handler(slog.LevelInfo)).Info("hello")
	if logs := svc.GetRecentLogs(); len(logs) != 1 || logs[0].Message != "hello" {
		t.Errorf("GetRecentLogs = %+v", logs)
	}
}
