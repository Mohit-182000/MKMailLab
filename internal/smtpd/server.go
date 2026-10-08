// Package smtpd runs LocalMail's capturing SMTP server on top of
// emersion/go-smtp. It owns the listener lifecycle (start/stop/restart),
// connection limits, authentication policy and status reporting.
package smtpd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/emersion/go-smtp"
	"golang.org/x/net/netutil"

	"localmail/internal/events"
	"localmail/internal/ingest"
)

// AuthMode mirrors settings.AuthMode without importing settings.
type AuthMode string

const (
	AuthNone     AuthMode = "none"
	AuthAny      AuthMode = "any"
	AuthRequired AuthMode = "required"
)

// Config is the runtime configuration of one server instance.
type Config struct {
	Host            string
	Port            int
	AuthMode        AuthMode
	Username        string
	Password        string
	MaxMessageBytes int64
	MaxConnections  int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	Domain          string
}

func (c Config) addr() string { return net.JoinHostPort(c.Host, strconv.Itoa(c.Port)) }

// State is the lifecycle state of the server.
type State string

const (
	StateStopped State = "stopped"
	StateRunning State = "running"
	StateError   State = "error"
)

// Status is reported to the UI.
type Status struct {
	State          State      `json:"state"`
	Running        bool       `json:"running"`
	Host           string     `json:"host"`
	Port           int        `json:"port"`
	Error          string     `json:"error"`
	StartedAt      *time.Time `json:"startedAt"`
	Received       int64      `json:"received"`       // messages accepted since start
	ActiveSessions int64      `json:"activeSessions"` // open SMTP connections
}

// Ingester stores accepted messages.
type Ingester interface {
	Ingest(ctx context.Context, in ingest.Inbound) (int64, error)
}

// Server manages the SMTP listener.
type Server struct {
	log    *slog.Logger
	ingest Ingester
	emit   events.Emitter

	mu        sync.Mutex
	srv       *smtp.Server
	done      chan struct{}
	cfg       Config
	state     State
	lastErr   string
	startedAt *time.Time

	received atomic.Int64
	active   atomic.Int64

	// baseCtx is cancelled on Stop so in-flight ingests are abandoned
	// promptly; Stop is bounded by the caller's context anyway.
	baseCtx    context.Context
	cancelBase context.CancelFunc
}

// New creates a stopped server.
func New(log *slog.Logger, ingester Ingester, emit events.Emitter) *Server {
	return &Server{log: log, ingest: ingester, emit: emit, state: StateStopped}
}

// Start begins listening with cfg. Starting a running server is a no-op.
// A failure (e.g. port in use) is reported in the returned status and error.
func (s *Server) Start(cfg Config) (Status, error) {
	s.mu.Lock()
	if s.state == StateRunning {
		st := s.statusLocked()
		s.mu.Unlock()
		return st, nil
	}
	cfg = withDefaults(cfg)
	s.cfg = cfg

	ln, err := net.Listen("tcp", cfg.addr())
	if err != nil {
		s.state = StateError
		s.lastErr = friendlyListenError(err, cfg)
		st := s.statusLocked()
		s.mu.Unlock()
		s.log.Error("smtp listen failed", "addr", cfg.addr(), "err", err)
		s.emit.Emit(events.SMTPStatus, st)
		return st, errors.New(st.Error)
	}
	if tcp, ok := ln.Addr().(*net.TCPAddr); ok {
		cfg.Port = tcp.Port // resolves port 0 to the one actually bound
		s.cfg = cfg
	}
	ln = netutil.LimitListener(ln, cfg.MaxConnections)

	s.baseCtx, s.cancelBase = context.WithCancel(context.Background())
	srv := smtp.NewServer(&backend{s: s, cfg: cfg})
	srv.Domain = cfg.Domain
	srv.ReadTimeout = cfg.ReadTimeout
	srv.WriteTimeout = cfg.WriteTimeout
	srv.MaxMessageBytes = cfg.MaxMessageBytes
	srv.MaxRecipients = 1000
	// Real-world HTML mail often has very long lines; be lenient.
	srv.MaxLineLength = 1 << 20
	srv.AllowInsecureAuth = true // local tool; TLS is optional
	srv.EnableSMTPUTF8 = true
	srv.ErrorLog = slogAdapter{s.log}

	done := make(chan struct{})
	s.srv, s.done = srv, done
	now := time.Now().UTC()
	s.startedAt = &now
	s.state, s.lastErr = StateRunning, ""
	s.received.Store(0)
	st := s.statusLocked()
	s.mu.Unlock()

	go func() {
		defer close(done)
		if err := srv.Serve(ln); err != nil && !errors.Is(err, smtp.ErrServerClosed) && !errors.Is(err, net.ErrClosed) {
			s.log.Error("smtp server stopped unexpectedly", "err", err)
			s.mu.Lock()
			if s.srv == srv {
				s.state, s.lastErr, s.srv = StateError, "SMTP server stopped unexpectedly: "+err.Error(), nil
			}
			st := s.statusLocked()
			s.mu.Unlock()
			s.emit.Emit(events.SMTPStatus, st)
		}
	}()

	s.log.Info("smtp server listening", "addr", cfg.addr(), "auth", cfg.AuthMode)
	s.emit.Emit(events.SMTPStatus, st)
	return st, nil
}

// Stop closes the listener and waits for sessions to finish, forcibly
// closing them when ctx expires.
func (s *Server) Stop(ctx context.Context) (Status, error) {
	s.mu.Lock()
	srv, done, cancel := s.srv, s.done, s.cancelBase
	s.srv, s.done, s.cancelBase = nil, nil, nil
	s.state, s.lastErr, s.startedAt = StateStopped, "", nil
	st := s.statusLocked()
	s.mu.Unlock()

	if srv == nil {
		s.emit.Emit(events.SMTPStatus, st)
		return st, nil
	}

	var err error
	if serr := srv.Shutdown(ctx); serr != nil && !errors.Is(serr, smtp.ErrServerClosed) {
		// Idle clients may hold connections open; force them closed.
		_ = srv.Close()
		if !errors.Is(serr, context.DeadlineExceeded) && !errors.Is(serr, context.Canceled) {
			err = serr
		}
	}
	if cancel != nil {
		cancel()
	}
	select {
	case <-done:
	case <-ctx.Done():
	}
	s.log.Info("smtp server stopped")
	s.emit.Emit(events.SMTPStatus, st)
	return st, err
}

// Restart stops (if running) and starts with cfg.
func (s *Server) Restart(ctx context.Context, cfg Config) (Status, error) {
	if _, err := s.Stop(ctx); err != nil {
		s.log.Warn("error while stopping for restart", "err", err)
	}
	return s.Start(cfg)
}

// Status returns the current status.
func (s *Server) Status() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.statusLocked()
}

// publishStatus emits the current status (e.g. after the received counter
// changes) so the UI stays live without polling.
func (s *Server) publishStatus() {
	s.emit.Emit(events.SMTPStatus, s.Status())
}

// Running reports whether the server is accepting connections.
func (s *Server) Running() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state == StateRunning
}

func (s *Server) statusLocked() Status {
	return Status{
		State:          s.state,
		Running:        s.state == StateRunning,
		Host:           s.cfg.Host,
		Port:           s.cfg.Port,
		Error:          s.lastErr,
		StartedAt:      s.startedAt,
		Received:       s.received.Load(),
		ActiveSessions: s.active.Load(),
	}
}

func withDefaults(c Config) Config {
	if c.MaxConnections <= 0 {
		c.MaxConnections = 100
	}
	if c.MaxMessageBytes <= 0 {
		c.MaxMessageBytes = 25 << 20
	}
	if c.ReadTimeout <= 0 {
		c.ReadTimeout = 60 * time.Second
	}
	if c.WriteTimeout <= 0 {
		c.WriteTimeout = 60 * time.Second
	}
	if c.Domain == "" {
		c.Domain = "localmail.local"
	}
	if c.AuthMode == "" {
		c.AuthMode = AuthAny
	}
	return c
}

// Windows socket error codes.
const (
	wsaEACCES     = syscall.Errno(10013)
	wsaEADDRINUSE = syscall.Errno(10048)
	wsaEADDRNOTAV = syscall.Errno(10049)
)

func friendlyListenError(err error, cfg Config) string {
	var errno syscall.Errno
	if errors.As(err, &errno) {
		switch errno {
		case wsaEADDRINUSE, syscall.EADDRINUSE:
			return fmt.Sprintf("Port %d is already in use by another program. Stop that program or choose a different port.", cfg.Port)
		case wsaEACCES, syscall.EACCES:
			return fmt.Sprintf("Windows does not allow using port %d (it may be reserved by Hyper-V/WSL or need admin rights). Choose a different port, e.g. 2525.", cfg.Port)
		case wsaEADDRNOTAV, syscall.EADDRNOTAVAIL:
			return fmt.Sprintf("Address %s is not available on this computer.", cfg.Host)
		}
	}
	return "Could not start SMTP server: " + err.Error()
}

type slogAdapter struct{ log *slog.Logger }

func (a slogAdapter) Printf(format string, v ...any) { a.log.Warn(fmt.Sprintf(format, v...)) }
func (a slogAdapter) Println(v ...any)               { a.log.Warn(fmt.Sprint(v...)) }
