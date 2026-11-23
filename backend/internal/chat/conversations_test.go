package chat

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestDirectConversationUniquenessAndBlocks(t *testing.T) {
	d, alice := testDeps(t)
	bob := insertUser(t, d, "bob")
	h := testRouter(d)
	cookieA := authCookie(t, d, alice)
	cookieB := authCookie(t, d, bob)
	advance := withClock(t)

	conv := createDirect(t, h, cookieA, bob)
	if conv.Type != "direct" || len(conv.Members) != 2 {
		t.Fatalf("direct conversation: %+v", conv)
	}
	var pairKeys, total int
	_ = d.DB.QueryRow(`SELECT COUNT(*) FROM conversations WHERE pair_key IS NOT NULL`).Scan(&pairKeys)
	_ = d.DB.QueryRow(`SELECT COUNT(*) FROM conversations`).Scan(&total)
	if pairKeys != 1 || total != 1 {
		t.Fatalf("pair_key rows = %d, total = %d", pairKeys, total)
	}

	// Reopening the same pair returns the existing thread with 200.
	rec := request(t, h, http.MethodPost, "/api/v1/chat/conversations",
		map[string]any{"memberIds": []int64{alice}}, cookieB)
	if rec.Code != http.StatusOK {
		t.Fatalf("reopen direct: status %d body %q", rec.Code, rec.Body.String())
	}
	reopened := decodeBody[Conversation](t, rec)
	if reopened.ID != conv.ID {
		t.Fatalf("reopened id %d, want %d", reopened.ID, conv.ID)
	}
	_ = d.DB.QueryRow(`SELECT COUNT(*) FROM conversations`).Scan(&total)
	if total != 1 {
		t.Fatalf("duplicate direct conversation created: %d rows", total)
	}

	_ = sendConversationMessage(t, h, cookieA, conv.ID, "hello bob")
	advance(1100 * time.Millisecond)

	// A block in either direction prevents DM creation and reading.
	if _, err := d.DB.Exec(`
		INSERT INTO blocks(blocker_id, blocked_id, created_at) VALUES(?,?,datetime('now'))`, bob, alice); err != nil {
		t.Fatal(err)
	}
	rec = request(t, h, http.MethodPost, "/api/v1/chat/conversations",
		map[string]any{"memberIds": []int64{bob}}, cookieA)
	if rec.Code != http.StatusForbidden || errorCode(t, rec) != codeBlocked {
		t.Fatalf("blocked direct create: status %d body %q", rec.Code, rec.Body.String())
	}
	rec = request(t, h, http.MethodGet,
		"/api/v1/chat/conversations/"+itoa(conv.ID)+"/messages", nil, cookieA)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("blocked read: status %d body %q", rec.Code, rec.Body.String())
	}
	// Blocking hides the thread for both sides until the block is removed.
	rec = request(t, h, http.MethodGet,
		"/api/v1/chat/conversations/"+itoa(conv.ID)+"/messages", nil, cookieB)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("blocker read: status %d body %q", rec.Code, rec.Body.String())
	}
	rec = request(t, h, http.MethodGet, "/api/v1/chat/conversations", nil, cookieA)
	if rec.Code != http.StatusOK {
		t.Fatalf("list conversations: status %d", rec.Code)
	}
	if items := decodeBody[[]Conversation](t, rec); len(items) != 0 {
		t.Fatalf("blocked conversation still listed: %d items", len(items))
	}

	if _, err := d.DB.Exec(`DELETE FROM blocks WHERE blocker_id = ? AND blocked_id = ?`, bob, alice); err != nil {
		t.Fatal(err)
	}
	rec = request(t, h, http.MethodGet,
		"/api/v1/chat/conversations/"+itoa(conv.ID)+"/messages", nil, cookieA)
	if rec.Code != http.StatusOK {
		t.Fatalf("read after unblock: status %d body %q", rec.Code, rec.Body.String())
	}
	rec = request(t, h, http.MethodGet, "/api/v1/chat/conversations", nil, cookieA)
	items := decodeBody[[]Conversation](t, rec)
	if len(items) != 1 {
		t.Fatalf("conversations after unblock: %d items", len(items))
	}
	last := items[0].LastMessage
	if last == nil || last.Body != "hello bob" {
		t.Fatalf("last_message: %+v", last)
	}
}

func TestGroupConversationLimits(t *testing.T) {
	d, alice := testDeps(t)
	bob := insertUser(t, d, "bob")
	carol := insertUser(t, d, "carol")
	h := testRouter(d)
	cookieA := authCookie(t, d, alice)

	rec := request(t, h, http.MethodPost, "/api/v1/chat/conversations", map[string]any{
		"memberIds": []int64{bob, carol}, "name": "Cosplay Plans",
	}, cookieA)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create group: status %d body %q", rec.Code, rec.Body.String())
	}
	group := decodeBody[Conversation](t, rec)
	if group.Type != "group" || group.Name != "Cosplay Plans" || len(group.Members) != 3 {
		t.Fatalf("group conversation: %+v", group)
	}

	// Unknown users and self are rejected.
	rec = request(t, h, http.MethodPost, "/api/v1/chat/conversations",
		map[string]any{"memberIds": []int64{999999}}, cookieA)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unknown member: status %d body %q", rec.Code, rec.Body.String())
	}
	rec = request(t, h, http.MethodPost, "/api/v1/chat/conversations",
		map[string]any{"memberIds": []int64{alice, bob}}, cookieA)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("self member: status %d body %q", rec.Code, rec.Body.String())
	}

	// Total membership is capped at 20 (caller + 19 others).
	ids := make([]int64, 0, 20)
	for i := 0; i < 20; i++ {
		ids = append(ids, insertUser(t, d, fmt.Sprintf("extra%02d", i)))
	}
	rec = request(t, h, http.MethodPost, "/api/v1/chat/conversations",
		map[string]any{"memberIds": ids}, cookieA)
	if rec.Code != http.StatusBadRequest || errorCode(t, rec) != codeValidation {
		t.Fatalf("oversized group: status %d body %q", rec.Code, rec.Body.String())
	}

	// A blocked user cannot be added to a new group.
	if _, err := d.DB.Exec(`
		INSERT INTO blocks(blocker_id, blocked_id, created_at) VALUES(?,?,datetime('now'))`, bob, alice); err != nil {
		t.Fatal(err)
	}
	rec = request(t, h, http.MethodPost, "/api/v1/chat/conversations",
		map[string]any{"memberIds": []int64{bob, carol}}, cookieA)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("blocked group member: status %d body %q", rec.Code, rec.Body.String())
	}
}

func TestConversationNotVisibleToNonMember(t *testing.T) {
	d, alice := testDeps(t)
	bob := insertUser(t, d, "bob")
	carol := insertUser(t, d, "carol")
	h := testRouter(d)
	cookieA := authCookie(t, d, alice)
	cookieC := authCookie(t, d, carol)
	conv := createDirect(t, h, cookieA, bob)

	rec := request(t, h, http.MethodGet,
		"/api/v1/chat/conversations/"+itoa(conv.ID)+"/messages", nil, cookieC)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("non-member message read: status %d body %q", rec.Code, rec.Body.String())
	}
	rec = request(t, h, http.MethodPost,
		"/api/v1/chat/conversations/"+itoa(conv.ID)+"/messages",
		map[string]any{"body": "intruder"}, cookieC)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("non-member message send: status %d body %q", rec.Code, rec.Body.String())
	}
}
