package notifications

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Aswanidev-vs/promiseland/backend/internal/feed/shape/testutil"
	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
	"github.com/go-chi/chi/v5"
)

type notificationPage struct {
	Items      []Notification `json:"items"`
	NextCursor string         `json:"next_cursor"`
}

func notificationRouter(d *shared.Deps) chi.Router {
	r := chi.NewRouter()
	Register(r, d)
	return r
}

func request(t *testing.T, r chi.Router, method, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.AddCookie(&http.Cookie{Name: shared.SessionCookieName, Value: token})
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func makeNotification(t *testing.T, d *shared.Deps, userID int64, actorID *int64, notificationType string) int64 {
	t.Helper()
	res, err := d.DB.Exec(`
		INSERT INTO notifications (user_id, type, actor_id, target_type, target_id, data)
		VALUES (?, ?, ?, 'post', 1, '{"key":"value"}')`, userID, notificationType, actorID)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := res.LastInsertId()
	return id
}

func TestNotificationsOwnershipAndRead(t *testing.T) {
	d := testutil.Deps(t)
	aliceID, aliceToken := testutil.User(t, d, "alice")
	bobID, bobToken := testutil.User(t, d, "bob")
	aliceNotification := makeNotification(t, d, aliceID, &bobID, "follow")
	bobNotification := makeNotification(t, d, bobID, &aliceID, "reply")

	r := notificationRouter(d)
	if rec := request(t, r, http.MethodGet, "/notifications", "", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous status = %d", rec.Code)
	}

	rec := request(t, r, http.MethodGet, "/notifications", aliceToken, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d body=%s", rec.Code, rec.Body.String())
	}
	var page notificationPage
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != aliceNotification {
		t.Fatalf("alice notifications = %+v", page.Items)
	}
	item := page.Items[0]
	if item.Actor == nil || item.Actor.ID != bobID || string(item.Data) != `{"key":"value"}` || item.ReadAt != nil {
		t.Fatalf("notification shape = %+v", item)
	}

	// Another user cannot mark Alice's notification as read.
	if rec := request(t, r, http.MethodPost, "/notifications/"+itoa(aliceNotification)+"/read", bobToken, ""); rec.Code != http.StatusNotFound {
		t.Fatalf("cross-user mark read status = %d", rec.Code)
	}
	if rec := request(t, r, http.MethodPost, "/notifications/"+itoa(aliceNotification)+"/read", aliceToken, ""); rec.Code != http.StatusNoContent {
		t.Fatalf("mark read status = %d", rec.Code)
	}
	var readAt string
	if err := d.DB.QueryRow(`SELECT read_at FROM notifications WHERE id = ?`, aliceNotification).Scan(&readAt); err != nil || readAt == "" {
		t.Fatalf("read_at not set: %q %v", readAt, err)
	}

	makeNotification(t, d, aliceID, nil, "mention")
	if rec := request(t, r, http.MethodPost, "/notifications/read-all", aliceToken, ""); rec.Code != http.StatusNoContent {
		t.Fatalf("read-all POST status = %d", rec.Code)
	}
	if rec := request(t, r, http.MethodPatch, "/notifications/read-all", aliceToken, ""); rec.Code != http.StatusNoContent {
		t.Fatalf("read-all PATCH status = %d", rec.Code)
	}
	var unread int
	if err := d.DB.QueryRow(`SELECT COUNT(*) FROM notifications WHERE user_id = ? AND read_at IS NULL`, aliceID).Scan(&unread); err != nil || unread != 0 {
		t.Fatalf("unread = %d, %v", unread, err)
	}
	// Bob's notification is untouched.
	var bobUnread int
	if err := d.DB.QueryRow(`SELECT COUNT(*) FROM notifications WHERE id = ? AND read_at IS NULL`, bobNotification).Scan(&bobUnread); err != nil || bobUnread != 1 {
		t.Fatalf("bob unread = %d, %v", bobUnread, err)
	}
}

func TestNotificationPreferences(t *testing.T) {
	d := testutil.Deps(t)
	_, token := testutil.User(t, d, "alice")
	r := notificationRouter(d)

	rec := request(t, r, http.MethodGet, "/notifications/preferences", token, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("get preferences status = %d", rec.Code)
	}
	var prefs map[string]bool
	if err := json.Unmarshal(rec.Body.Bytes(), &prefs); err != nil {
		t.Fatal(err)
	}
	for _, category := range Categories {
		if !prefs[category] {
			t.Errorf("default %s should be enabled", category)
		}
	}

	rec = request(t, r, http.MethodPut, "/notifications/preferences", token, `{"follow":false,"reaction":false}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("put preferences status = %d body=%s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &prefs); err != nil {
		t.Fatal(err)
	}
	if prefs["follow"] || prefs["reaction"] || !prefs["reply"] {
		t.Fatalf("updated preferences = %+v", prefs)
	}

	rec = request(t, r, http.MethodGet, "/notifications/preferences", token, "")
	if err := json.Unmarshal(rec.Body.Bytes(), &prefs); err != nil {
		t.Fatal(err)
	}
	if prefs["follow"] || prefs["reaction"] || !prefs["mention"] {
		t.Fatalf("persisted preferences = %+v", prefs)
	}

	if rec := request(t, r, http.MethodPut, "/notifications/preferences", token, `{"unknown":true}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("unknown category status = %d", rec.Code)
	}
}

func itoa(value int64) string {
	if value == 0 {
		return "0"
	}
	negative := value < 0
	if negative {
		value = -value
	}
	var buf [20]byte
	i := len(buf)
	for value > 0 {
		i--
		buf[i] = byte('0' + value%10)
		value /= 10
	}
	if negative {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
