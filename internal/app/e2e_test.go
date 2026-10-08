package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"localmail/internal/bridge"
	"localmail/internal/config"
	"localmail/internal/domain"
	"localmail/internal/events"
	"localmail/internal/logging"
	"localmail/internal/smtpd"
	"localmail/internal/testmail"
)

// TestEndToEnd exercises the real object graph: SMTP → parse → SQLite →
// mailbox API → content handler, plus events.
func TestEndToEnd(t *testing.T) {
	ctx := context.Background()
	paths, err := config.Resolve(config.ResolveOptions{Override: filepath.Join(t.TempDir(), "data")})
	if err != nil {
		t.Fatal(err)
	}
	if err := paths.EnsureDirs(); err != nil {
		t.Fatal(err)
	}
	logs, err := logging.New(logging.Options{Dir: paths.Logs})
	if err != nil {
		t.Fatal(err)
	}
	defer logs.Close()

	a, err := New(ctx, Options{Paths: paths, Logging: logs})
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var got []string
	a.AttachEvents(func(name string, _ any) {
		mu.Lock()
		got = append(got, name)
		mu.Unlock()
	})

	// Disable autostart so the test does not need port 1025.
	cfg := a.settings.SMTP()
	cfg.AutoStart = false
	if _, err := a.settings.SaveSMTP(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	if err := a.lifecycle.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = a.lifecycle.Stop(ctx) }()

	smtpCfg := bridge.SMTPConfigFrom(a.settings.SMTP())
	smtpCfg.Port = 0
	st, err := a.smtp.Start(smtpCfg)
	if err != nil {
		t.Fatal(err)
	}

	err = testmail.Send(ctx, testmail.Target{Host: "127.0.0.1", Port: st.Port}, testmail.Message{
		From: "App <app@example.com>", To: "user@example.com", Subject: "Hello ✓",
		Body: `<h1 onclick="x()">Hi</h1><script>alert(1)</script>`, IsHTML: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	page, err := a.mailAPI.List(ctx, domain.ListQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || page.Unread != 1 || page.Items[0].Subject != "Hello ✓" {
		t.Fatalf("page = %+v", page)
	}
	id := page.Items[0].ID

	msg, err := a.mailAPI.Open(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if !msg.IsRead || msg.From.Address != "app@example.com" || !strings.Contains(msg.HTML, "<h1") {
		t.Fatalf("message = %+v", msg)
	}

	rec := httptest.NewRecorder()
	a.content.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/messages/"+strconv.FormatInt(id, 10)+"/html", nil))
	if rec.Code != 200 || strings.Contains(rec.Body.String(), "<script") || !strings.Contains(rec.Body.String(), "<h1>Hi</h1>") {
		t.Fatalf("preview: %d %s", rec.Code, rec.Body.String())
	}

	if err := a.mailAPI.DeleteAll(ctx); err != nil {
		t.Fatal(err)
	}
	if page, _ := a.mailAPI.List(ctx, domain.ListQuery{}); page.Total != 0 {
		t.Fatal("mailbox not cleared")
	}

	stopCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if st, _ := a.smtp.Stop(stopCtx); st.State != smtpd.StateStopped {
		t.Fatal("not stopped")
	}

	mu.Lock()
	defer mu.Unlock()
	joined := strings.Join(got, ",")
	for _, want := range []string{events.SMTPStatus, events.MailReceived, events.MailChanged} {
		if !strings.Contains(joined, want) {
			t.Errorf("event %s not emitted; got %s", want, joined)
		}
	}
}
