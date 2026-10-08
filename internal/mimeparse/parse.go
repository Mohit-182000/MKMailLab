// Package mimeparse turns raw RFC 5322 messages into domain data.
//
// Parsing never fails outright: malformed input yields a Result with
// Status=partial/failed and human-readable Errors, and the caller still stores
// the raw bytes. A panic inside the underlying MIME library is recovered.
package mimeparse

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"mime"
	"net/mail"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/jhillyerd/enmime/v2"
	"golang.org/x/text/encoding/htmlindex"

	"localmail/internal/domain"
)

// Result is the parsed form of a message.
type Result struct {
	MessageID   string
	Subject     string
	From        domain.Address
	To          []domain.Address
	Cc          []domain.Address
	Bcc         []domain.Address
	ReplyTo     []domain.Address
	Date        *time.Time
	Headers     []domain.Header
	Text        string
	HTML        string
	Attachments []domain.Attachment
	Snippet     string
	Status      domain.ParseStatus
	Errors      []string
}

const snippetLen = 200

var wordDecoder = &mime.WordDecoder{CharsetReader: charsetReader}

// Parse parses raw. It never panics.
func Parse(raw []byte) (res Result) {
	res.Status = domain.ParseOK
	res.To, res.Cc, res.Bcc, res.ReplyTo = []domain.Address{}, []domain.Address{}, []domain.Address{}, []domain.Address{}
	res.Attachments = []domain.Attachment{}

	// Headers are read independently of the MIME library so they are kept in
	// their original order and survive even if body parsing fails.
	res.Headers = readHeaders(raw)
	h := headerLookup(res.Headers)
	res.MessageID = strings.TrimSpace(h("Message-Id"))
	res.Subject = decodeWords(h("Subject"))
	res.From = firstAddress(parseAddresses(h("From"), &res))
	res.To = parseAddresses(h("To"), &res)
	res.Cc = parseAddresses(h("Cc"), &res)
	res.Bcc = parseAddresses(h("Bcc"), &res)
	res.ReplyTo = parseAddresses(h("Reply-To"), &res)
	if d := strings.TrimSpace(h("Date")); d != "" {
		if t, err := mail.ParseDate(d); err == nil {
			t = t.UTC()
			res.Date = &t
		} else {
			res.addError(fmt.Sprintf("invalid Date header %q", d))
		}
	}

	env, err := readEnvelope(raw)
	if err != nil {
		res.Status = domain.ParseFailed
		res.addError("could not parse MIME structure: " + err.Error())
		res.Text = bodyAfterHeaders(raw)
		res.Snippet = makeSnippet(res.Text)
		return res
	}

	res.Text = env.Text
	res.HTML = env.HTML
	for _, e := range env.Errors {
		// Non-severe entries are informational (e.g. "text generated from
		// HTML") and must not flag the message as malformed.
		if e.Severe {
			res.addError(e.Error())
		}
	}
	for i, p := range Parts(env) {
		res.Attachments = append(res.Attachments, domain.Attachment{
			PartIndex:   i,
			FileName:    fileName(p, i),
			ContentType: p.ContentType,
			Size:        int64(len(p.Content)),
			ContentID:   strings.Trim(p.ContentID, "<>"),
			Inline:      strings.EqualFold(p.Disposition, "inline") || (p.Disposition == "" && p.ContentID != ""),
		})
	}
	res.Snippet = makeSnippet(res.Text)
	return res
}

func (r *Result) addError(msg string) {
	r.Errors = append(r.Errors, msg)
	if r.Status == domain.ParseOK {
		r.Status = domain.ParsePartial
	}
}

// readEnvelope wraps enmime, converting panics into errors.
func readEnvelope(raw []byte) (env *enmime.Envelope, err error) {
	defer func() {
		if p := recover(); p != nil {
			env, err = nil, fmt.Errorf("parser panic: %v", p)
		}
	}()
	return enmime.ReadEnvelope(bytes.NewReader(raw))
}

// Parts returns attachment and inline parts in a stable order. The index of a
// part in this slice is its PartIndex; ExtractPart relies on the same order.
func Parts(env *enmime.Envelope) []*enmime.Part {
	parts := make([]*enmime.Part, 0, len(env.Attachments)+len(env.Inlines))
	parts = append(parts, env.Attachments...)
	parts = append(parts, env.Inlines...)
	for _, p := range env.OtherParts {
		if p.ContentID != "" || p.FileName != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

// ExtractPart re-parses raw and returns the part at index.
func ExtractPart(raw []byte, index int) (*enmime.Part, error) {
	env, err := readEnvelope(raw)
	if err != nil {
		return nil, err
	}
	parts := Parts(env)
	if index < 0 || index >= len(parts) {
		return nil, domain.ErrNotFound
	}
	return parts[index], nil
}

func fileName(p *enmime.Part, i int) string {
	if p.FileName != "" {
		return p.FileName
	}
	ext := ""
	if exts, _ := mime.ExtensionsByType(p.ContentType); len(exts) > 0 {
		ext = exts[0]
	}
	return fmt.Sprintf("part-%d%s", i+1, ext)
}

// readHeaders reads the header block, unfolding continuation lines and
// decoding RFC 2047 encoded words, preserving order and duplicates.
func readHeaders(raw []byte) []domain.Header {
	headers := []domain.Header{}
	sc := bufio.NewScanner(bytes.NewReader(raw))
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r")
		if line == "" {
			break
		}
		if (line[0] == ' ' || line[0] == '\t') && len(headers) > 0 {
			last := &headers[len(headers)-1]
			last.Value += " " + strings.TrimSpace(line)
			continue
		}
		name, value, ok := strings.Cut(line, ":")
		if !ok {
			continue // garbage line in header block; skip it
		}
		headers = append(headers, domain.Header{Name: strings.TrimSpace(name), Value: strings.TrimSpace(value)})
	}
	for i := range headers {
		headers[i].Value = decodeWords(headers[i].Value)
	}
	return headers
}

func headerLookup(hs []domain.Header) func(string) string {
	return func(name string) string {
		for _, h := range hs {
			if strings.EqualFold(h.Name, name) {
				return h.Value
			}
		}
		return ""
	}
}

func decodeWords(s string) string {
	if !strings.Contains(s, "=?") {
		return s
	}
	if d, err := wordDecoder.DecodeHeader(s); err == nil {
		return d
	}
	return s
}

// parseAddresses parses an address list leniently: if strict parsing fails,
// it falls back to splitting on commas so nothing is silently dropped.
func parseAddresses(v string, res *Result) []domain.Address {
	v = strings.TrimSpace(v)
	out := []domain.Address{}
	if v == "" {
		return out
	}
	parser := mail.AddressParser{WordDecoder: wordDecoder}
	if list, err := parser.ParseList(v); err == nil {
		for _, a := range list {
			out = append(out, domain.Address{Name: a.Name, Address: a.Address})
		}
		return out
	}
	res.addError(fmt.Sprintf("malformed address list %q", v))
	for _, part := range strings.Split(v, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if a, err := parser.Parse(part); err == nil {
			out = append(out, domain.Address{Name: a.Name, Address: a.Address})
		} else {
			out = append(out, domain.Address{Address: part})
		}
	}
	return out
}

func firstAddress(list []domain.Address) domain.Address {
	if len(list) == 0 {
		return domain.Address{}
	}
	return list[0]
}

func bodyAfterHeaders(raw []byte) string {
	if i := bytes.Index(raw, []byte("\r\n\r\n")); i >= 0 {
		return string(raw[i+4:])
	}
	if i := bytes.Index(raw, []byte("\n\n")); i >= 0 {
		return string(raw[i+2:])
	}
	return ""
}

// dividerRun matches decorative separators produced by HTML→text conversion.
var dividerRun = regexp.MustCompile(`[-=_*~#]{3,}`)

func makeSnippet(text string) string {
	text = dividerRun.ReplaceAllString(text, " ")
	var b strings.Builder
	space := false
	n := 0
	for _, r := range text {
		if unicode.IsSpace(r) {
			space = true
			continue
		}
		if space && b.Len() > 0 {
			b.WriteByte(' ')
			n++
		}
		space = false
		b.WriteRune(r)
		n++
		if n >= snippetLen {
			break
		}
	}
	return b.String()
}

func charsetReader(charset string, input io.Reader) (io.Reader, error) {
	enc, err := htmlindex.Get(charset)
	if err != nil {
		return nil, err
	}
	return enc.NewDecoder().Reader(input), nil
}
