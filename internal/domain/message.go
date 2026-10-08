// Package domain defines the core entities shared by the SMTP, ingest,
// storage and API layers. It has no dependencies on other internal packages.
package domain

import (
	"errors"
	"time"
)

// ErrNotFound is returned by stores when a requested entity does not exist.
var ErrNotFound = errors.New("not found")

// Address is an email address with an optional display name.
type Address struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

// Header is a single message header, in original order.
type Header struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Attachment describes a MIME part that is a file. Its bytes are not stored
// separately; they are extracted from the raw message on demand.
type Attachment struct {
	ID          int64  `json:"id"`
	PartIndex   int    `json:"partIndex"`
	FileName    string `json:"fileName"`
	ContentType string `json:"contentType"`
	Size        int64  `json:"size"`
	ContentID   string `json:"contentId"`
	Inline      bool   `json:"inline"`
}

// ParseStatus records how well a message could be parsed.
type ParseStatus string

const (
	ParseOK      ParseStatus = "ok"      // parsed cleanly
	ParsePartial ParseStatus = "partial" // parsed with recoverable problems
	ParseFailed  ParseStatus = "failed"  // only the raw source is usable
)

// MessageSummary is the lightweight projection used by the inbox list. It
// never contains bodies so listing stays fast with large mailboxes.
type MessageSummary struct {
	ID              int64     `json:"id"`
	MessageID       string    `json:"messageId"`
	Subject         string    `json:"subject"`
	From            Address   `json:"from"`
	To              []Address `json:"to"`
	Snippet         string    `json:"snippet"`
	ReceivedAt      time.Time `json:"receivedAt"`
	Size            int64     `json:"size"`
	AttachmentCount int       `json:"attachmentCount"`
	HasHTML         bool      `json:"hasHtml"`
	IsRead          bool      `json:"isRead"`
	IsStarred       bool      `json:"isStarred"`
}

// Message is the full detail view of a captured email.
type Message struct {
	MessageSummary
	Cc           []Address    `json:"cc"`
	Bcc          []Address    `json:"bcc"`
	ReplyTo      []Address    `json:"replyTo"`
	EnvelopeFrom string       `json:"envelopeFrom"`
	EnvelopeTo   []string     `json:"envelopeTo"`
	Date         *time.Time   `json:"date"`
	Headers      []Header     `json:"headers"`
	Text         string       `json:"text"`
	HTML         string       `json:"html"`
	Attachments  []Attachment `json:"attachments"`
	ParseStatus  ParseStatus  `json:"parseStatus"`
	ParseErrors  []string     `json:"parseErrors"`
	RemoteAddr   string       `json:"remoteAddr"`
	Helo         string       `json:"helo"`
}

// NewMessage is everything needed to persist a freshly received email.
type NewMessage struct {
	Message
	Raw []byte
}

// ListQuery selects a page of message summaries, newest first.
type ListQuery struct {
	// Search is a free-text filter over subject, sender and recipients.
	Search string `json:"search"`
	// UnreadOnly restricts results to unread messages.
	UnreadOnly bool `json:"unreadOnly"`
	// Cursor is the opaque value from a previous page's NextCursor.
	Cursor string `json:"cursor"`
	// Limit is the page size (clamped to 1..200, default 50).
	Limit int `json:"limit"`
}

// Page is one page of results.
type Page struct {
	Items      []MessageSummary `json:"items"`
	NextCursor string           `json:"nextCursor"`
	Total      int              `json:"total"`
	Unread     int              `json:"unread"`
}
