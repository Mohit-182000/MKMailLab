package config

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, path, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestMigrateLegacyDataMovesAndRenames(t *testing.T) {
	local := t.TempDir()
	legacy := filepath.Join(local, "LocalMail")
	write(t, filepath.Join(legacy, "localmail.db"), "db")
	write(t, filepath.Join(legacy, "localmail.db-wal"), "wal")
	write(t, filepath.Join(legacy, "logs", "localmail.log"), "log")

	newRoot := filepath.Join(local, "MKMailLab")
	res, err := MigrateLegacyData(local, newRoot)
	if err != nil || !res.Moved {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Fatal("legacy directory still exists")
	}
	for name, want := range map[string]string{"mkmaillab.db": "db", "mkmaillab.db-wal": "wal", "logs/localmail.log": "log"} {
		data, err := os.ReadFile(filepath.Join(newRoot, name))
		if err != nil || string(data) != want {
			t.Errorf("%s: %q %v", name, data, err)
		}
	}
}

func TestMigrateLegacyDataSkips(t *testing.T) {
	local := t.TempDir()
	newRoot := filepath.Join(local, "MKMailLab")

	// Nothing to migrate.
	if res, err := MigrateLegacyData(local, newRoot); err != nil || res.Moved {
		t.Fatalf("res=%+v err=%v", res, err)
	}

	// New directory already exists: never overwrite it.
	write(t, filepath.Join(local, "LocalMail", "localmail.db"), "old")
	write(t, filepath.Join(newRoot, "mkmaillab.db"), "new")
	if res, err := MigrateLegacyData(local, newRoot); err != nil || res.Moved {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	data, _ := os.ReadFile(filepath.Join(newRoot, "mkmaillab.db"))
	if string(data) != "new" {
		t.Fatal("existing data was overwritten")
	}

	// No LOCALAPPDATA.
	if res, err := MigrateLegacyData("", newRoot); err != nil || res.Moved {
		t.Fatalf("res=%+v err=%v", res, err)
	}
}
