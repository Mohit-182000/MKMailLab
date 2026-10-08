// Package htmlsafe neutralises active content in captured email HTML while
// preserving its layout and CSS, so the preview stays faithful.
//
// This is one layer of defence. The preview is additionally served with a
// strict Content-Security-Policy (no scripts) inside an iframe sandbox
// without allow-scripts; any one layer alone blocks script execution.
package htmlsafe

import (
	"bytes"
	"io"
	"strings"

	"golang.org/x/net/html"
)

// CIDResolver maps a Content-ID (without angle brackets) to a URL. It returns
// false if the CID is unknown.
type CIDResolver func(cid string) (string, bool)

// ContentSecurityPolicy is sent with every rendered email. Scripts, plugins,
// frames and form submission are forbidden; images, styles, fonts and media
// may load from anywhere (remote images are expected in real emails).
const ContentSecurityPolicy = "default-src 'none'; img-src * data: blob:; style-src * 'unsafe-inline'; " +
	"font-src * data:; media-src * data:; form-action 'none'; frame-ancestors 'self'; base-uri 'none'"

// Elements removed together with everything inside them.
var dropWithContent = map[string]bool{
	"script": true, "iframe": true, "object": true, "applet": true,
	"frameset": true, "noembed": true, "template": true,
}

// Elements whose tags are removed (they are void or their content is harmless).
var dropTag = map[string]bool{
	"base": true, "embed": true, "frame": true, "portal": true,
}

// Attributes that carry URLs and must not use script-capable schemes.
var urlAttrs = map[string]bool{
	"href": true, "src": true, "action": true, "formaction": true, "background": true,
	"poster": true, "xlink:href": true, "lowsrc": true, "dynsrc": true, "cite": true, "longdesc": true,
}

// Sanitize returns html with active content removed. A <base target="_blank">
// is prepended so link clicks never navigate the preview frame itself.
func Sanitize(src string, resolve CIDResolver) string {
	var out bytes.Buffer
	out.Grow(len(src) + 32)
	out.WriteString(`<base target="_blank">`)

	z := html.NewTokenizer(strings.NewReader(src))
	skipDepth := 0
	skipTag := ""

	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			if z.Err() == io.EOF {
				break
			}
			break // malformed tail: emit what we have
		}

		if skipDepth > 0 {
			switch tt {
			case html.StartTagToken:
				if name, _ := z.TagName(); string(name) == skipTag {
					skipDepth++
				}
			case html.EndTagToken:
				if name, _ := z.TagName(); string(name) == skipTag {
					skipDepth--
				}
			}
			continue
		}

		switch tt {
		case html.StartTagToken, html.SelfClosingTagToken:
			tok := z.Token()
			name := strings.ToLower(tok.Data)
			if dropWithContent[name] {
				if tt == html.StartTagToken {
					skipDepth, skipTag = 1, name
				}
				continue
			}
			if dropTag[name] || (name == "meta" && isRefresh(tok)) {
				continue
			}
			tok.Attr = cleanAttrs(name, tok.Attr, resolve)
			out.WriteString(tok.String())
		case html.EndTagToken:
			tok := z.Token()
			name := strings.ToLower(tok.Data)
			if dropTag[name] || dropWithContent[name] {
				continue
			}
			out.WriteString(tok.String())
		default:
			// Text, comments and doctype are emitted verbatim. Raw bytes keep
			// CSS inside <style> intact (re-escaping would break selectors).
			out.Write(z.Raw())
		}
	}
	return out.String()
}

func isRefresh(t html.Token) bool {
	for _, a := range t.Attr {
		if strings.EqualFold(a.Key, "http-equiv") && strings.EqualFold(strings.TrimSpace(a.Val), "refresh") {
			return true
		}
	}
	return false
}

func cleanAttrs(tag string, attrs []html.Attribute, resolve CIDResolver) []html.Attribute {
	out := attrs[:0]
	for _, a := range attrs {
		key := strings.ToLower(a.Key)
		if a.Namespace != "" {
			key = strings.ToLower(a.Namespace) + ":" + key
		}
		switch {
		case strings.HasPrefix(key, "on"): // event handlers
			continue
		case key == "srcdoc" || key == "formaction" || (key == "action" && tag == "form"):
			continue
		case key == "style" && hasDangerousCSS(a.Val):
			continue
		case urlAttrs[key] || key == "srcset":
			v := strings.TrimSpace(a.Val)
			if isDangerousURL(v) {
				continue
			}
			if resolve != nil && len(v) > 4 && strings.EqualFold(v[:4], "cid:") {
				if u, ok := resolve(strings.Trim(v[4:], "<>")); ok {
					a.Val = u
				}
			}
		}
		out = append(out, a)
	}
	return out
}

func isDangerousURL(v string) bool {
	// Browsers ignore whitespace/control characters inside the scheme.
	var b strings.Builder
	for _, r := range v {
		if r > ' ' {
			b.WriteRune(r)
		}
		if b.Len() > 16 {
			break
		}
	}
	s := strings.ToLower(b.String())
	return strings.HasPrefix(s, "javascript:") || strings.HasPrefix(s, "vbscript:") ||
		strings.HasPrefix(s, "data:text/html") || strings.HasPrefix(s, "data:application/xhtml") ||
		strings.HasPrefix(s, "data:image/svg")
}

func hasDangerousCSS(v string) bool {
	s := strings.ToLower(v)
	return strings.Contains(s, "expression(") || strings.Contains(s, "javascript:") || strings.Contains(s, "-moz-binding")
}
