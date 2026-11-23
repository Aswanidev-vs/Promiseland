package chat

import (
	"net/http"
	"testing"
	"time"
)

func TestTypingRateLimit(t *testing.T) {
	d, alice := testDeps(t)
	bob := insertUser(t, d, "bob")
	h := testRouter(d)
	cookieA := authCookie(t, d, alice)
	cookieB := authCookie(t, d, bob)
	advance := withClock(t)
	conv := createDirect(t, h, cookieA, bob)
	path := "/api/v1/chat/typing"

	rec := request(t, h, http.MethodPost, path,
		map[string]any{"conversationId": conv.ID, "isTyping": true}, cookieA)
	if rec.Code != http.StatusOK {
		t.Fatalf("typing start: status %d body %q", rec.Code, rec.Body.String())
	}
	var count int
	_ = d.DB.QueryRow(`SELECT COUNT(*) FROM typing_state WHERE user_id = ? AND conversation_id = ?`, alice, conv.ID).Scan(&count)
	if count != 1 {
		t.Fatalf("typing rows = %d, want 1", count)
	}

	rec = request(t, h, http.MethodPost, path,
		map[string]any{"conversationId": conv.ID, "isTyping": true}, cookieA)
	if rec.Code != http.StatusTooManyRequests || errorCode(t, rec) != codeRateLimited {
		t.Fatalf("typing burst: status %d body %q", rec.Code, rec.Body.String())
	}

	advance(2100 * time.Millisecond)
	rec = request(t, h, http.MethodPost, path,
		map[string]any{"conversationId": conv.ID, "isTyping": false}, cookieA)
	if rec.Code != http.StatusOK {
		t.Fatalf("typing stop: status %d body %q", rec.Code, rec.Body.String())
	}
	_ = d.DB.QueryRow(`SELECT COUNT(*) FROM typing_state WHERE user_id = ? AND conversation_id = ?`, alice, conv.ID).Scan(&count)
	if count != 0 {
		t.Fatalf("typing rows after stop = %d, want 0", count)
	}

	// The limiter is per user and scope, so another member is not blocked.
	rec = request(t, h, http.MethodPost, path,
		map[string]any{"conversationId": conv.ID, "isTyping": true}, cookieB)
	if rec.Code != http.StatusOK {
		t.Fatalf("other user typing: status %d", rec.Code)
	}
	rec = request(t, h, http.MethodPost, path,
		map[string]any{"channelId": 1, "conversationId": conv.ID, "isTyping": true}, cookieA)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("ambiguous scope: status %d body %q", rec.Code, rec.Body.String())
	}
	rec = request(t, h, http.MethodPost, path,
		map[string]any{"conversationId": 999999, "isTyping": true}, cookieA)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing scope: status %d body %q", rec.Code, rec.Body.String())
	}
}

func TestPresenceSnapshot(t *testing.T) {
	d, alice := testDeps(t)
	bob := insertUser(t, d, "bob")
	h := testRouter(d)
	cookieA := authCookie(t, d, alice)
	cookieB := authCookie(t, d, bob)

	path := "/api/v1/chat/presence?ids=" + itoa(alice) + "," + itoa(bob)
	rec := request(t, h, http.MethodGet, path, nil, cookieA)
	if rec.Code != http.StatusOK {
		t.Fatalf("presence: status %d body %q", rec.Code, rec.Body.String())
	}
	statuses := decodeBody[map[int64]string](t, rec)
	if statuses[alice] != "online" || statuses[bob] != "offline" {
		t.Fatalf("initial presence: %+v", statuses)
	}

	// Any authenticated chat request acts as a heartbeat.
	rec = request(t, h, http.MethodGet, "/api/v1/chat/servers", nil, cookieB)
	if rec.Code != http.StatusOK {
		t.Fatalf("heartbeat request: status %d", rec.Code)
	}
	rec = request(t, h, http.MethodGet, path, nil, cookieA)
	statuses = decodeBody[map[int64]string](t, rec)
	if statuses[bob] != "online" {
		t.Fatalf("presence after heartbeat: %+v", statuses)
	}
}

func TestPublishWithoutSubscribers(t *testing.T) {
	d, alice := testDeps(t)
	m := newModule(d)
	// The realtime hub has no subscribers in unit tests; publishing must be a
	// safe no-op for every chat event type.
	m.publish("message.new", map[string]any{"id": 1}, 10, 0, nil)
	m.publish("reaction.updated", map[string]any{"id": 1}, 0, 20, []int64{alice})
	m.publish("message.deleted", map[string]any{"id": 1}, 10, 0, nil)
	m.publish("typing", map[string]any{"user_id": alice}, 0, 20, []int64{alice})
}
