package shared

import (
	"strings"
	"testing"
)

func TestValidOptionalURL(t *testing.T) {
	valid := []string{
		"",
		"https://example.com/art",
		"http://example.com",
		"  https://example.com/path?x=1  ",
	}
	for _, value := range valid {
		if !ValidOptionalURL(value) {
			t.Errorf("ValidOptionalURL(%q) = false, want true", value)
		}
	}

	invalid := []string{
		"javascript:alert(1)",
		"JavaScript:alert(1)",
		"data:text/html,<script>alert(1)</script>",
		"file:///etc/passwd",
		"//example.com",
		"/relative/path",
		"https://",
		"not a url",
	}
	for _, value := range invalid {
		if ValidOptionalURL(value) {
			t.Errorf("ValidOptionalURL(%q) = true, want false", value)
		}
	}
}

func TestValidProfileImageAcceptsOwnUploads(t *testing.T) {
	key := "/media/" + strings.Repeat("a1", 24) + ".png"

	valid := []string{"", "https://example.com/me.jpg", key, "/media/" + strings.Repeat("0f", 24) + ".webp"}
	for _, value := range valid {
		if !ValidProfileImage(value) {
			t.Errorf("ValidProfileImage(%q) = false, want true", value)
		}
	}

	invalid := []string{
		"/media/abc.png", // key too short
		"/media/" + strings.Repeat("a1", 24) + ".svg", // extension not served
		"/media/" + strings.Repeat("A1", 24) + ".png", // not lowercase hex
		"/media/../secret.png",                        // traversal
		"media/" + strings.Repeat("a1", 24) + ".png",  // not rooted
		"javascript:alert(1)",
	}
	for _, value := range invalid {
		if ValidProfileImage(value) {
			t.Errorf("ValidProfileImage(%q) = true, want false", value)
		}
	}
}
