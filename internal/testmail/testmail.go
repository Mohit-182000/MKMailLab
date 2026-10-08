// Package testmail sends a test email through MKMailLab's own SMTP server,
// exercising the same path a real application would.
package testmail

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

// Message is a test email composed in the UI.
type Message struct {
	From    string `json:"from"`
	To      string `json:"to"` // comma-separated
	Subject string `json:"subject"`
	Body    string `json:"body"`
	IsHTML  bool   `json:"isHtml"`
}

// Target is where to deliver.
type Target struct {
	Host     string
	Port     int
	Username string // empty = no AUTH
	Password string
}

// Send delivers m to target. It honours ctx for the dial and overall deadline.
func Send(ctx context.Context, target Target, m Message) error {
	from, err := mail.ParseAddress(strings.TrimSpace(m.From))
	if err != nil {
		return fmt.Errorf("invalid From address: %w", err)
	}
	toList, err := mail.ParseAddressList(strings.TrimSpace(m.To))
	if err != nil || len(toList) == 0 {
		return fmt.Errorf("invalid To address list")
	}

	host := target.Host
	if host == "0.0.0.0" || host == "::" || host == "" {
		host = "127.0.0.1" // listening on all interfaces; connect via loopback
	}
	addr := net.JoinHostPort(host, strconv.Itoa(target.Port))

	dialer := net.Dialer{Timeout: 5 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("could not connect to %s: %w", addr, err)
	}
	deadline := time.Now().Add(15 * time.Second)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	_ = conn.SetDeadline(deadline)

	c, err := smtp.NewClient(conn, host)
	if err != nil {
		_ = conn.Close()
		return err
	}
	defer c.Close()

	if err := c.Hello("mkmaillab-test"); err != nil {
		return err
	}
	if target.Username != "" {
		if ok, _ := c.Extension("AUTH"); ok {
			if err := c.Auth(smtp.PlainAuth("", target.Username, target.Password, host)); err != nil {
				return fmt.Errorf("authentication failed: %w", err)
			}
		}
	}
	if err := c.Mail(from.Address); err != nil {
		return err
	}
	for _, a := range toList {
		if err := c.Rcpt(a.Address); err != nil {
			return err
		}
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(Build(from, toList, m, time.Now())); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}

// Build renders the RFC 5322 message.
func Build(from *mail.Address, to []*mail.Address, m Message, now time.Time) []byte {
	var b bytes.Buffer
	addrs := make([]string, len(to))
	for i, a := range to {
		addrs[i] = a.String()
	}
	subject := m.Subject
	if strings.TrimSpace(subject) == "" {
		subject = "MKMailLab test email"
	}
	contentType := "text/plain; charset=utf-8"
	if m.IsHTML {
		contentType = "text/html; charset=utf-8"
	}

	fmt.Fprintf(&b, "From: %s\r\n", from.String())
	fmt.Fprintf(&b, "To: %s\r\n", strings.Join(addrs, ", "))
	fmt.Fprintf(&b, "Subject: %s\r\n", mime.QEncoding.Encode("utf-8", subject))
	fmt.Fprintf(&b, "Date: %s\r\n", now.Format(time.RFC1123Z))
	fmt.Fprintf(&b, "Message-ID: <%s@mkmaillab.local>\r\n", randomID())
	b.WriteString("MIME-Version: 1.0\r\n")
	fmt.Fprintf(&b, "Content-Type: %s\r\n", contentType)
	b.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
	b.WriteString("X-Mailer: MKMailLab test sender\r\n\r\n")

	qp := quotedprintable.NewWriter(&b)
	_, _ = qp.Write([]byte(m.Body))
	_ = qp.Close()
	b.WriteString("\r\n")
	return b.Bytes()
}

func randomID() string {
	var buf [12]byte
	_, _ = rand.Read(buf[:])
	return hex.EncodeToString(buf[:])
}
