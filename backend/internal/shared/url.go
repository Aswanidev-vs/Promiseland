package shared

import (
	"net/url"
	"regexp"
	"strings"
)

// MaxURLLen bounds a stored external URL.
const MaxURLLen = 2048

// ValidOptionalURL reports whether value is empty or an absolute http(s) URL
// with a host. Stored URLs are rendered as links and images, so other schemes
// (javascript:, data:, file:) are refused at the boundary rather than trusted
// to the browser.
func ValidOptionalURL(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return true
	}
	if len(value) > MaxURLLen {
		return false
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	return parsed.Host != ""
}

// mediaPathPattern is the exact shape the upload endpoint produces: a 24-byte
// random key in hex plus an allowed image extension.
var mediaPathPattern = regexp.MustCompile(`^/media/[0-9a-f]{48}\.(jpg|jpeg|png|gif|webp)$`)

// ValidProfileImage reports whether value is empty, an absolute http(s) URL, or
// a path to one of our own uploads. Profile images are rendered as img src, so
// the same-origin form needs no scheme and cannot leave the app.
func ValidProfileImage(value string) bool {
	value = strings.TrimSpace(value)
	return value == "" || ValidOptionalURL(value) || mediaPathPattern.MatchString(value)
}
