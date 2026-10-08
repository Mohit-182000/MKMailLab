// Package mailhttp serves message content (sanitised HTML, inline images,
// attachments, raw source) to the WebView via Wails' in-process asset
// server. No TCP port is opened.
//
// Routes (mounted under /lm by the composition root):
//
//	GET /messages/{id}/html                    sanitised HTML preview
//	GET /messages/{id}/cid/{cid}               inline part by Content-ID
//	GET /messages/{id}/attachments/{aid}       attachment (?download=1 forces save)
//	GET /messages/{id}/raw                     raw source (?download=1 → .eml)
package mailhttp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"unicode"

	"localmail/internal/domain"
	"localmail/internal/htmlsafe"
)

// Mailbox is what the handler needs from the mailbox service.
type Mailbox interface {
	Get(ctx context.Context, id int64) (*domain.Message, error)
	Raw(ctx context.Context, id int64) ([]byte, error)
	Attachment(ctx context.Context, msgID, attID int64) (domain.Attachment, []byte, error)
	InlineByCID(ctx context.Context, msgID int64, cid string) (domain.Attachment, []byte, error)
}

// Handler serves message content.
type Handler struct {
	mb     Mailbox
	log    *slog.Logger
	prefix string
	mux    *http.ServeMux
}

// New creates the handler. prefix is the mount point (e.g. "/lm"), used to
// build absolute URLs for rewritten cid: references.
func New(mb Mailbox, log *slog.Logger, prefix string) *Handler {
	h := &Handler{mb: mb, log: log, prefix: strings.TrimSuffix(prefix, "/"), mux: http.NewServeMux()}
	h.mux.HandleFunc("GET /messages/{id}/html", h.html)
	h.mux.HandleFunc("GET /messages/{id}/cid/{cid}", h.cid)
	h.mux.HandleFunc("GET /messages/{id}/attachments/{aid}", h.attachment)
	h.mux.HandleFunc("GET /messages/{id}/raw", h.raw)
	return h
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	h.mux.ServeHTTP(w, r)
}

func (h *Handler) html(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	m, err := h.mb.Get(r.Context(), id)
	if err != nil {
		h.fail(w, err)
		return
	}
	body := m.HTML
	if body == "" {
		// Text-only email: render it readably instead of a blank frame.
		body = "<pre style=\"white-space:pre-wrap;font:13px/1.5 Consolas,monospace;margin:16px\">" +
			escapeHTML(m.Text) + "</pre>"
	}
	cids := map[string]bool{}
	for _, a := range m.Attachments {
		if a.ContentID != "" {
			cids[a.ContentID] = true
		}
	}
	safe := htmlsafe.Sanitize(body, func(cid string) (string, bool) {
		if !cids[cid] {
			return "", false
		}
		return fmt.Sprintf("%s/messages/%d/cid/%s", h.prefix, id, url.PathEscape(cid)), true
	})

	w.Header().Set("Content-Security-Policy", htmlsafe.ContentSecurityPolicy)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(safe))
}

func (h *Handler) cid(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	a, content, err := h.mb.InlineByCID(r.Context(), id, r.PathValue("cid"))
	if err != nil {
		h.fail(w, err)
		return
	}
	h.serveContent(w, a, content, false)
}

func (h *Handler) attachment(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	aid, ok := pathID(w, r, "aid")
	if !ok {
		return
	}
	a, content, err := h.mb.Attachment(r.Context(), id, aid)
	if err != nil {
		h.fail(w, err)
		return
	}
	h.serveContent(w, a, content, r.URL.Query().Get("download") == "1")
}

func (h *Handler) raw(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	raw, err := h.mb.Raw(r.Context(), id)
	if err != nil {
		h.fail(w, err)
		return
	}
	w.Header().Set("Content-Security-Policy", "default-src 'none'")
	if r.URL.Query().Get("download") == "1" {
		w.Header().Set("Content-Type", "message/rfc822")
		w.Header().Set("Content-Disposition", contentDisposition("attachment", fmt.Sprintf("message-%d.eml", id)))
	} else {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	}
	_, _ = w.Write(raw)
}

// Types that are safe to display inline in the WebView. Anything else
// (HTML, SVG, scripts, executables, unknown) is always downloaded.
var inlineSafe = map[string]bool{
	"image/png": true, "image/jpeg": true, "image/gif": true, "image/webp": true, "image/bmp": true,
	"image/x-icon": true, "image/avif": true, "application/pdf": true, "text/plain": true,
	"text/csv": true, "application/json": true,
}

func (h *Handler) serveContent(w http.ResponseWriter, a domain.Attachment, content []byte, forceDownload bool) {
	ct := strings.ToLower(a.ContentType)
	if ct != "application/pdf" {
		w.Header().Set("Content-Security-Policy", "default-src 'none'; img-src 'self' data:; style-src 'unsafe-inline'")
	}
	if inlineSafe[ct] && !forceDownload {
		if strings.HasPrefix(ct, "text/") || ct == "application/json" {
			ct += "; charset=utf-8"
		}
		w.Header().Set("Content-Type", ct)
		w.Header().Set("Content-Disposition", contentDisposition("inline", a.FileName))
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", contentDisposition("attachment", a.FileName))
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(content)))
	_, _ = w.Write(content)
}

func (h *Handler) fail(w http.ResponseWriter, err error) {
	if errors.Is(err, domain.ErrNotFound) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	h.log.Error("content request failed", "err", err)
	http.Error(w, "internal error", http.StatusInternalServerError)
}

func pathID(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "bad id", http.StatusBadRequest)
		return 0, false
	}
	return id, true
}

// SafeFileName strips directory components and characters that are invalid
// or dangerous in Windows file names, so a hostile attachment name can never
// cause path traversal when saved.
func SafeFileName(name string) string {
	name = path.Base(strings.ReplaceAll(name, `\`, "/"))
	var b strings.Builder
	for _, r := range name {
		if r < 32 || strings.ContainsRune(`<>:"/\|?*`, r) || r == unicode.ReplacementChar {
			b.WriteRune('_')
			continue
		}
		b.WriteRune(r)
	}
	out := strings.Trim(b.String(), " .")
	if out == "" || out == "." || out == ".." {
		return "attachment"
	}
	upper := strings.ToUpper(strings.SplitN(out, ".", 2)[0])
	switch upper {
	case "CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "LPT1", "LPT2", "LPT3":
		out = "_" + out
	}
	if len(out) > 180 {
		out = out[:180]
	}
	return out
}

func contentDisposition(kind, name string) string {
	return mime.FormatMediaType(kind, map[string]string{"filename": SafeFileName(name)})
}

func escapeHTML(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;").Replace(s)
}
