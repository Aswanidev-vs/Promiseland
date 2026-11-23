package discovery

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
)

func TestDiscoveryEdgeCases(t *testing.T) {
	env := newTestEnv(t)
	viewer := seedUser(t, env, "viewer", "public", true)
	target := seedUser(t, env, "target", "public", true)
	token := seedSession(t, env, viewer)

	// Missing auth never reaches a handler.
	for _, path := range []string{
		"/api/v1/discovery/friends",
		"/api/v1/discovery/settings",
		"/api/v1/connections",
	} {
		if status, body := env.do(t, "", http.MethodGet, path, nil); status != http.StatusUnauthorized {
			t.Errorf("unauthenticated GET %s = %d (%s)", path, status, body)
		}
	}

	// Empty, malformed and overlong bodies fail with field details instead of
	// leaking database errors.
	for _, tc := range []struct {
		name string
		body any
	}{
		{"empty object", map[string]any{}},
		{"negative id", map[string]any{"user_id": -1}},
		{"wrong type", map[string]any{"userId": "target"}},
		{"overlong message", map[string]any{"userId": target, "message": strings.Repeat("x", maxMessageRunes+1)}},
	} {
		status, body := env.do(t, token, http.MethodPost, "/api/v1/connections", tc.body)
		if status != http.StatusBadRequest {
			t.Errorf("%s = %d (%s)", tc.name, status, body)
			continue
		}
		parsed := parseError(t, body)
		if parsed.Error.Code != codeValidation || len(parsed.Error.Details) == 0 {
			t.Errorf("%s details = %+v", tc.name, parsed)
		}
	}

	// Suppression writes are idempotent when retried.
	path := fmt.Sprintf("/api/v1/discovery/%d/suppress", target)
	for i := 0; i < 2; i++ {
		for _, method := range []string{http.MethodPost, http.MethodDelete} {
			status, body := env.do(t, token, method, path, nil)
			if status != http.StatusOK {
				t.Fatalf("%s suppression attempt %d = %d (%s)", method, i, status, body)
			}
		}
	}

	// Invalid ids and a cursor past the last page are handled explicitly.
	if status, body := env.do(t, token, http.MethodPost, "/api/v1/discovery/0/suppress", nil); status != http.StatusBadRequest {
		t.Errorf("zero user id = %d (%s)", status, body)
	}
	if status, body := env.do(t, token, http.MethodPatch, "/api/v1/connections/abc",
		map[string]any{"status": "accepted"}); status != http.StatusBadRequest {
		t.Errorf("invalid connection id = %d (%s)", status, body)
	}
	// In DESC count order, (0, maxID) is the key past the final zero-score item.
	pastEnd := listFriends(t, env, token, "?limit=1&cursor="+shared.EncodeCursor("0|999999"))
	if pastEnd.Items == nil || len(pastEnd.Items) != 0 || pastEnd.NextCursor != "" {
		t.Errorf("page past end = %+v", pastEnd)
	}
}
