package smtpd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"sync"
	"testing"
	"time"

	"localmail/internal/events"
	"localmail/internal/ingest"
	"localmail/internal/logging"
)

type memIngest struct {
	mu   sync.Mutex
	msgs []ingest.Inbound
	err  error
}

func (m *memIngest) Ingest(_ context.Context, in ingest.Inbound) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return 0, m.err
	}
	m.msgs = append(m.msgs, in)
	return int64(len(m.msgs)), nil
}

func (m *memIngest) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.msgs)
}

func startTest(t *testing.T, cfg Config) (*Server, *memIngest, string) {
	t.Helper()
	ing := &memIngest{}
	s := New(logging.Discard(), ing, &events.Recorder{})
	if cfg.Host == "" {
		cfg.Host = "127.0.0.1"
	}
	st, err := s.Start(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_, _ = s.Stop(ctx)
	})
	return s, ing, fmt.Sprintf("127.0.0.1:%d", st.Port)
}

const testMsg = "From: a@example.com\r\nTo: b@example.com\r\nSubject: Hi\r\n\r\n.leading dot line\r\nBody\r\n"

func TestSendMailNoAuth(t *testing.T) {
	_, ing, addr := startTest(t, Config{AuthMode: AuthNone})
	if err := smtp.SendMail(addr, nil, "a@example.com", []string{"b@example.com", "c@example.com"}, []byte(testMsg)); err != nil {
		t.Fatal(err)
	}
	if ing.count() != 1 {
		t.Fatalf("got %d messages", ing.count())
	}
	got := ing.msgs[0]
	if got.EnvelopeFrom != "a@example.com" || len(got.EnvelopeTo) != 2 || got.Helo != "localhost" {
		t.Errorf("envelope = %+v", got)
	}
	if !strings.Contains(string(got.Raw), "\r\n.leading dot line\r\n") {
		t.Errorf("dot-stuffing not undone: %q", got.Raw)
	}
}

func TestAuthAnyAcceptsAnyCredentials(t *testing.T) {
	_, ing, addr := startTest(t, Config{AuthMode: AuthAny})
	auth := smtp.PlainAuth("", "whatever", "secret", "127.0.0.1")
	if err := smtp.SendMail(addr, auth, "a@example.com", []string{"b@example.com"}, []byte(testMsg)); err != nil {
		t.Fatal(err)
	}
	if ing.count() != 1 {
		t.Fatal("message not captured")
	}
}

func TestAuthRequired(t *testing.T) {
	_, ing, addr := startTest(t, Config{AuthMode: AuthRequired, Username: "dev", Password: "pw"})

	if err := smtp.SendMail(addr, nil, "a@example.com", []string{"b@example.com"}, []byte(testMsg)); err == nil {
		t.Fatal("unauthenticated send should fail")
	}
	bad := smtp.PlainAuth("", "dev", "wrong", "127.0.0.1")
	if err := smtp.SendMail(addr, bad, "a@example.com", []string{"b@example.com"}, []byte(testMsg)); err == nil {
		t.Fatal("wrong password should fail")
	}
	good := smtp.PlainAuth("", "dev", "pw", "127.0.0.1")
	if err := smtp.SendMail(addr, good, "a@example.com", []string{"b@example.com"}, []byte(testMsg)); err != nil {
		t.Fatalf("valid credentials rejected: %v", err)
	}
	if ing.count() != 1 {
		t.Fatalf("captured %d", ing.count())
	}
}

// TestAuthLogin drives AUTH LOGIN by hand (net/smtp has no LOGIN client).
func TestAuthLogin(t *testing.T) {
	_, ing, addr := startTest(t, Config{AuthMode: AuthRequired, Username: "dev", Password: "pw"})
	c := dialRaw(t, addr)
	c.expect("220")
	c.send("EHLO test")
	c.expectMulti("250")
	c.send("AUTH LOGIN")
	c.expect("334 VXNlcm5hbWU6") // "Username:"
	c.send("ZGV2")               // dev
	c.expect("334 UGFzc3dvcmQ6") // "Password:"
	c.send("cHc=")               // pw
	c.expect("235")
	c.send("MAIL FROM:<a@example.com>")
	c.expect("250")
	c.send("RCPT TO:<b@example.com>")
	c.expect("250")
	c.send("DATA")
	c.expect("354")
	c.send("Subject: login\r\n\r\nhello\r\n.")
	c.expect("250")
	c.send("QUIT")
	c.expect("221")
	if ing.count() != 1 {
		t.Fatal("message not captured")
	}
}

func TestMessageTooLarge(t *testing.T) {
	_, ing, addr := startTest(t, Config{AuthMode: AuthNone, MaxMessageBytes: 1024})
	big := testMsg + strings.Repeat("x", 4096) + "\r\n"
	err := smtp.SendMail(addr, nil, "a@example.com", []string{"b@example.com"}, []byte(big))
	if err == nil || !strings.Contains(err.Error(), "552") {
		t.Fatalf("err = %v, want 552", err)
	}
	if ing.count() != 0 {
		t.Fatal("oversized message stored")
	}
}

func TestStorageFailureReturns451(t *testing.T) {
	s, ing, addr := startTest(t, Config{AuthMode: AuthNone})
	ing.err = errors.New("disk full")
	err := smtp.SendMail(addr, nil, "a@example.com", []string{"b@example.com"}, []byte(testMsg))
	if err == nil || !strings.Contains(err.Error(), "451") {
		t.Fatalf("err = %v, want 451", err)
	}
	if s.Status().Received != 0 {
		t.Fatal("failed message counted as received")
	}
}

func TestConcurrentClients(t *testing.T) {
	s, ing, addr := startTest(t, Config{AuthMode: AuthAny})
	const n = 50
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			msg := fmt.Sprintf("Subject: concurrent %d\r\n\r\nbody %d\r\n", i, i)
			errs <- smtp.SendMail(addr, nil, "a@example.com", []string{"b@example.com"}, []byte(msg))
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if ing.count() != n || s.Status().Received != n {
		t.Fatalf("captured %d, status %d, want %d", ing.count(), s.Status().Received, n)
	}
}

func TestStartStopRestartAndPortInUse(t *testing.T) {
	s, _, addr := startTest(t, Config{AuthMode: AuthNone})
	if !s.Running() {
		t.Fatal("not running")
	}
	port := s.Status().Port

	// A second server on the same port must fail with a friendly message.
	other := New(logging.Discard(), &memIngest{}, &events.Recorder{})
	st, err := other.Start(Config{Host: "127.0.0.1", Port: port})
	if err == nil || st.State != StateError || !strings.Contains(st.Error, "already in use") {
		t.Fatalf("expected port-in-use error, got state=%s err=%v msg=%q", st.State, err, st.Error)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	st, err = s.Stop(ctx)
	if err != nil || st.Running {
		t.Fatalf("stop: %+v %v", st, err)
	}
	if _, err := net.DialTimeout("tcp", addr, 300*time.Millisecond); err == nil {
		t.Fatal("still accepting connections after stop")
	}
	if _, err := s.Stop(ctx); err != nil {
		t.Fatal("double stop should be a no-op")
	}

	st, err = s.Restart(ctx, Config{Host: "127.0.0.1", Port: port, AuthMode: AuthNone})
	if err != nil || !st.Running {
		t.Fatalf("restart: %+v %v", st, err)
	}
	if err := smtp.SendMail(addr, nil, "a@example.com", []string{"b@example.com"}, []byte(testMsg)); err != nil {
		t.Fatalf("send after restart: %v", err)
	}
}

func TestStopClosesIdleConnections(t *testing.T) {
	s, _, addr := startTest(t, Config{AuthMode: AuthNone})
	c := dialRaw(t, addr)
	c.expect("220")
	c.send("EHLO idle")
	c.expectMulti("250")

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, _ = s.Stop(ctx)
	if time.Since(start) > 2*time.Second {
		t.Fatal("Stop hung on idle connection")
	}
}

// --- raw protocol helper ---

type rawConn struct {
	t *testing.T
	c net.Conn
	r *bufio.Reader
}

func dialRaw(t *testing.T, addr string) *rawConn {
	t.Helper()
	c, err := net.DialTimeout("tcp", addr, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close() })
	_ = c.SetDeadline(time.Now().Add(5 * time.Second))
	return &rawConn{t: t, c: c, r: bufio.NewReader(c)}
}

func (r *rawConn) send(line string) {
	r.t.Helper()
	if _, err := r.c.Write([]byte(line + "\r\n")); err != nil {
		r.t.Fatal(err)
	}
}

func (r *rawConn) expect(prefix string) {
	r.t.Helper()
	line, err := r.r.ReadString('\n')
	if err != nil {
		r.t.Fatalf("read: %v", err)
	}
	if !strings.HasPrefix(line, prefix) {
		r.t.Fatalf("got %q, want prefix %q", line, prefix)
	}
}

func (r *rawConn) expectMulti(code string) {
	r.t.Helper()
	for {
		line, err := r.r.ReadString('\n')
		if err != nil {
			r.t.Fatalf("read: %v", err)
		}
		if !strings.HasPrefix(line, code) {
			r.t.Fatalf("got %q, want %s", line, code)
		}
		if len(line) > 3 && line[3] == ' ' {
			return
		}
	}
}
