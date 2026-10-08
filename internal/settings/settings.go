// Package settings owns typed, validated application settings persisted as
// JSON documents in the settings table. Corrupted or missing values fall back
// to defaults so a bad settings row can never prevent startup.
package settings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"

	"localmail/internal/domain"
)

// AuthMode controls SMTP authentication.
type AuthMode string

const (
	AuthNone     AuthMode = "none"     // AUTH not advertised
	AuthAny      AuthMode = "any"      // AUTH advertised, any credentials accepted
	AuthRequired AuthMode = "required" // AUTH required with the configured credentials
)

// SMTP holds SMTP server settings.
type SMTP struct {
	Host           string   `json:"host"`
	Port           int      `json:"port"`
	AuthMode       AuthMode `json:"authMode"`
	Username       string   `json:"username"`
	Password       string   `json:"password"`
	MaxMessageMB   int      `json:"maxMessageMb"`
	MaxConnections int      `json:"maxConnections"`
	AutoStart      bool     `json:"autoStart"`
}

// DefaultSMTP returns first-run SMTP settings.
func DefaultSMTP() SMTP {
	return SMTP{
		Host:           "127.0.0.1",
		Port:           1025,
		AuthMode:       AuthAny,
		MaxMessageMB:   25,
		MaxConnections: 100,
		AutoStart:      true,
	}
}

// ValidationError describes an invalid field; Field matches the JSON name so
// the UI can highlight it.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string { return e.Message }

// Validate checks the settings and normalises whitespace.
func (s *SMTP) Validate() error {
	s.Host = strings.TrimSpace(s.Host)
	s.Username = strings.TrimSpace(s.Username)
	switch {
	case s.Host == "":
		return &ValidationError{"host", "Host is required"}
	case s.Host != "localhost" && net.ParseIP(s.Host) == nil:
		return &ValidationError{"host", "Host must be an IP address (e.g. 127.0.0.1 or 0.0.0.0) or localhost"}
	case s.Port < 1 || s.Port > 65535:
		return &ValidationError{"port", "Port must be between 1 and 65535"}
	case s.MaxMessageMB < 1 || s.MaxMessageMB > 500:
		return &ValidationError{"maxMessageMb", "Maximum message size must be between 1 and 500 MB"}
	case s.MaxConnections < 1 || s.MaxConnections > 10000:
		return &ValidationError{"maxConnections", "Maximum connections must be between 1 and 10000"}
	}
	switch s.AuthMode {
	case AuthNone, AuthAny:
	case AuthRequired:
		if s.Username == "" {
			return &ValidationError{"username", "Username is required when authentication is required"}
		}
		if s.Password == "" {
			return &ValidationError{"password", "Password is required when authentication is required"}
		}
	default:
		return &ValidationError{"authMode", "Unknown authentication mode"}
	}
	return nil
}

// ExposedToNetwork reports whether the host accepts connections from other
// machines (anything other than loopback).
func (s SMTP) ExposedToNetwork() bool {
	if s.Host == "localhost" {
		return false
	}
	ip := net.ParseIP(s.Host)
	return ip == nil || !ip.IsLoopback()
}

// Store persists raw setting values.
type Store interface {
	GetSetting(ctx context.Context, key string) (string, error)
	PutSetting(ctx context.Context, key, value string) error
}

const keySMTP = "smtp"

// Service caches settings in memory and persists changes.
type Service struct {
	store Store
	log   *slog.Logger

	mu   sync.RWMutex
	smtp SMTP
}

// NewService creates a service holding defaults until Load is called.
func NewService(store Store, log *slog.Logger) *Service {
	return &Service{store: store, log: log, smtp: DefaultSMTP()}
}

// Load reads persisted settings. Missing fields keep their defaults; corrupt
// documents are logged and replaced by defaults.
func (s *Service) Load(ctx context.Context) error {
	cfg := DefaultSMTP()
	raw, err := s.store.GetSetting(ctx, keySMTP)
	switch {
	case errors.Is(err, domain.ErrNotFound):
		// first run
	case err != nil:
		return fmt.Errorf("settings: load smtp: %w", err)
	default:
		if uerr := json.Unmarshal([]byte(raw), &cfg); uerr != nil {
			s.log.Warn("smtp settings corrupted; using defaults", "err", uerr)
			cfg = DefaultSMTP()
		} else if verr := cfg.Validate(); verr != nil {
			s.log.Warn("smtp settings invalid; using defaults", "err", verr)
			cfg = DefaultSMTP()
		}
	}
	s.mu.Lock()
	s.smtp = cfg
	s.mu.Unlock()
	return nil
}

// SMTP returns the current SMTP settings.
func (s *Service) SMTP() SMTP {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.smtp
}

// SaveSMTP validates and persists new SMTP settings.
func (s *Service) SaveSMTP(ctx context.Context, cfg SMTP) (SMTP, error) {
	if err := cfg.Validate(); err != nil {
		return SMTP{}, err
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		return SMTP{}, err
	}
	if err := s.store.PutSetting(ctx, keySMTP, string(data)); err != nil {
		return SMTP{}, fmt.Errorf("settings: save smtp: %w", err)
	}
	s.mu.Lock()
	s.smtp = cfg
	s.mu.Unlock()
	s.log.Info("smtp settings saved", "host", cfg.Host, "port", cfg.Port, "auth", cfg.AuthMode)
	return cfg, nil
}
