package shared

import (
	"net/url"
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
