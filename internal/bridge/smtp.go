package bridge

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"localmail/internal/settings"
	"localmail/internal/smtpd"
	"localmail/internal/testmail"
)

// SMTPConfigFrom converts persisted settings into a server configuration.
func SMTPConfigFrom(s settings.SMTP) smtpd.Config {
	return smtpd.Config{
		Host:            s.Host,
		Port:            s.Port,
		AuthMode:        smtpd.AuthMode(s.AuthMode),
		Username:        s.Username,
		Password:        s.Password,
		MaxMessageBytes: int64(s.MaxMessageMB) << 20,
		MaxConnections:  s.MaxConnections,
	}
}

// SaveResult is returned after saving SMTP settings.
type SaveResult struct {
	Config  settings.SMTP `json:"config"`
	Status  smtpd.Status  `json:"status"`
	Warning string        `json:"warning"`
}

// SMTPService controls the SMTP server from the UI.
type SMTPService struct {
	server   *smtpd.Server
	settings *settings.Service
	log      *slog.Logger
}

// NewSMTPService wires the service.
func NewSMTPService(server *smtpd.Server, st *settings.Service, log *slog.Logger) *SMTPService {
	return &SMTPService{server: server, settings: st, log: log}
}

// GetStatus returns the server status.
func (s *SMTPService) GetStatus() smtpd.Status {
	return s.server.Status()
}

// Start starts the server with the saved settings.
func (s *SMTPService) Start() (smtpd.Status, error) {
	return s.server.Start(SMTPConfigFrom(s.settings.SMTP()))
}

// Stop stops the server.
func (s *SMTPService) Stop() (smtpd.Status, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.server.Stop(ctx)
}

// GetConfig returns the saved SMTP settings.
func (s *SMTPService) GetConfig() settings.SMTP {
	return s.settings.SMTP()
}

// GetDefaultConfig returns first-run SMTP settings (for "Reset to defaults").
func (s *SMTPService) GetDefaultConfig() settings.SMTP {
	return settings.DefaultSMTP()
}

// SaveConfig validates and saves settings, restarting the server if it is
// running so changes apply immediately.
func (s *SMTPService) SaveConfig(ctx context.Context, cfg settings.SMTP) (SaveResult, error) {
	saved, err := s.settings.SaveSMTP(ctx, cfg)
	if err != nil {
		return SaveResult{}, err
	}
	res := SaveResult{Config: saved, Status: s.server.Status()}
	if saved.ExposedToNetwork() {
		res.Warning = "The SMTP server accepts connections from other computers on your network. Use 127.0.0.1 unless you need remote access."
	}
	if s.server.Running() {
		rctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		st, rerr := s.server.Restart(rctx, SMTPConfigFrom(saved))
		res.Status = st
		if rerr != nil {
			// Settings are saved; report the restart problem via status.
			s.log.Warn("restart after settings change failed", "err", rerr)
		}
	}
	return res, nil
}

// SendTestEmail sends a message through the running server.
func (s *SMTPService) SendTestEmail(ctx context.Context, m testmail.Message) error {
	st := s.server.Status()
	if !st.Running {
		return errors.New("start the SMTP server before sending a test email")
	}
	cfg := s.settings.SMTP()
	target := testmail.Target{Host: cfg.Host, Port: st.Port}
	if cfg.AuthMode == settings.AuthRequired {
		target.Username, target.Password = cfg.Username, cfg.Password
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	return testmail.Send(ctx, target, m)
}
