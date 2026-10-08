// Package ingest turns a received SMTP transaction into a stored message:
// parse → enrich (BCC, fallbacks) → persist → publish event.
package ingest

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"localmail/internal/domain"
	"localmail/internal/events"
	"localmail/internal/mimeparse"
)

// Inbound is one SMTP transaction.
type Inbound struct {
	EnvelopeFrom string
	EnvelopeTo   []string
	RemoteAddr   string
	Helo         string
	Raw          []byte
}

// Store persists messages.
type Store interface {
	InsertMessage(ctx context.Context, m *domain.NewMessage) (int64, error)
}

// Service is the ingest pipeline.
type Service struct {
	store Store
	emit  events.Emitter
	log   *slog.Logger
	now   func() time.Time
}

// NewService creates the pipeline.
func NewService(store Store, emit events.Emitter, log *slog.Logger, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{store: store, emit: emit, log: log, now: now}
}

// Ingest parses and stores a message, returning its ID. It only fails if the
// message could not be persisted; parse problems are recorded on the message.
func (s *Service) Ingest(ctx context.Context, in Inbound) (int64, error) {
	r := mimeparse.Parse(in.Raw)

	msg := &domain.NewMessage{
		Message: domain.Message{
			MessageSummary: domain.MessageSummary{
				MessageID:       r.MessageID,
				Subject:         r.Subject,
				From:            r.From,
				To:              r.To,
				Snippet:         r.Snippet,
				ReceivedAt:      s.now().UTC(),
				Size:            int64(len(in.Raw)),
				AttachmentCount: len(r.Attachments),
				HasHTML:         r.HTML != "",
			},
			Cc:           r.Cc,
			Bcc:          computeBcc(in.EnvelopeTo, r.To, r.Cc, r.Bcc),
			ReplyTo:      r.ReplyTo,
			EnvelopeFrom: in.EnvelopeFrom,
			EnvelopeTo:   in.EnvelopeTo,
			Date:         r.Date,
			Headers:      r.Headers,
			Text:         r.Text,
			HTML:         r.HTML,
			Attachments:  r.Attachments,
			ParseStatus:  r.Status,
			ParseErrors:  r.Errors,
			RemoteAddr:   in.RemoteAddr,
			Helo:         in.Helo,
		},
		Raw: in.Raw,
	}
	if msg.From.Address == "" {
		msg.From.Address = in.EnvelopeFrom
	}
	if len(msg.To) == 0 {
		for _, a := range in.EnvelopeTo {
			msg.To = append(msg.To, domain.Address{Address: a})
		}
	}

	id, err := s.store.InsertMessage(ctx, msg)
	if err != nil {
		s.log.Error("failed to store message", "err", err, "size", len(in.Raw), "from", in.EnvelopeFrom)
		return 0, err
	}
	msg.ID = id

	s.log.Info("message received", "id", id, "from", msg.From.Address, "rcpt", len(in.EnvelopeTo),
		"size", len(in.Raw), "parse", r.Status)
	if r.Status != domain.ParseOK {
		s.log.Warn("message parsed with problems", "id", id, "status", r.Status, "errors", r.Errors)
	}
	s.emit.Emit(events.MailReceived, msg.MessageSummary)
	return id, nil
}

// computeBcc returns envelope recipients that do not appear in To or Cc,
// merged with any explicit Bcc header — i.e. who actually received a blind
// copy, which header inspection alone cannot reveal.
func computeBcc(envelope []string, to, cc, bccHeader []domain.Address) []domain.Address {
	visible := map[string]bool{}
	for _, list := range [][]domain.Address{to, cc} {
		for _, a := range list {
			visible[strings.ToLower(a.Address)] = true
		}
	}
	out := []domain.Address{}
	seen := map[string]bool{}
	for _, a := range bccHeader {
		k := strings.ToLower(a.Address)
		if !seen[k] {
			seen[k] = true
			out = append(out, a)
		}
	}
	for _, e := range envelope {
		k := strings.ToLower(e)
		if !visible[k] && !seen[k] {
			seen[k] = true
			out = append(out, domain.Address{Address: e})
		}
	}
	return out
}
