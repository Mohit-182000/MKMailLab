// Package mailbox implements inbox operations (list, read, delete, flags)
// and publishes change events so every view stays in sync.
package mailbox

import (
	"context"
	"log/slog"

	"localmail/internal/domain"
	"localmail/internal/events"
	"localmail/internal/mimeparse"
)

// Store is the persistence the mailbox needs.
type Store interface {
	ListMessages(ctx context.Context, q domain.ListQuery) (domain.Page, error)
	GetMessage(ctx context.Context, id int64) (*domain.Message, error)
	GetRaw(ctx context.Context, id int64) ([]byte, error)
	SetRead(ctx context.Context, ids []int64, read bool) error
	SetStarred(ctx context.Context, ids []int64, starred bool) error
	DeleteMessages(ctx context.Context, ids []int64) (int64, error)
	DeleteAllMessages(ctx context.Context) (int64, error)
}

// ChangeKind describes what happened to messages.
type ChangeKind string

const (
	ChangeRead      ChangeKind = "read"
	ChangeUnread    ChangeKind = "unread"
	ChangeStarred   ChangeKind = "starred"
	ChangeUnstarred ChangeKind = "unstarred"
	ChangeDeleted   ChangeKind = "deleted"
	ChangeCleared   ChangeKind = "cleared"
)

// Change is the payload of events.MailChanged.
type Change struct {
	Kind ChangeKind `json:"kind"`
	IDs  []int64    `json:"ids"`
}

// Service provides mailbox operations.
type Service struct {
	store Store
	emit  events.Emitter
	log   *slog.Logger
}

// NewService creates the service.
func NewService(store Store, emit events.Emitter, log *slog.Logger) *Service {
	return &Service{store: store, emit: emit, log: log}
}

// List returns a page of messages.
func (s *Service) List(ctx context.Context, q domain.ListQuery) (domain.Page, error) {
	return s.store.ListMessages(ctx, q)
}

// Open returns a message and marks it read.
func (s *Service) Open(ctx context.Context, id int64) (*domain.Message, error) {
	m, err := s.store.GetMessage(ctx, id)
	if err != nil {
		return nil, err
	}
	if !m.IsRead {
		if err := s.store.SetRead(ctx, []int64{id}, true); err != nil {
			s.log.Warn("mark read failed", "id", id, "err", err)
		} else {
			m.IsRead = true
			s.emit.Emit(events.MailChanged, Change{Kind: ChangeRead, IDs: []int64{id}})
		}
	}
	return m, nil
}

// Get returns a message without side effects.
func (s *Service) Get(ctx context.Context, id int64) (*domain.Message, error) {
	return s.store.GetMessage(ctx, id)
}

// Raw returns the original message source.
func (s *Service) Raw(ctx context.Context, id int64) ([]byte, error) {
	return s.store.GetRaw(ctx, id)
}

// Attachment returns attachment metadata and content.
func (s *Service) Attachment(ctx context.Context, msgID, attID int64) (domain.Attachment, []byte, error) {
	m, err := s.store.GetMessage(ctx, msgID)
	if err != nil {
		return domain.Attachment{}, nil, err
	}
	for _, a := range m.Attachments {
		if a.ID == attID {
			return s.partContent(ctx, msgID, a)
		}
	}
	return domain.Attachment{}, nil, domain.ErrNotFound
}

// InlineByCID returns the inline part with the given Content-ID.
func (s *Service) InlineByCID(ctx context.Context, msgID int64, cid string) (domain.Attachment, []byte, error) {
	m, err := s.store.GetMessage(ctx, msgID)
	if err != nil {
		return domain.Attachment{}, nil, err
	}
	for _, a := range m.Attachments {
		if a.ContentID != "" && a.ContentID == cid {
			return s.partContent(ctx, msgID, a)
		}
	}
	return domain.Attachment{}, nil, domain.ErrNotFound
}

func (s *Service) partContent(ctx context.Context, msgID int64, a domain.Attachment) (domain.Attachment, []byte, error) {
	raw, err := s.store.GetRaw(ctx, msgID)
	if err != nil {
		return a, nil, err
	}
	part, err := mimeparse.ExtractPart(raw, a.PartIndex)
	if err != nil {
		return a, nil, err
	}
	return a, part.Content, nil
}

// SetRead marks messages read/unread.
func (s *Service) SetRead(ctx context.Context, ids []int64, read bool) error {
	if err := s.store.SetRead(ctx, ids, read); err != nil {
		return err
	}
	kind := ChangeRead
	if !read {
		kind = ChangeUnread
	}
	s.emit.Emit(events.MailChanged, Change{Kind: kind, IDs: ids})
	return nil
}

// SetStarred stars/unstars messages.
func (s *Service) SetStarred(ctx context.Context, ids []int64, starred bool) error {
	if err := s.store.SetStarred(ctx, ids, starred); err != nil {
		return err
	}
	kind := ChangeStarred
	if !starred {
		kind = ChangeUnstarred
	}
	s.emit.Emit(events.MailChanged, Change{Kind: kind, IDs: ids})
	return nil
}

// Delete removes messages.
func (s *Service) Delete(ctx context.Context, ids []int64) error {
	n, err := s.store.DeleteMessages(ctx, ids)
	if err != nil {
		return err
	}
	s.log.Info("messages deleted", "count", n)
	s.emit.Emit(events.MailChanged, Change{Kind: ChangeDeleted, IDs: ids})
	return nil
}

// DeleteAll empties the mailbox.
func (s *Service) DeleteAll(ctx context.Context) error {
	n, err := s.store.DeleteAllMessages(ctx)
	if err != nil {
		return err
	}
	s.log.Info("mailbox cleared", "count", n)
	s.emit.Emit(events.MailChanged, Change{Kind: ChangeCleared, IDs: []int64{}})
	return nil
}
