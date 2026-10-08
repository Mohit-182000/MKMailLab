package settings

import (
	"context"
	"errors"
	"testing"

	"localmail/internal/domain"
	"localmail/internal/logging"
)

type memStore map[string]string

func (m memStore) GetSetting(_ context.Context, k string) (string, error) {
	v, ok := m[k]
	if !ok {
		return "", domain.ErrNotFound
	}
	return v, nil
}

func (m memStore) PutSetting(_ context.Context, k, v string) error {
	m[k] = v
	return nil
}

func TestValidate(t *testing.T) {
	cases := []struct {
		name  string
		mod   func(*SMTP)
		field string
	}{
		{"defaults ok", func(*SMTP) {}, ""},
		{"all interfaces ok", func(s *SMTP) { s.Host = "0.0.0.0" }, ""},
		{"localhost ok", func(s *SMTP) { s.Host = " localhost " }, ""},
		{"hostname rejected", func(s *SMTP) { s.Host = "example.com" }, "host"},
		{"port zero", func(s *SMTP) { s.Port = 0 }, "port"},
		{"port too big", func(s *SMTP) { s.Port = 70000 }, "port"},
		{"size", func(s *SMTP) { s.MaxMessageMB = 0 }, "maxMessageMb"},
		{"required needs user", func(s *SMTP) { s.AuthMode = AuthRequired }, "username"},
		{"required needs password", func(s *SMTP) { s.AuthMode = AuthRequired; s.Username = "u" }, "password"},
		{"required ok", func(s *SMTP) { s.AuthMode = AuthRequired; s.Username = "u"; s.Password = "p" }, ""},
		{"bad mode", func(s *SMTP) { s.AuthMode = "weird" }, "authMode"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := DefaultSMTP()
			c.mod(&s)
			err := s.Validate()
			var ve *ValidationError
			if c.field == "" {
				if err != nil {
					t.Fatalf("unexpected error %v", err)
				}
				return
			}
			if !errors.As(err, &ve) || ve.Field != c.field {
				t.Fatalf("err = %v, want field %s", err, c.field)
			}
		})
	}
}

func TestExposedToNetwork(t *testing.T) {
	for host, want := range map[string]bool{"127.0.0.1": false, "localhost": false, "::1": false, "0.0.0.0": true, "192.168.1.5": true} {
		if got := (SMTP{Host: host}).ExposedToNetwork(); got != want {
			t.Errorf("%s: got %v want %v", host, got, want)
		}
	}
}

func TestLoadSave(t *testing.T) {
	ctx := context.Background()
	store := memStore{}
	svc := NewService(store, logging.Discard())

	if err := svc.Load(ctx); err != nil {
		t.Fatal(err)
	}
	if svc.SMTP() != DefaultSMTP() {
		t.Fatal("first run should use defaults")
	}

	cfg := DefaultSMTP()
	cfg.Port = 2525
	if _, err := svc.SaveSMTP(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	bad := cfg
	bad.Port = -1
	if _, err := svc.SaveSMTP(ctx, bad); err == nil {
		t.Fatal("invalid settings saved")
	}

	reloaded := NewService(store, logging.Discard())
	if err := reloaded.Load(ctx); err != nil {
		t.Fatal(err)
	}
	if reloaded.SMTP().Port != 2525 {
		t.Fatalf("port = %d", reloaded.SMTP().Port)
	}

	// Partial document: missing fields keep defaults.
	store[keySMTP] = `{"port": 3000}`
	_ = reloaded.Load(ctx)
	if got := reloaded.SMTP(); got.Port != 3000 || got.Host != "127.0.0.1" || !got.AutoStart {
		t.Fatalf("partial merge wrong: %+v", got)
	}

	// Corrupt document falls back to defaults.
	store[keySMTP] = `{not json`
	if err := reloaded.Load(ctx); err != nil {
		t.Fatal(err)
	}
	if reloaded.SMTP() != DefaultSMTP() {
		t.Fatal("corrupt settings should fall back to defaults")
	}
}
