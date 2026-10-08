package bridge

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"localmail/internal/domain"
	"localmail/internal/mailbox"
	"localmail/internal/mailhttp"
)

// MailService exposes inbox operations to the UI.
type MailService struct {
	mb    *mailbox.Service
	saver FileSaver
}

// NewMailService wires the service. saver may be nil (tests).
func NewMailService(mb *mailbox.Service, saver FileSaver) *MailService {
	return &MailService{mb: mb, saver: saver}
}

// List returns a page of message summaries, newest first.
func (s *MailService) List(ctx context.Context, q domain.ListQuery) (domain.Page, error) {
	return s.mb.List(ctx, q)
}

// Open returns a full message and marks it read.
func (s *MailService) Open(ctx context.Context, id int64) (*domain.Message, error) {
	return s.mb.Open(ctx, id)
}

// SetRead marks messages read or unread.
func (s *MailService) SetRead(ctx context.Context, ids []int64, read bool) error {
	return s.mb.SetRead(ctx, ids, read)
}

// SetStarred stars or unstars messages.
func (s *MailService) SetStarred(ctx context.Context, ids []int64, starred bool) error {
	return s.mb.SetStarred(ctx, ids, starred)
}

// Delete removes messages.
func (s *MailService) Delete(ctx context.Context, ids []int64) error {
	return s.mb.Delete(ctx, ids)
}

// DeleteAll empties the mailbox.
func (s *MailService) DeleteAll(ctx context.Context) error {
	return s.mb.DeleteAll(ctx)
}

// FileSaver asks the user where to save a file. It returns "" if cancelled.
type FileSaver interface {
	PromptSavePath(defaultName, filterName, pattern string) (string, error)
}

// SaveRaw saves the original message as an .eml file chosen by the user.
// It returns the saved path, or "" if the user cancelled.
func (s *MailService) SaveRaw(ctx context.Context, id int64) (string, error) {
	raw, err := s.mb.Raw(ctx, id)
	if err != nil {
		return "", err
	}
	name := fmt.Sprintf("message-%d.eml", id)
	if m, err := s.mb.Get(ctx, id); err == nil && strings.TrimSpace(m.Subject) != "" {
		name = mailhttp.SafeFileName(truncate(m.Subject, 80)) + ".eml"
	}
	return s.save(name, "Email message (*.eml)", "*.eml", raw)
}

// SaveAttachment saves an attachment to a location chosen by the user.
func (s *MailService) SaveAttachment(ctx context.Context, msgID, attID int64) (string, error) {
	a, content, err := s.mb.Attachment(ctx, msgID, attID)
	if err != nil {
		return "", err
	}
	return s.save(mailhttp.SafeFileName(a.FileName), "All files (*.*)", "*.*", content)
}

func (s *MailService) save(name, filterName, pattern string, data []byte) (string, error) {
	if s.saver == nil {
		return "", errors.New("saving files is not available")
	}
	path, err := s.saver.PromptSavePath(name, filterName, pattern)
	if err != nil || path == "" {
		return "", err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("could not save file: %w", err)
	}
	return path, nil
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
