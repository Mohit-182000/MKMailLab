package logging

import (
	"bufio"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewWritesJSONAndRedacts(t *testing.T) {
	dir := t.TempDir()
	lg, err := New(Options{Dir: dir, Level: slog.LevelDebug})
	if err != nil {
		t.Fatal(err)
	}
	log := Component(lg.Logger, "smtp")
	log.Info("auth attempt", "user", "dev", "smtp_password", "hunter2",
		slog.Group("req", slog.String("AuthToken", "abc")))
	if err := lg.Close(); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "mkmaillab.log"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if strings.Contains(text, "hunter2") || strings.Contains(text, "abc") {
		t.Fatalf("secret leaked into log file: %s", text)
	}
	var rec map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(text)), &rec); err != nil {
		t.Fatalf("log line is not JSON: %v\n%s", err, text)
	}
	if rec["component"] != "smtp" || rec["msg"] != "auth attempt" || rec["user"] != "dev" {
		t.Fatalf("unexpected record: %v", rec)
	}

	entries := lg.Ring.Snapshot()
	if len(entries) != 1 {
		t.Fatalf("ring has %d entries, want 1", len(entries))
	}
	e := entries[0]
	if e.Component != "smtp" || e.Attrs["smtp_password"] != Redacted || e.Attrs["req.AuthToken"] != Redacted {
		t.Fatalf("ring entry not redacted/flattened: %+v", e)
	}
}

func TestLevelIsAdjustableAtRuntime(t *testing.T) {
	lg, err := New(Options{Dir: t.TempDir(), Level: slog.LevelWarn})
	if err != nil {
		t.Fatal(err)
	}
	defer lg.Close()

	lg.Logger.Info("hidden")
	lg.Level.Set(slog.LevelInfo)
	lg.Logger.Info("shown")

	got := lg.Ring.Snapshot()
	if len(got) != 1 || got[0].Message != "shown" {
		t.Fatalf("got %+v", got)
	}
}

func TestWithMinLevel(t *testing.T) {
	r := NewRing(10)
	base := slog.New(r.Handler(slog.LevelDebug))
	quiet := WithMinLevel(base, slog.LevelWarn).With("component", "wails")

	quiet.Info("asset request")
	quiet.Warn("webview problem")
	base.Debug("app debug still visible")

	got := r.Snapshot()
	if len(got) != 2 || got[0].Message != "webview problem" || got[0].Component != "wails" {
		t.Fatalf("got %+v", got)
	}
}

func TestRingWrapsAndSubscribes(t *testing.T) {
	r := NewRing(3)
	log := slog.New(r.Handler(slog.LevelInfo))

	ch, cancel := r.Subscribe(10)
	for i := range 5 {
		log.Info("m", "i", i)
	}
	snap := r.Snapshot()
	if len(snap) != 3 {
		t.Fatalf("len = %d, want 3", len(snap))
	}
	// Oldest retained should be i=2, newest i=4.
	if snap[0].Attrs["i"] != int64(2) || snap[2].Attrs["i"] != int64(4) {
		t.Fatalf("wrong order: %+v", snap)
	}
	if snap[0].Seq >= snap[2].Seq {
		t.Fatalf("sequence not increasing")
	}

	received := 0
	timeout := time.After(time.Second)
	for received < 5 {
		select {
		case <-ch:
			received++
		case <-timeout:
			t.Fatalf("received %d of 5 live entries", received)
		}
	}
	cancel()
	cancel() // idempotent
	if _, ok := <-ch; ok {
		t.Fatal("channel not closed after cancel")
	}

	r.Clear()
	if len(r.Snapshot()) != 0 {
		t.Fatal("Clear did not empty ring")
	}
}

func TestRingWithGroupAndAttrs(t *testing.T) {
	r := NewRing(4)
	log := slog.New(r.Handler(slog.LevelInfo)).With("component", "ingest").WithGroup("msg").With("id", 7)
	log.Info("stored", "size", 10)

	e := r.Snapshot()[0]
	if e.Component != "ingest" || e.Attrs["msg.id"] != int64(7) || e.Attrs["msg.size"] != int64(10) {
		t.Fatalf("unexpected entry: %+v", e)
	}
}

func TestRotatingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")
	rf, err := OpenRotatingFile(path, 100, 2)
	if err != nil {
		t.Fatal(err)
	}
	line := strings.Repeat("x", 59) + "\n" // 60 bytes; two lines exceed 100
	for range 7 {
		if _, err := rf.Write([]byte(line)); err != nil {
			t.Fatal(err)
		}
	}
	if err := rf.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := rf.Write([]byte("late")); err == nil {
		t.Fatal("write after close should fail")
	}

	for _, name := range []string{"app.log", "app.1.log", "app.2.log"} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(data) != 60 {
			t.Errorf("%s has %d bytes, want 60 (records must not be split)", name, len(data))
		}
		sc := bufio.NewScanner(strings.NewReader(string(data)))
		for sc.Scan() {
			if sc.Text() != strings.TrimSuffix(line, "\n") {
				t.Errorf("%s: corrupted line", name)
			}
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "app.3.log")); !os.IsNotExist(err) {
		t.Fatal("more backups kept than configured")
	}
}

func TestIsSensitiveKey(t *testing.T) {
	for _, k := range []string{"password", "SMTP_PASSWORD", "authToken", "Authorization", "api_key", "clientSecret"} {
		if !IsSensitiveKey(k) {
			t.Errorf("%q should be sensitive", k)
		}
	}
	for _, k := range []string{"user", "host", "port", "subject"} {
		if IsSensitiveKey(k) {
			t.Errorf("%q should not be sensitive", k)
		}
	}
}
