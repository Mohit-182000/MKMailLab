package smtpd

import (
	"bytes"
	"context"
	"crypto/subtle"
	"errors"
	"io"
	"time"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"

	"localmail/internal/ingest"
)

const ingestTimeout = 30 * time.Second

var errStorage = &smtp.SMTPError{
	Code:         451,
	EnhancedCode: smtp.EnhancedCode{4, 3, 0},
	Message:      "Temporary local storage failure, please retry",
}

type backend struct {
	s   *Server
	cfg Config
}

func (b *backend) NewSession(c *smtp.Conn) (smtp.Session, error) {
	b.s.active.Add(1)
	remote := ""
	if nc := c.Conn(); nc != nil {
		remote = nc.RemoteAddr().String()
	}
	b.s.log.Debug("smtp connection opened", "remote", remote)
	return &session{s: b.s, cfg: b.cfg, conn: c, remote: remote}, nil
}

// session handles one SMTP connection. go-smtp serialises calls per session.
type session struct {
	s      *Server
	cfg    Config
	conn   *smtp.Conn
	remote string

	authed bool
	from   string
	to     []string
}

var _ smtp.AuthSession = (*session)(nil)

func (ss *session) AuthMechanisms() []string {
	if ss.cfg.AuthMode == AuthNone {
		return nil
	}
	return []string{sasl.Plain, sasl.Login}
}

func (ss *session) Auth(mech string) (sasl.Server, error) {
	check := func(username, password string) error {
		if ss.cfg.AuthMode == AuthRequired && !ss.credentialsMatch(username, password) {
			ss.s.log.Warn("smtp auth failed", "remote", ss.remote, "user", username)
			return smtp.ErrAuthFailed
		}
		ss.authed = true
		ss.s.log.Debug("smtp auth ok", "remote", ss.remote, "user", username)
		return nil
	}
	switch mech {
	case sasl.Plain:
		return sasl.NewPlainServer(func(_, username, password string) error { return check(username, password) }), nil
	case sasl.Login:
		return &loginServer{check: check}, nil
	default:
		return nil, smtp.ErrAuthUnsupported
	}
}

func (ss *session) credentialsMatch(user, pass string) bool {
	u := subtle.ConstantTimeCompare([]byte(user), []byte(ss.cfg.Username))
	p := subtle.ConstantTimeCompare([]byte(pass), []byte(ss.cfg.Password))
	return u&p == 1
}

func (ss *session) Mail(from string, _ *smtp.MailOptions) error {
	if ss.cfg.AuthMode == AuthRequired && !ss.authed {
		return smtp.ErrAuthRequired
	}
	ss.from = from
	return nil
}

func (ss *session) Rcpt(to string, _ *smtp.RcptOptions) error {
	ss.to = append(ss.to, to)
	return nil
}

func (ss *session) Data(r io.Reader) error {
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		// ErrDataTooLarge carries the correct 552 reply.
		if errors.Is(err, smtp.ErrDataTooLarge) {
			ss.s.log.Warn("smtp message too large", "remote", ss.remote, "limit", ss.cfg.MaxMessageBytes)
		}
		return err
	}

	ctx, cancel := context.WithTimeout(ss.s.baseCtxOrBackground(), ingestTimeout)
	defer cancel()
	_, err := ss.s.ingest.Ingest(ctx, ingest.Inbound{
		EnvelopeFrom: ss.from,
		EnvelopeTo:   append([]string(nil), ss.to...),
		RemoteAddr:   ss.remote,
		Helo:         ss.conn.Hostname(),
		Raw:          buf.Bytes(),
	})
	if err != nil {
		return errStorage
	}
	ss.s.received.Add(1)
	ss.s.publishStatus()
	return nil
}

func (ss *session) Reset() {
	ss.from, ss.to = "", nil
}

func (ss *session) Logout() error {
	ss.s.active.Add(-1)
	ss.s.log.Debug("smtp connection closed", "remote", ss.remote)
	ss.s.publishStatus()
	return nil
}

func (s *Server) baseCtxOrBackground() context.Context {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.baseCtx != nil {
		return s.baseCtx
	}
	return context.Background()
}

// loginServer implements the (non-standard but ubiquitous) AUTH LOGIN
// mechanism, which go-sasl no longer ships. .NET's SmtpClient and many PHP
// libraries prefer it over PLAIN.
type loginServer struct {
	check    func(username, password string) error
	step     int
	username string
}

func (l *loginServer) Next(response []byte) ([]byte, bool, error) {
	switch l.step {
	case 0:
		l.step = 1
		if len(response) == 0 {
			return []byte("Username:"), false, nil
		}
		fallthrough // initial response carried the username
	case 1:
		l.username = string(response)
		l.step = 2
		return []byte("Password:"), false, nil
	case 2:
		l.step = 3
		return nil, true, l.check(l.username, string(response))
	default:
		return nil, true, errors.New("unexpected LOGIN response")
	}
}
