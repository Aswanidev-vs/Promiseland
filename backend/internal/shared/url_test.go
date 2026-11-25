package shared

import "testing"

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
