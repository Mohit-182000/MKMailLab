package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"localmail/internal/brand"
)

func TestResolve(t *testing.T) {
	base := t.TempDir()
	exeDir := filepath.Join(base, "app")
	exe := filepath.Join(exeDir, "localmail.exe")
	local := filepath.Join(base, "AppData", "Local")
	marker := filepath.Join(exeDir, brand.PortableMarker)

	tests := []struct {
		name         string
		opts         ResolveOptions
		wantRoot     string
		wantPortable bool
		wantErr      error
	}{
		{
			name:     "installed mode uses LOCALAPPDATA",
			opts:     ResolveOptions{ExePath: exe, LocalAppData: local},
			wantRoot: filepath.Join(local, brand.DataDirName),
		},
		{
			name:     "dev builds use a separate directory",
			opts:     ResolveOptions{ExePath: exe, LocalAppData: local, Dev: true},
			wantRoot: filepath.Join(local, brand.DataDirName+"-Dev"),
		},
		{
			name: "portable marker beside exe",
			opts: ResolveOptions{
				ExePath: exe, LocalAppData: local,
				FileExists: func(p string) bool { return p == marker },
			},
			wantRoot:     filepath.Join(exeDir, "data"),
			wantPortable: true,
		},
		{
			name: "override wins over portable marker",
			opts: ResolveOptions{
				ExePath: exe, LocalAppData: local, Override: filepath.Join(base, "custom"),
				FileExists: func(string) bool { return true },
			},
			wantRoot: filepath.Join(base, "custom"),
		},
		{
			name:    "no inputs is an error",
			opts:    ResolveOptions{},
			wantErr: ErrNoDataDir,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.opts.FileExists == nil {
				tt.opts.FileExists = func(string) bool { return false }
			}
			got, err := Resolve(tt.opts)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Root != tt.wantRoot {
				t.Errorf("Root = %q, want %q", got.Root, tt.wantRoot)
			}
			if got.Portable != tt.wantPortable {
				t.Errorf("Portable = %v, want %v", got.Portable, tt.wantPortable)
			}
			if got.Database != filepath.Join(tt.wantRoot, "localmail.db") {
				t.Errorf("Database = %q", got.Database)
			}
			if got.Logs != filepath.Join(tt.wantRoot, "logs") {
				t.Errorf("Logs = %q", got.Logs)
			}
		})
	}
}

func TestEnsureDirsAndCleanTemp(t *testing.T) {
	p, err := Resolve(ResolveOptions{Override: filepath.Join(t.TempDir(), "root")})
	if err != nil {
		t.Fatal(err)
	}
	if err := p.EnsureDirs(); err != nil {
		t.Fatalf("EnsureDirs: %v", err)
	}
	for _, dir := range []string{p.Root, p.Logs, p.Certs, p.Temp} {
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			t.Fatalf("%s not created: %v", dir, err)
		}
	}

	stale := filepath.Join(p.Temp, "stale.html")
	if err := os.WriteFile(stale, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	p.CleanTemp()
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("stale temp file not removed: %v", err)
	}
}
