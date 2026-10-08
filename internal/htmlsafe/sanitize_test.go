package htmlsafe

import (
	"strings"
	"testing"
)

func TestSanitizeRemovesActiveContent(t *testing.T) {
	in := `<html><head><base href="http://evil/"><meta http-equiv="refresh" content="0;url=http://evil">
<script>alert(1)</script><style>a > b { color: red }</style></head>
<body onload="steal()">
<p onclick="x()" style="color:blue">Hello <b>world</b></p>
<a href="javascript:alert(1)">bad</a><a href=" JaVa&#x09;script:alert(1)">bad2</a>
<a href="https://example.com">good</a>
<img src="cid:logo@x" onerror="x()"><img src="data:image/svg+xml;base64,AAAA">
<iframe src="http://evil"><p>inside iframe</p></iframe>
<object data="x.swf"></object><embed src="x.swf">
<form action="http://evil"><input formaction="http://evil"></form>
<div style="width: expression(alert(1))">ie</div>
<svg><script>alert(2)</script></svg>
<template><script>alert(3)</script></template>
</body></html>`

	out := Sanitize(in, func(cid string) (string, bool) {
		if cid == "logo@x" {
			return "/lm/messages/1/cid/logo@x", true
		}
		return "", false
	})

	for _, bad := range []string{
		"<script", "alert(", "onload", "onclick", "onerror", "javascript", "http://evil",
		"<iframe", "inside iframe", "<object", "<embed", "expression(", "data:image/svg",
		`http-equiv="refresh"`, "formaction",
	} {
		if strings.Contains(strings.ToLower(out), strings.ToLower(bad)) {
			t.Errorf("output still contains %q:\n%s", bad, out)
		}
	}
	for _, good := range []string{
		"a > b { color: red }", `style="color:blue"`, "<b>world</b>", `href="https://example.com"`,
		`src="/lm/messages/1/cid/logo@x"`, `<base target="_blank">`,
	} {
		if !strings.Contains(out, good) {
			t.Errorf("output lost %q:\n%s", good, out)
		}
	}
}

func TestSanitizeMalformedHTML(t *testing.T) {
	inputs := []string{"", "<", "<p", "<script>never closed", "<<<>>>", "<div><span>unclosed", "\x00\xff<b>x"}
	for _, in := range inputs {
		out := Sanitize(in, nil)
		if strings.Contains(out, "<script") {
			t.Errorf("%q produced script: %q", in, out)
		}
	}
}

func TestUnknownCIDLeftUntouched(t *testing.T) {
	out := Sanitize(`<img src="cid:missing">`, func(string) (string, bool) { return "", false })
	if !strings.Contains(out, `src="cid:missing"`) {
		t.Fatalf("got %s", out)
	}
}
