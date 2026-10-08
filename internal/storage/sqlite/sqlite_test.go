package sqlite

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"localmail/internal/domain"
	"localmail/internal/logging"
)

func openTest(t *testing.T) *DB {
	t.Helper()
	db, err := Open(context.Background(), filepath.Join(t.TempDir(), "test.db"), logging.Discard())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func sample(subject string, at time.Time) *domain.NewMessage {
	date := at.Add(-time.Minute)
	return &domain.NewMessage{
		Message: domain.Message{
			MessageSummary: domain.MessageSummary{
				MessageID:  "<" + subject + "@test>",
				Subject:    subject,
				From:       domain.Address{Name: "App", Address: "app@example.com"},
				To:         []domain.Address{{Name: "User", Address: "user@example.com"}},
				Snippet:    "hello",
				ReceivedAt: at,
				Size:       123,
			},
			Cc:           []domain.Address{{Address: "cc@example.com"}},
			Bcc:          []domain.Address{{Address: "hidden@example.com"}},
			EnvelopeFrom: "bounce@example.com",
			EnvelopeTo:   []string{"user@example.com", "cc@example.com", "hidden@example.com"},
			Date:         &date,
			Headers:      []domain.Header{{Name: "Subject", Value: subject}, {Name: "X-Test", Value: "1"}},
			Text:         "hello",
			HTML:         "<p>hello</p>",
			Attachments:  []domain.Attachment{{PartIndex: 0, FileName: "a.txt", ContentType: "text/plain", Size: 5}},
			ParseStatus:  domain.ParseOK,
		},
		Raw: []byte("Subject: " + subject + "\r\n\r\nhello"),
	}
}

func TestMigrationsAreIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "m.db")
	for range 2 {
		db, err := Open(context.Background(), path, logging.Discard())
		if err != nil {
			t.Fatal(err)
		}
		v, err := db.SchemaVersion(context.Background())
		if err != nil || v != 2 {
			t.Fatalf("version = %d, err = %v", v, err)
		}
		_ = db.Close()
	}
}

func TestInsertAndGetRoundTrip(t *testing.T) {
	db := openTest(t)
	ctx := context.Background()
	at := time.Date(2026, 10, 8, 10, 0, 0, 0, time.UTC)

	id, err := db.InsertMessage(ctx, sample("Welcome", at))
	if err != nil {
		t.Fatal(err)
	}
	m, err := db.GetMessage(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if m.Subject != "Welcome" || m.From.Address != "app@example.com" || !m.ReceivedAt.Equal(at) {
		t.Errorf("summary mismatch: %+v", m.MessageSummary)
	}
	if len(m.To) != 1 || len(m.Cc) != 1 || len(m.Bcc) != 1 || len(m.EnvelopeTo) != 3 {
		t.Errorf("recipients mismatch: to=%v cc=%v bcc=%v env=%v", m.To, m.Cc, m.Bcc, m.EnvelopeTo)
	}
	if len(m.Headers) != 2 || m.Headers[1].Name != "X-Test" {
		t.Errorf("headers mismatch: %v", m.Headers)
	}
	if m.HTML != "<p>hello</p>" || m.Text != "hello" || !m.HasHTML {
		t.Errorf("bodies mismatch")
	}
	if len(m.Attachments) != 1 || m.Attachments[0].FileName != "a.txt" || m.AttachmentCount != 1 {
		t.Errorf("attachments mismatch: %v", m.Attachments)
	}
	if m.Date == nil || !m.Date.Equal(at.Add(-time.Minute)) {
		t.Errorf("date mismatch: %v", m.Date)
	}
	raw, err := db.GetRaw(ctx, id)
	if err != nil || string(raw) != "Subject: Welcome\r\n\r\nhello" {
		t.Errorf("raw mismatch: %q %v", raw, err)
	}

	if _, err := db.GetMessage(ctx, 9999); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("missing message err = %v", err)
	}
}

func TestListPaginationAndSearch(t *testing.T) {
	db := openTest(t)
	ctx := context.Background()
	base := time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC)
	for i := range 7 {
		subject := fmt.Sprintf("Order %d", i)
		if i == 3 {
			subject = "Password reset 100%"
		}
		// Two messages share a timestamp to exercise the id tiebreaker.
		at := base.Add(time.Duration(i/2) * time.Minute)
		if _, err := db.InsertMessage(ctx, sample(subject, at)); err != nil {
			t.Fatal(err)
		}
	}

	var seen []int64
	cursor := ""
	for {
		page, err := db.ListMessages(ctx, domain.ListQuery{Limit: 3, Cursor: cursor})
		if err != nil {
			t.Fatal(err)
		}
		if page.Total != 7 || page.Unread != 7 {
			t.Fatalf("total/unread = %d/%d", page.Total, page.Unread)
		}
		for _, it := range page.Items {
			seen = append(seen, it.ID)
		}
		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
	}
	if len(seen) != 7 {
		t.Fatalf("paged through %d items, want 7: %v", len(seen), seen)
	}
	for i := 1; i < len(seen); i++ {
		if seen[i] >= seen[i-1] {
			t.Fatalf("not newest-first / duplicate: %v", seen)
		}
	}

	page, err := db.ListMessages(ctx, domain.ListQuery{Search: "100%"})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].Subject != "Password reset 100%" || page.Total != 1 {
		t.Fatalf("search returned %+v", page)
	}

	if err := db.SetRead(ctx, []int64{seen[0], seen[1]}, true); err != nil {
		t.Fatal(err)
	}
	page, _ = db.ListMessages(ctx, domain.ListQuery{UnreadOnly: true})
	if page.Total != 5 {
		t.Fatalf("unread total = %d, want 5", page.Total)
	}

	if _, err := db.ListMessages(ctx, domain.ListQuery{Cursor: "garbage"}); err == nil {
		t.Fatal("expected error for bad cursor")
	}
}

func TestDeleteCascades(t *testing.T) {
	db := openTest(t)
	ctx := context.Background()
	id1, _ := db.InsertMessage(ctx, sample("a", time.Now()))
	id2, _ := db.InsertMessage(ctx, sample("b", time.Now()))

	n, err := db.DeleteMessages(ctx, []int64{id1})
	if err != nil || n != 1 {
		t.Fatalf("delete n=%d err=%v", n, err)
	}
	for _, table := range []string{"email_bodies", "email_raw", "email_recipients", "email_headers", "email_attachments"} {
		var count int
		if err := db.reader.QueryRow(`SELECT COUNT(*) FROM `+table+` WHERE email_id = ?`, id1).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Errorf("%s still has %d rows for deleted message", table, count)
		}
	}
	if _, err := db.GetMessage(ctx, id2); err != nil {
		t.Fatalf("other message affected: %v", err)
	}

	n, err = db.DeleteAllMessages(ctx)
	if err != nil || n != 1 {
		t.Fatalf("delete all n=%d err=%v", n, err)
	}
}

func TestSettings(t *testing.T) {
	db := openTest(t)
	ctx := context.Background()
	if _, err := db.GetSetting(ctx, "smtp"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err = %v", err)
	}
	if err := db.PutSetting(ctx, "smtp", `{"port":1025}`); err != nil {
		t.Fatal(err)
	}
	if err := db.PutSetting(ctx, "smtp", `{"port":2525}`); err != nil {
		t.Fatal(err)
	}
	v, err := db.GetSetting(ctx, "smtp")
	if err != nil || v != `{"port":2525}` {
		t.Fatalf("v=%q err=%v", v, err)
	}
}

func TestIDsAreNeverReused(t *testing.T) {
	db := openTest(t)
	ctx := context.Background()
	id1, _ := db.InsertMessage(ctx, sample("a", time.Now()))
	id2, _ := db.InsertMessage(ctx, sample("b", time.Now()))
	if _, err := db.DeleteMessages(ctx, []int64{id2}); err != nil {
		t.Fatal(err)
	}
	id3, _ := db.InsertMessage(ctx, sample("c", time.Now()))
	if _, err := db.DeleteAllMessages(ctx); err != nil {
		t.Fatal(err)
	}
	id4, err := db.InsertMessage(ctx, sample("d", time.Now()))
	if err != nil {
		t.Fatal(err)
	}
	if !(id1 < id2 && id2 < id3 && id3 < id4) {
		t.Fatalf("ids reused or not increasing: %d %d %d %d", id1, id2, id3, id4)
	}
}

func TestSequenceSeededFromExistingData(t *testing.T) {
	// Simulates upgrading a v1 database that already holds messages.
	path := filepath.Join(t.TempDir(), "up.db")
	db, err := Open(context.Background(), path, logging.Discard())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.writer.Exec(`DELETE FROM id_sequence; DELETE FROM schema_migrations WHERE version = 2; DROP TABLE id_sequence`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.writer.Exec(`INSERT INTO emails (id, received_at) VALUES (41, 0)`); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()

	db, err = Open(context.Background(), path, logging.Discard()) // re-runs migration 2
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	id, err := db.InsertMessage(context.Background(), sample("next", time.Now()))
	if err != nil || id != 42 {
		t.Fatalf("id = %d, err = %v; want 42", id, err)
	}
}
