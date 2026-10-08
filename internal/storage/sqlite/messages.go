package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"localmail/internal/domain"
)

const (
	defaultPageSize = 50
	maxPageSize     = 200
)

// InsertMessage stores a message and all its parts in one transaction and
// returns the new ID. Nothing is persisted if any step fails.
func (db *DB) InsertMessage(ctx context.Context, m *domain.NewMessage) (int64, error) {
	toJSON, err := json.Marshal(nonNilAddrs(m.To))
	if err != nil {
		return 0, err
	}
	errsJSON, err := json.Marshal(nonNilStrings(m.ParseErrors))
	if err != nil {
		return 0, err
	}
	var dateHeader any
	if m.Date != nil {
		dateHeader = m.Date.UnixMilli()
	}
	status := m.ParseStatus
	if status == "" {
		status = domain.ParseOK
	}

	var id int64
	err = db.withTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `INSERT INTO emails (
			message_id, subject, from_name, from_addr, to_json, envelope_from, date_header,
			received_at, size_bytes, has_html, has_text, attachment_count, snippet,
			is_read, is_starred, remote_addr, helo, parse_status, parse_errors_json
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			m.MessageID, m.Subject, m.From.Name, m.From.Address, string(toJSON), m.EnvelopeFrom, dateHeader,
			m.ReceivedAt.UnixMilli(), m.Size, m.HTML != "", m.Text != "", len(m.Attachments), m.Snippet,
			m.IsRead, m.IsStarred, m.RemoteAddr, m.Helo, string(status), string(errsJSON))
		if err != nil {
			return fmt.Errorf("insert email: %w", err)
		}
		if id, err = res.LastInsertId(); err != nil {
			return err
		}

		if _, err := tx.ExecContext(ctx, `INSERT INTO email_bodies (email_id, text_body, html_body) VALUES (?,?,?)`,
			id, m.Text, m.HTML); err != nil {
			return fmt.Errorf("insert body: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO email_raw (email_id, raw) VALUES (?,?)`, id, m.Raw); err != nil {
			return fmt.Errorf("insert raw: %w", err)
		}

		rcpt, err := tx.PrepareContext(ctx, `INSERT INTO email_recipients (email_id, kind, name, address) VALUES (?,?,?,?)`)
		if err != nil {
			return err
		}
		defer rcpt.Close()
		groups := []struct {
			kind  string
			addrs []domain.Address
		}{{"to", m.To}, {"cc", m.Cc}, {"bcc", m.Bcc}, {"reply_to", m.ReplyTo}}
		for _, g := range groups {
			for _, a := range g.addrs {
				if _, err := rcpt.ExecContext(ctx, id, g.kind, a.Name, a.Address); err != nil {
					return fmt.Errorf("insert recipient: %w", err)
				}
			}
		}
		for _, a := range m.EnvelopeTo {
			if _, err := rcpt.ExecContext(ctx, id, "envelope", "", a); err != nil {
				return fmt.Errorf("insert envelope recipient: %w", err)
			}
		}

		hdr, err := tx.PrepareContext(ctx, `INSERT INTO email_headers (email_id, ordinal, name, value) VALUES (?,?,?,?)`)
		if err != nil {
			return err
		}
		defer hdr.Close()
		for i, h := range m.Headers {
			if _, err := hdr.ExecContext(ctx, id, i, h.Name, h.Value); err != nil {
				return fmt.Errorf("insert header: %w", err)
			}
		}

		att, err := tx.PrepareContext(ctx, `INSERT INTO email_attachments
			(email_id, part_index, filename, content_type, size_bytes, content_id, inline) VALUES (?,?,?,?,?,?,?)`)
		if err != nil {
			return err
		}
		defer att.Close()
		for _, a := range m.Attachments {
			if _, err := att.ExecContext(ctx, id, a.PartIndex, a.FileName, a.ContentType, a.Size, a.ContentID, a.Inline); err != nil {
				return fmt.Errorf("insert attachment: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return id, nil
}

const summaryColumns = `id, message_id, subject, from_name, from_addr, to_json, snippet,
	received_at, size_bytes, attachment_count, has_html, is_read, is_starred`

func scanSummary(sc interface{ Scan(...any) error }) (domain.MessageSummary, error) {
	var (
		s          domain.MessageSummary
		toJSON     string
		receivedMs int64
	)
	err := sc.Scan(&s.ID, &s.MessageID, &s.Subject, &s.From.Name, &s.From.Address, &toJSON, &s.Snippet,
		&receivedMs, &s.Size, &s.AttachmentCount, &s.HasHTML, &s.IsRead, &s.IsStarred)
	if err != nil {
		return s, err
	}
	s.ReceivedAt = time.UnixMilli(receivedMs).UTC()
	if err := json.Unmarshal([]byte(toJSON), &s.To); err != nil || s.To == nil {
		s.To = []domain.Address{}
	}
	return s, nil
}

// ListMessages returns a page of summaries, newest first, using keyset
// pagination on (received_at, id) so deep pages stay fast.
func (db *DB) ListMessages(ctx context.Context, q domain.ListQuery) (domain.Page, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = defaultPageSize
	}
	limit = min(limit, maxPageSize)

	var (
		where []string
		args  []any
	)
	if s := strings.TrimSpace(q.Search); s != "" {
		like := "%" + escapeLike(s) + "%"
		where = append(where, `(subject LIKE ? ESCAPE '\' OR from_addr LIKE ? ESCAPE '\' OR from_name LIKE ? ESCAPE '\' OR to_json LIKE ? ESCAPE '\')`)
		args = append(args, like, like, like, like)
	}
	if q.UnreadOnly {
		where = append(where, "is_read = 0")
	}
	filterWhere := strings.Join(where, " AND ")
	filterArgs := append([]any(nil), args...)

	if q.Cursor != "" {
		ms, id, err := decodeCursor(q.Cursor)
		if err != nil {
			return domain.Page{}, err
		}
		where = append(where, "(received_at < ? OR (received_at = ? AND id < ?))")
		args = append(args, ms, ms, id)
	}

	query := "SELECT " + summaryColumns + " FROM emails"
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY received_at DESC, id DESC LIMIT ?"
	args = append(args, limit+1)

	rows, err := db.reader.QueryContext(ctx, query, args...)
	if err != nil {
		return domain.Page{}, fmt.Errorf("list messages: %w", err)
	}
	defer rows.Close()

	page := domain.Page{Items: make([]domain.MessageSummary, 0, limit)}
	for rows.Next() {
		s, err := scanSummary(rows)
		if err != nil {
			return domain.Page{}, err
		}
		page.Items = append(page.Items, s)
	}
	if err := rows.Err(); err != nil {
		return domain.Page{}, err
	}
	if len(page.Items) > limit {
		page.Items = page.Items[:limit]
		last := page.Items[limit-1]
		page.NextCursor = encodeCursor(last.ReceivedAt.UnixMilli(), last.ID)
	}

	countQuery := "SELECT COUNT(*), COALESCE(SUM(is_read = 0), 0) FROM emails"
	if filterWhere != "" {
		countQuery += " WHERE " + filterWhere
	}
	if err := db.reader.QueryRowContext(ctx, countQuery, filterArgs...).Scan(&page.Total, &page.Unread); err != nil {
		return domain.Page{}, fmt.Errorf("count messages: %w", err)
	}
	return page, nil
}

// GetMessage returns the full message with bodies, headers, recipients and
// attachment metadata.
func (db *DB) GetMessage(ctx context.Context, id int64) (*domain.Message, error) {
	row := db.reader.QueryRowContext(ctx, `SELECT `+summaryColumns+`,
		envelope_from, date_header, remote_addr, helo, parse_status, parse_errors_json,
		COALESCE(b.text_body, ''), COALESCE(b.html_body, '')
		FROM emails e LEFT JOIN email_bodies b ON b.email_id = e.id WHERE e.id = ?`, id)

	var (
		m          domain.Message
		toJSON     string
		receivedMs int64
		dateMs     sql.NullInt64
		status     string
		errsJSON   string
	)
	err := row.Scan(&m.ID, &m.MessageID, &m.Subject, &m.From.Name, &m.From.Address, &toJSON, &m.Snippet,
		&receivedMs, &m.Size, &m.AttachmentCount, &m.HasHTML, &m.IsRead, &m.IsStarred,
		&m.EnvelopeFrom, &dateMs, &m.RemoteAddr, &m.Helo, &status, &errsJSON, &m.Text, &m.HTML)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get message %d: %w", id, err)
	}
	m.ReceivedAt = time.UnixMilli(receivedMs).UTC()
	if dateMs.Valid {
		t := time.UnixMilli(dateMs.Int64).UTC()
		m.Date = &t
	}
	m.ParseStatus = domain.ParseStatus(status)
	_ = json.Unmarshal([]byte(errsJSON), &m.ParseErrors)
	m.ParseErrors = nonNilStrings(m.ParseErrors)

	m.To, m.Cc, m.Bcc, m.ReplyTo = []domain.Address{}, []domain.Address{}, []domain.Address{}, []domain.Address{}
	m.EnvelopeTo = []string{}
	rows, err := db.reader.QueryContext(ctx, `SELECT kind, name, address FROM email_recipients WHERE email_id = ? ORDER BY id`, id)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var kind string
		var a domain.Address
		if err := rows.Scan(&kind, &a.Name, &a.Address); err != nil {
			rows.Close()
			return nil, err
		}
		switch kind {
		case "to":
			m.To = append(m.To, a)
		case "cc":
			m.Cc = append(m.Cc, a)
		case "bcc":
			m.Bcc = append(m.Bcc, a)
		case "reply_to":
			m.ReplyTo = append(m.ReplyTo, a)
		case "envelope":
			m.EnvelopeTo = append(m.EnvelopeTo, a.Address)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	m.Headers = []domain.Header{}
	rows, err = db.reader.QueryContext(ctx, `SELECT name, value FROM email_headers WHERE email_id = ? ORDER BY ordinal`, id)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var h domain.Header
		if err := rows.Scan(&h.Name, &h.Value); err != nil {
			rows.Close()
			return nil, err
		}
		m.Headers = append(m.Headers, h)
	}
	rows.Close()

	m.Attachments = []domain.Attachment{}
	rows, err = db.reader.QueryContext(ctx, `SELECT id, part_index, filename, content_type, size_bytes, content_id, inline
		FROM email_attachments WHERE email_id = ? ORDER BY part_index`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var a domain.Attachment
		if err := rows.Scan(&a.ID, &a.PartIndex, &a.FileName, &a.ContentType, &a.Size, &a.ContentID, &a.Inline); err != nil {
			return nil, err
		}
		m.Attachments = append(m.Attachments, a)
	}
	return &m, rows.Err()
}

// GetRaw returns the original message bytes exactly as received.
func (db *DB) GetRaw(ctx context.Context, id int64) ([]byte, error) {
	var raw []byte
	err := db.reader.QueryRowContext(ctx, `SELECT raw FROM email_raw WHERE email_id = ?`, id).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return raw, err
}

// SetRead marks messages read or unread.
func (db *DB) SetRead(ctx context.Context, ids []int64, read bool) error {
	return db.updateFlag(ctx, "is_read", ids, read)
}

// SetStarred stars or unstars messages.
func (db *DB) SetStarred(ctx context.Context, ids []int64, starred bool) error {
	return db.updateFlag(ctx, "is_starred", ids, starred)
}

func (db *DB) updateFlag(ctx context.Context, column string, ids []int64, value bool) error {
	if len(ids) == 0 {
		return nil
	}
	placeholders, args := inClause(ids)
	args = append([]any{value}, args...)
	_, err := db.writer.ExecContext(ctx, `UPDATE emails SET `+column+` = ? WHERE id IN (`+placeholders+`)`, args...)
	return err
}

// DeleteMessages removes messages (cascading to all parts). It returns the
// number of messages deleted.
func (db *DB) DeleteMessages(ctx context.Context, ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	placeholders, args := inClause(ids)
	res, err := db.writer.ExecContext(ctx, `DELETE FROM emails WHERE id IN (`+placeholders+`)`, args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// DeleteAllMessages empties the mailbox and reclaims disk space.
func (db *DB) DeleteAllMessages(ctx context.Context) (int64, error) {
	res, err := db.writer.ExecContext(ctx, `DELETE FROM emails`)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	if _, err := db.writer.ExecContext(ctx, `PRAGMA incremental_vacuum`); err != nil {
		db.log.Warn("incremental vacuum failed", "err", err)
	}
	return n, nil
}

func inClause(ids []int64) (string, []any) {
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	return strings.TrimSuffix(strings.Repeat("?,", len(ids)), ","), args
}

func escapeLike(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(s)
}

func encodeCursor(ms, id int64) string {
	return strconv.FormatInt(ms, 36) + "." + strconv.FormatInt(id, 36)
}

func decodeCursor(c string) (int64, int64, error) {
	a, b, ok := strings.Cut(c, ".")
	if !ok {
		return 0, 0, fmt.Errorf("invalid cursor")
	}
	ms, err1 := strconv.ParseInt(a, 36, 64)
	id, err2 := strconv.ParseInt(b, 36, 64)
	if err1 != nil || err2 != nil {
		return 0, 0, fmt.Errorf("invalid cursor")
	}
	return ms, id, nil
}

func nonNilAddrs(a []domain.Address) []domain.Address {
	if a == nil {
		return []domain.Address{}
	}
	return a
}

func nonNilStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
