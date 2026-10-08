package mailhttp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"localmail/internal/domain"
	"localmail/internal/logging"
)

type fakeMailbox struct{}

func (fakeMailbox) Get(_ context.Context, id int64) (*domain.Message, error) {
	if id != 1 {
		return nil, domain.ErrNotFound
	}
	return &domain.Message{
		HTML:        `<p onclick="x()">Hi <img src="cid:logo"></p><script>alert(1)</script>`,
		Attachments: []domain.Attachment{{ID: 7, FileName: "logo.png", ContentType: "image/png", ContentID: "logo", Inline: true}},
	}, nil
}

func (fakeMailbox) Raw(_ context.Context, id int64) ([]byte, error) {
	if id != 1 {
		return nil, domain.ErrNotFound
	}
	return []byte("Subject: x\r\n\r\nbody"), nil
}

func (fakeMailbox) Attachment(_ context.Context, _, aid int64) (domain.Attachment, []byte, error) {
	switch aid {
	case 7:
		return domain.Attachment{FileName: "logo.png", ContentType: "image/png"}, []byte("PNG"), nil
	case 8:
		return domain.Attachment{FileName: `..\..\evil.html`, ContentType: "text/html"}, []byte("<script>"), nil
	}
	return domain.Attachment{}, nil, domain.ErrNotFound
}

func (fakeMailbox) InlineByCID(_ context.Context, _ int64, cid string) (domain.Attachment, []byte, error) {
	if cid == "logo" {
		return domain.Attachment{FileName: "logo.png", ContentType: "image/png"}, []byte("PNG"), nil
	}
	return domain.Attachment{}, nil, domain.ErrNotFound
}

func get(t *testing.T, h http.Handler, target string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
	return rec
}

func TestHTMLIsSanitisedWithCSP(t *testing.T) {
	h := New(fakeMailbox{}, logging.Discard(), "/lm")
	rec := get(t, h, "/messages/1/html")
	if rec.Code != 200 {
		t.Fatalf("code %d", rec.Code)
	}
	body := rec.Body.String()
	if strings.Contains(body, "<script") || strings.Contains(body, "onclick") {
		t.Fatalf("unsanitised: %s", body)
	}
	if !strings.Contains(body, `src="/lm/messages/1/cid/logo"`) {
		t.Fatalf("cid not rewritten: %s", body)
	}
	if csp := rec.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "default-src 'none'") {
		t.Fatalf("missing CSP: %q", csp)
	}
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("missing nosniff")
	}
}

func TestAttachmentsAndRaw(t *testing.T) {
	h := New(fakeMailbox{}, logging.Discard(), "/lm")

	rec := get(t, h, "/messages/1/attachments/7")
	if rec.Header().Get("Content-Type") != "image/png" || !strings.HasPrefix(rec.Header().Get("Content-Disposition"), "inline") {
		t.Errorf("image should be inline: %v", rec.Header())
	}

	// HTML attachment: never rendered inline, name made safe.
	rec = get(t, h, "/messages/1/attachments/8")
	if rec.Header().Get("Content-Type") != "application/octet-stream" {
		t.Errorf("html attachment served as %q", rec.Header().Get("Content-Type"))
	}
	if cd := rec.Header().Get("Content-Disposition"); !strings.Contains(cd, "attachment") || strings.Contains(cd, "..") || strings.Contains(cd, `\`) {
		t.Errorf("unsafe disposition: %q", cd)
	}

	rec = get(t, h, "/messages/1/raw?download=1")
	if rec.Header().Get("Content-Type") != "message/rfc822" || !strings.Contains(rec.Header().Get("Content-Disposition"), "message-1.eml") {
		t.Errorf("raw download headers: %v", rec.Header())
	}

	for _, target := range []string{"/messages/2/html", "/messages/1/attachments/99", "/messages/1/cid/nope"} {
		if rec := get(t, h, target); rec.Code != 404 {
			t.Errorf("%s: code %d, want 404", target, rec.Code)
		}
	}
	if rec := get(t, h, "/messages/abc/html"); rec.Code != 400 {
		t.Errorf("bad id code %d", rec.Code)
	}
}

func TestSafeFileName(t *testing.T) {
	cases := map[string]string{
		"report.pdf":             "report.pdf",
		`..\..\Windows\evil.exe`: "evil.exe",
		"../../etc/passwd":       "passwd",
		"a<b>c:d|e?.txt":         "a_b_c_d_e_.txt",
		"CON.txt":                "_CON.txt",
		"":                       "attachment",
		"...":                    "attachment",
		"name. ":                 "name",
	}
	for in, want := range cases {
		if got := SafeFileName(in); got != want {
			t.Errorf("SafeFileName(%q) = %q, want %q", in, got, want)
		}
	}
}
