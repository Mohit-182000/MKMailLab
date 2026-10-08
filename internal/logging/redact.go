package logging

import (
	"log/slog"
	"strings"
)

// Redacted replaces the value of any attribute whose key looks sensitive.
const Redacted = "[REDACTED]"

var sensitiveKeyParts = []string{
	"password", "passwd", "secret", "token", "authorization",
	"credential", "apikey", "api_key", "cookie",
}

// IsSensitiveKey reports whether an attribute key should never be logged in
// clear text. Matching is case-insensitive and substring-based so that keys
// like "smtp_password" or "AuthToken" are covered.
func IsSensitiveKey(key string) bool {
	k := strings.ToLower(key)
	for _, part := range sensitiveKeyParts {
		if strings.Contains(k, part) {
			return true
		}
	}
	return false
}

// redactAttr is a slog ReplaceAttr function shared by every sink.
func redactAttr(_ []string, a slog.Attr) slog.Attr {
	if a.Value.Kind() != slog.KindGroup && IsSensitiveKey(a.Key) {
		return slog.String(a.Key, Redacted)
	}
	return a
}
