package teamups

import (
	"encoding/json"
	"strings"
	"unicode/utf8"
)

// cleanCategories validates and normalises a client-supplied category list.
func cleanCategories(values []string) ([]string, map[string]string) {
	details := map[string]string{}
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = normalize(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	if len(out) < 1 || len(out) > 5 {
		details["categories"] = "Choose between 1 and 5 categories."
		return out, details
	}
	for _, value := range out {
		if !allowedCategories[value] {
			details["categories"] = "One or more categories are not supported."
			break
		}
	}
	return out, details
}

func encodeCategories(categories []string) string {
	raw, err := json.Marshal(categories)
	if err != nil {
		return "[]"
	}
	return string(raw)
}

func decodeCategories(raw string) []string {
	out := []string{}
	var values []string
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return out
	}
	seen := map[string]bool{}
	for _, value := range values {
		value = normalize(value)
		if value != "" && allowedCategories[value] && !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	return out
}

func validText(value string, min, max int) bool {
	n := utf8.RuneCountInString(strings.TrimSpace(value))
	return n >= min && n <= max
}
