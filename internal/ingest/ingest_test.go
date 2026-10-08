package ingest

import (
	"context"
	"errors"
	"testing"
	"time"

	"localmail/internal/domain"
	"localmail/internal/events"
	"localmail/internal/logging"
)

type fakeStore struct {
	got *domain.NewMessage
	err error
}

func (f *fakeStore) InsertMessage(_ context.Context, m *domain.NewMessage) (int64, error) {
	f.got = m
	return 42, f.err
}

func TestIngestEnrichesAndEmits(t *testing.T) {
	store := &fakeStore{}
	rec := &events.Recorder{}
	now := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	svc := NewService(store, rec, logging.Discard(), func() time.Time { return now })

	raw := []byte("From: App <app@example.com>\r\nTo: user@example.com\r\nCc: CC@example.com\r\nSubject: Hi\r\n\r\nBody\r\n")
	id, err := svc.Ingest(context.Background(), Inbound{
		EnvelopeFrom: "bounce@example.com",
		EnvelopeTo:   []string{"user@example.com", "cc@example.com", "secret@example.com"},
		RemoteAddr:   "127.0.0.1:5000",
		Helo:         "laptop",
		Raw:          raw,
	})
	if err != nil || id != 42 {
		t.Fatalf("id=%d err=%v", id, err)
	}
	m := store.got
	if len(m.Bcc) != 1 || m.Bcc[0].Address != "secret@example.com" {
		t.Errorf("bcc = %v (cc match must be case-insensitive)", m.Bcc)
	}
	if !m.ReceivedAt.Equal(now) || m.Size != int64(len(raw)) || m.Helo != "laptop" {
		t.Errorf("metadata wrong: %+v", m.MessageSummary)
	}
	evs := rec.Named(events.MailReceived)
	if len(evs) != 1 || evs[0].Data.(domain.MessageSummary).ID != 42 {
		t.Fatalf("events = %+v", rec.Events)
	}
}

func TestIngestFallsBackToEnvelope(t *testing.T) {
	store := &fakeStore{}
	svc := NewService(store, &events.Recorder{}, logging.Discard(), nil)
	_, err := svc.Ingest(context.Background(), Inbound{
		EnvelopeFrom: "app@example.com",
		EnvelopeTo:   []string{"user@example.com"},
		Raw:          []byte("no headers at all"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if store.got.From.Address != "app@example.com" || len(store.got.To) != 1 {
		t.Fatalf("fallbacks not applied: from=%v to=%v", store.got.From, store.got.To)
	}
}

func TestIngestStoreFailureDoesNotEmit(t *testing.T) {
	rec := &events.Recorder{}
	svc := NewService(&fakeStore{err: errors.New("disk full")}, rec, logging.Discard(), nil)
	if _, err := svc.Ingest(context.Background(), Inbound{Raw: []byte("Subject: x\r\n\r\n")}); err == nil {
		t.Fatal("expected error")
	}
	if len(rec.Events) != 0 {
		t.Fatal("must not announce a message that was not stored")
	}
}
