package discovery

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
)

func TestDiscoveryCursorPagination(t *testing.T) {
	env := newTestEnv(t)
	viewer := seedUser(t, env, "viewer", "public", true)
	token := seedSession(t, env, viewer)

	anime := make([]int64, 3)
	for i := range anime {
		anime[i] = addAnime(t, env, fmt.Sprintf("Cursor Anime %d", i+1))
		addAnimeInterest(t, env, viewer, anime[i])
	}
	candidates := make([]int64, 5)
	for i := range candidates {
		candidates[i] = seedUser(t, env, fmt.Sprintf("cand%d", i+1), "public", true)
	}
	for i := 0; i < 3; i++ {
		addAnimeInterest(t, env, candidates[0], anime[i])
	}
	for i := 0; i < 2; i++ {
		addAnimeInterest(t, env, candidates[1], anime[i])
		addAnimeInterest(t, env, candidates[2], anime[i])
	}
	addAnimeInterest(t, env, candidates[3], anime[0])

	full := listFriends(t, env, token, "")
	wantOrder := candidateIDs(full.Items)
	if len(wantOrder) != 5 {
		t.Fatalf("full page = %v", candidateIDs(full.Items))
	}

	var all []int64
	cursor := ""
	for pageIndex := 0; pageIndex < 10; pageIndex++ {
		query := "?limit=2"
		if cursor != "" {
			query += "&cursor=" + cursor
		}
		result := listFriends(t, env, token, query)
		if len(result.Items) > 2 {
			t.Fatalf("page %d size = %d", pageIndex, len(result.Items))
		}
		for _, item := range result.Items {
			all = append(all, item.User.ID)
		}
		if result.NextCursor == "" {
			break
		}
		raw, err := shared.DecodeCursor(result.NextCursor)
		if err != nil || len(strings.Split(raw, "|")) != 2 {
			t.Fatalf("cursor format = %q (%v)", raw, err)
		}
		cursor = result.NextCursor
	}
	if fmt.Sprint(all) != fmt.Sprint(wantOrder) {
		t.Fatalf("paginated order = %v, want %v", all, wantOrder)
	}
	if len(all) != len(wantOrder) {
		t.Fatalf("pagination duplicated or dropped candidates: %v", all)
	}

	status, body := env.do(t, token, http.MethodGet,
		"/api/v1/discovery/friends?cursor=not-a-cursor", nil)
	if status != http.StatusBadRequest || parseError(t, body).Error.Code != codeValidation {
		t.Errorf("invalid cursor = %d (%s)", status, body)
	}
}

func notificationCount(t *testing.T, env *testEnv, userID int64, typ string) int {
	t.Helper()
	var count int
	if err := env.d.DB.QueryRow(`
		SELECT COUNT(*) FROM notifications WHERE user_id = ? AND type = ?`, userID, typ).Scan(&count); err != nil {
		t.Fatalf("count notifications: %v", err)
	}
	return count
}

func TestConnectionsDedupReviewNotificationsAndFilters(t *testing.T) {
	env := newTestEnv(t)
	alice := seedUser(t, env, "alice", "public", true)
	bob := seedUser(t, env, "bob", "public", true)
	carol := seedUser(t, env, "carol", "public", true)
	private := seedUser(t, env, "private", "private", true)
	hidden := seedUser(t, env, "hidden", "public", false)
	blocked := seedUser(t, env, "blocked", "public", true)
	aliceToken := seedSession(t, env, alice)
	bobToken := seedSession(t, env, bob)
	carolToken := seedSession(t, env, carol)
	env.exec(t, `INSERT INTO blocks (blocker_id, blocked_id) VALUES (?, ?)`, alice, blocked)

	// The frontend's snake_case body is accepted.
	status, body := env.do(t, aliceToken, http.MethodPost, "/api/v1/connections",
		map[string]any{"user_id": bob, "message": "hello bob"})
	if status != http.StatusCreated {
		t.Fatalf("create = %d (%s)", status, body)
	}
	connection := decodeJSON[Connection](t, body)
	if connection.Status != "pending" || connection.Message != "hello bob" ||
		connection.Requester.ID != alice || connection.Addressee.ID != bob || connection.ID <= 0 {
		t.Fatalf("connection shape = %+v", connection)
	}
	if notificationCount(t, env, bob, "connection_request") != 1 {
		t.Error("recipient did not receive a connection_request notification")
	}

	// Duplicate pairs are rejected in both directions and both key spellings.
	status, body = env.do(t, aliceToken, http.MethodPost, "/api/v1/connections",
		map[string]any{"userId": bob, "message": "again"})
	if status != http.StatusConflict || parseError(t, body).Error.Code != codeConnection {
		t.Errorf("duplicate = %d (%s)", status, body)
	}
	status, body = env.do(t, bobToken, http.MethodPost, "/api/v1/connections",
		map[string]any{"userId": alice, "message": "reverse"})
	if status != http.StatusConflict || parseError(t, body).Error.Code != codeConnection {
		t.Errorf("reverse duplicate = %d (%s)", status, body)
	}
	// camelCase wins when both spellings are present.
	status, body = env.do(t, bobToken, http.MethodPost, "/api/v1/connections",
		map[string]any{"user_id": alice, "userId": carol, "message": "camel wins"})
	if status != http.StatusCreated {
		t.Fatalf("alias precedence create = %d (%s)", status, body)
	}
	if decodeJSON[Connection](t, body).Addressee.ID != carol {
		t.Error("camelCase userId did not win over user_id")
	}

	// Invalid targets.
	for _, tc := range []struct {
		name   string
		token  string
		target int64
		status int
	}{
		{"self", aliceToken, alice, http.StatusBadRequest},
		{"private", aliceToken, private, http.StatusForbidden},
		{"not discoverable", aliceToken, hidden, http.StatusForbidden},
		{"blocked", aliceToken, blocked, http.StatusForbidden},
		{"missing", aliceToken, 99999, http.StatusNotFound},
	} {
		status, body := env.do(t, tc.token, http.MethodPost, "/api/v1/connections",
			map[string]any{"userId": tc.target})
		if status != tc.status {
			t.Errorf("%s create = %d (%s), want %d", tc.name, status, body, tc.status)
		}
	}

	// Only the addressee may review a pending request.
	for _, tc := range []struct {
		token string
		want  int
	}{{aliceToken, http.StatusForbidden}, {carolToken, http.StatusForbidden}} {
		status, body := env.do(t, tc.token, http.MethodPatch,
			fmt.Sprintf("/api/v1/connections/%d", connection.ID),
			map[string]any{"status": "accepted"})
		if status != tc.want {
			t.Errorf("review by %s = %d (%s)", tc.token, status, body)
		}
	}
	status, body = env.do(t, bobToken, http.MethodPatch,
		fmt.Sprintf("/api/v1/connections/%d", connection.ID),
		map[string]any{"status": "accepted"})
	if status != http.StatusOK {
		t.Fatalf("accept = %d (%s)", status, body)
	}
	if decoded := decodeJSON[Connection](t, body); decoded.Status != "accepted" {
		t.Errorf("accepted status = %+v", decoded)
	}
	if notificationCount(t, env, alice, "connection_accepted") != 1 {
		t.Error("accepting did not notify the requester")
	}
	// A second review is a conflict.
	status, body = env.do(t, bobToken, http.MethodPatch,
		fmt.Sprintf("/api/v1/connections/%d", connection.ID),
		map[string]any{"status": "ignored"})
	if status != http.StatusConflict {
		t.Errorf("second review = %d (%s)", status, body)
	}
	// Invalid statuses are field errors.
	status, body = env.do(t, bobToken, http.MethodPatch,
		fmt.Sprintf("/api/v1/connections/%d", connection.ID),
		map[string]any{"status": "maybe"})
	if status != http.StatusBadRequest || parseError(t, body).Error.Code != codeValidation {
		t.Errorf("invalid review status = %d (%s)", status, body)
	}

	// Filters and pagination.
	for _, tc := range []struct {
		name, query, token string
		want               int
	}{
		{"alice all", "", aliceToken, 1},
		{"alice incoming", "?direction=incoming", aliceToken, 0},
		{"alice outgoing", "?direction=outgoing", aliceToken, 1},
		{"bob incoming", "?direction=incoming", bobToken, 1},
		{"bob accepted", "?status=accepted", bobToken, 1},
		{"bob pending", "?status=pending", bobToken, 1},
		{"carol all", "", carolToken, 1},
	} {
		status, body := env.do(t, tc.token, http.MethodGet, "/api/v1/connections"+tc.query, nil)
		if status != http.StatusOK {
			t.Fatalf("%s list = %d (%s)", tc.name, status, body)
		}
		result := decodeJSON[page[Connection]](t, body)
		if len(result.Items) != tc.want {
			t.Errorf("%s got %d items, want %d", tc.name, len(result.Items), tc.want)
		}
	}
	for _, query := range []string{"?status=nope", "?direction=sideways", "?cursor=bad"} {
		status, body := env.do(t, aliceToken, http.MethodGet, "/api/v1/connections"+query, nil)
		if status != http.StatusBadRequest || parseError(t, body).Error.Code != codeValidation {
			t.Errorf("filter %s = %d (%s)", query, status, body)
		}
	}

	// Either party may delete; outsiders may not.
	if status, body := env.do(t, carolToken, http.MethodDelete,
		fmt.Sprintf("/api/v1/connections/%d", connection.ID), nil); status != http.StatusForbidden {
		t.Errorf("outsider delete = %d (%s)", status, body)
	}
	if status, body := env.do(t, aliceToken, http.MethodDelete,
		fmt.Sprintf("/api/v1/connections/%d", connection.ID), nil); status != http.StatusOK {
		t.Errorf("delete = %d (%s)", status, body)
	}
	status, body = env.do(t, aliceToken, http.MethodGet, "/api/v1/connections?status=accepted", nil)
	if result := decodeJSON[page[Connection]](t, body); status != http.StatusOK || len(result.Items) != 0 {
		t.Errorf("after delete = %d (%s)", status, body)
	}
}
