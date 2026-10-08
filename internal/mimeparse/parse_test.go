package mimeparse

import (
	"strings"
	"testing"

	"localmail/internal/domain"
)

func crlf(s string) []byte { return []byte(strings.ReplaceAll(s, "\n", "\r\n")) }

func TestParsePlainText(t *testing.T) {
	raw := crlf(`From: "App" <app@example.com>
To: user@example.com, "Second" <second@example.com>
Cc: cc@example.com
Subject: =?UTF-8?B?V2VsY29tZSDwn46J?=
Date: Wed, 08 Oct 2026 10:00:00 +0530
Message-ID: <abc@example.com>
X-Long: first
  continued

Hello   there,
this is a test.
`)
	r := Parse(raw)
	if r.Status != domain.ParseOK {
		t.Fatalf("status %s errors %v", r.Status, r.Errors)
	}
	if r.Subject != "Welcome 🎉" {
		t.Errorf("subject = %q", r.Subject)
	}
	if r.From.Name != "App" || r.From.Address != "app@example.com" {
		t.Errorf("from = %+v", r.From)
	}
	if len(r.To) != 2 || r.To[1].Name != "Second" || len(r.Cc) != 1 {
		t.Errorf("to=%v cc=%v", r.To, r.Cc)
	}
	if r.Date == nil || r.Date.Hour() != 4 || r.Date.Minute() != 30 {
		t.Errorf("date = %v (want UTC 04:30)", r.Date)
	}
	if r.MessageID != "<abc@example.com>" {
		t.Errorf("message id = %q", r.MessageID)
	}
	if !strings.Contains(r.Text, "this is a test.") || r.HTML != "" {
		t.Errorf("text=%q html=%q", r.Text, r.HTML)
	}
	if r.Snippet != "Hello there, this is a test." {
		t.Errorf("snippet = %q", r.Snippet)
	}
	var long string
	for _, h := range r.Headers {
		if h.Name == "X-Long" {
			long = h.Value
		}
	}
	if long != "first continued" {
		t.Errorf("folded header = %q", long)
	}
	if r.Headers[0].Name != "From" {
		t.Errorf("header order not preserved: %v", r.Headers)
	}
}

func TestParseMultipartWithAttachments(t *testing.T) {
	raw := crlf(`From: app@example.com
To: user@example.com
Subject: Invoice
MIME-Version: 1.0
Content-Type: multipart/mixed; boundary="mix"

--mix
Content-Type: multipart/related; boundary="rel"

--rel
Content-Type: multipart/alternative; boundary="alt"

--alt
Content-Type: text/plain; charset=utf-8

Your invoice
--alt
Content-Type: text/html; charset=utf-8

<p>Your <b>invoice</b> <img src="cid:logo"></p>
--alt--
--rel
Content-Type: image/png
Content-ID: <logo>
Content-Disposition: inline
Content-Transfer-Encoding: base64

iVBORw0KGgo=
--rel--
--mix
Content-Type: application/pdf; name="invoice.pdf"
Content-Disposition: attachment; filename="invoice.pdf"
Content-Transfer-Encoding: base64

JVBERi0xLjQK
--mix--
`)
	r := Parse(raw)
	if r.Status == domain.ParseFailed {
		t.Fatalf("failed: %v", r.Errors)
	}
	if !strings.Contains(r.HTML, "<b>invoice</b>") || !strings.Contains(r.Text, "Your invoice") {
		t.Errorf("html=%q text=%q", r.HTML, r.Text)
	}
	if len(r.Attachments) != 2 {
		t.Fatalf("attachments = %+v", r.Attachments)
	}
	pdf, logo := r.Attachments[0], r.Attachments[1]
	if pdf.FileName != "invoice.pdf" || pdf.ContentType != "application/pdf" || pdf.Inline || pdf.Size != 9 {
		t.Errorf("pdf = %+v", pdf)
	}
	if logo.ContentID != "logo" || !logo.Inline || logo.ContentType != "image/png" {
		t.Errorf("logo = %+v", logo)
	}

	part, err := ExtractPart(raw, pdf.PartIndex)
	if err != nil || string(part.Content) != "%PDF-1.4\n" {
		t.Errorf("extract pdf: %q %v", part.Content, err)
	}
	if _, err := ExtractPart(raw, 99); err == nil {
		t.Error("expected error for missing part")
	}
}

func TestParseLegacyCharset(t *testing.T) {
	raw := crlf("From: a@example.com\nSubject: =?ISO-8859-1?Q?Caf=E9?=\nContent-Type: text/plain; charset=windows-1252\n\nna\xefve caf\xe9\n")
	r := Parse(raw)
	if r.Subject != "Café" {
		t.Errorf("subject = %q", r.Subject)
	}
	if !strings.Contains(r.Text, "naïve café") {
		t.Errorf("text = %q", r.Text)
	}
}

func TestParseMalformedNeverPanics(t *testing.T) {
	inputs := [][]byte{
		nil,
		[]byte("garbage with no headers"),
		[]byte("\r\n\r\n\r\n"),
		crlf("From: <<<broken\nTo: ,,,\nDate: yesterday\nContent-Type: multipart/mixed; boundary=\"x\"\n\n--x\nContent-Type: text/plain\n\nunterminated"),
		crlf("Content-Type: multipart/mixed\n\nno boundary param"),
		crlf("Content-Transfer-Encoding: base64\n\n!!!not base64!!!"),
		[]byte(strings.Repeat("X-Header: v\r\n", 5000) + "\r\nbody"),
	}
	for i, in := range inputs {
		r := Parse(in)
		if r.To == nil || r.Attachments == nil || r.Headers == nil {
			t.Errorf("case %d: nil slices in result", i)
		}
	}

	r := Parse(inputs[3])
	if r.Status == domain.ParseOK || len(r.Errors) == 0 {
		t.Errorf("malformed input should report problems: %s %v", r.Status, r.Errors)
	}
}

func TestHTMLOnlyIsNotFlaggedAsProblem(t *testing.T) {
	raw := crlf("From: a@example.com\nSubject: html\nContent-Type: text/html; charset=utf-8\n\n<div>Acme Store</div><hr><p>Your ---------- order shipped</p>\n")
	r := Parse(raw)
	if r.Status != domain.ParseOK {
		t.Fatalf("informational notes must not degrade status: %s %v", r.Status, r.Errors)
	}
	if strings.Contains(r.Snippet, "---") {
		t.Errorf("snippet kept divider: %q", r.Snippet)
	}
}
