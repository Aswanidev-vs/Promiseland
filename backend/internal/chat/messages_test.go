package chat

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
)

func TestMessagePaginationAndReplies(t *testing.T) {
	d, alice := testDeps(t)
	bob := insertUser(t, d, "bob")
	h := testRouter(d)
	cookieA := authCookie(t, d, alice)
	advance := withClock(t)

	server := createServer(t, h, cookieA, "History Club")
	channelID := firstChannelID(t, d, server.ID)
	ids := make([]int64, 0, 5)
	for i := 0; i < 5; i++ {
		msg := sendChannelMessage(t, h, cookieA, channelID, fmt.Sprintf("message %d", i))
		ids = append(ids, msg.ID)
		advance(1100 * time.Millisecond)
	}

	path := "/api/v1/chat/channels/" + itoa(channelID) + "/messages?limit=2"
	rec := request(t, h, http.MethodGet, path, nil, cookieA)
	if rec.Code != http.StatusOK {
		t.Fatalf("first page: status %d body %q", rec.Code, rec.Body.String())
	}
	page := decodeBody[shared.Page[ChatMessage]](t, rec)
	if len(page.Items) != 2 || page.Items[0].ID != ids[4] || page.Items[1].ID != ids[3] {
		t.Fatalf("first page items: %+v (want %d,%d)", page.Items, ids[4], ids[3])
	}
	if page.NextCursor == "" {
		t.Fatal("first page has no next_cursor")
	}

	rec = request(t, h, http.MethodGet, path+"&cursor="+page.NextCursor, nil, cookieA)
	page2 := decodeBody[shared.Page[ChatMessage]](t, rec)
	if rec.Code != http.StatusOK || len(page2.Items) != 2 || page2.Items[0].ID != ids[2] || page2.Items[1].ID != ids[1] {
		t.Fatalf("second page: status %d items %+v", rec.Code, page2.Items)
	}
	rec = request(t, h, http.MethodGet, path+"&cursor="+page2.NextCursor, nil, cookieA)
	page3 := decodeBody[shared.Page[ChatMessage]](t, rec)
	if rec.Code != http.StatusOK || len(page3.Items) != 1 || page3.Items[0].ID != ids[0] || page3.NextCursor != "" {
		t.Fatalf("last page: status %d items %+v cursor %q", rec.Code, page3.Items, page3.NextCursor)
	}

	// Reading history advances the read cursor, so unread_count is zero.
	rec = request(t, h, http.MethodGet, "/api/v1/chat/servers", nil, cookieA)
	servers := decodeBody[[]ChatServer](t, rec)
	if len(servers) != 1 || len(servers[0].Channels) != 1 || servers[0].Channels[0].UnreadCount != 0 {
		t.Fatalf("unread after read: %+v", servers)
	}

	sendPath := "/api/v1/chat/channels/" + itoa(channelID) + "/messages"
	rec = request(t, h, http.MethodPost, sendPath,
		map[string]any{"body": "reply one", "parentId": ids[0]}, cookieA)
	if rec.Code != http.StatusCreated {
		t.Fatalf("reply 1: status %d body %q", rec.Code, rec.Body.String())
	}
	reply1 := decodeBody[ChatMessage](t, rec)
	if reply1.ParentID == nil || *reply1.ParentID != ids[0] {
		t.Fatalf("reply parent: %+v", reply1.ParentID)
	}
	advance(1100 * time.Millisecond)
	rec = request(t, h, http.MethodPost, sendPath,
		map[string]any{"body": "reply two", "parentId": reply1.ID}, cookieA)
	if rec.Code != http.StatusCreated {
		t.Fatalf("reply 2: status %d body %q", rec.Code, rec.Body.String())
	}
	reply2 := decodeBody[ChatMessage](t, rec)
	advance(1100 * time.Millisecond)
	rec = request(t, h, http.MethodPost, sendPath,
		map[string]any{"body": "too deep", "parentId": reply2.ID}, cookieA)
	if rec.Code != http.StatusBadRequest || errorCode(t, rec) != codeValidation {
		t.Fatalf("depth-3 reply: status %d body %q", rec.Code, rec.Body.String())
	}

	// A parent from another scope is not a valid reply target. The rejected
	// depth-3 attempt above still consumed a rate-limit slot, so advance first.
	advance(1100 * time.Millisecond)
	conv := createDirect(t, h, cookieA, bob)
	foreign := sendConversationMessage(t, h, cookieA, conv.ID, "foreign parent")
	advance(1100 * time.Millisecond)
	rec = request(t, h, http.MethodPost, sendPath,
		map[string]any{"body": "bad parent", "parentId": foreign.ID}, cookieA)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("cross-scope reply: status %d body %q", rec.Code, rec.Body.String())
	}

	// Body validation happens before rate limiting.
	rec = request(t, h, http.MethodPost, sendPath, map[string]any{"body": "   "}, cookieA)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("empty body: status %d", rec.Code)
	}
	rec = request(t, h, http.MethodPost, sendPath, map[string]any{"body": strings.Repeat("x", 4001)}, cookieA)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("overlong body: status %d", rec.Code)
	}
}

func TestReactionToggle(t *testing.T) {
	d, alice := testDeps(t)
	bob := insertUser(t, d, "bob")
	h := testRouter(d)
	cookieA := authCookie(t, d, alice)
	cookieB := authCookie(t, d, bob)
	conv := createDirect(t, h, cookieA, bob)
	message := sendConversationMessage(t, h, cookieA, conv.ID, "react to me")

	reactPath := "/api/v1/chat/messages/" + itoa(message.ID) + "/reactions"
	rec := request(t, h, http.MethodPost, reactPath, map[string]any{"reaction": "heart"}, cookieA)
	if rec.Code != http.StatusOK {
		t.Fatalf("react: status %d body %q", rec.Code, rec.Body.String())
	}
	updated := decodeBody[ChatMessage](t, rec)
	if len(updated.Reactions) != 1 || updated.Reactions[0].Reaction != "heart" ||
		updated.Reactions[0].Count != 1 || !updated.Reactions[0].Me {
		t.Fatalf("reaction state: %+v", updated.Reactions)
	}

	rec = request(t, h, http.MethodPost, reactPath, map[string]any{"reaction": "heart"}, cookieA)
	updated = decodeBody[ChatMessage](t, rec)
	if rec.Code != http.StatusOK || len(updated.Reactions) != 0 {
		t.Fatalf("toggle off: status %d reactions %+v", rec.Code, updated.Reactions)
	}

	rec = request(t, h, http.MethodPost, reactPath, map[string]any{"reaction": "HEART"}, cookieB)
	updated = decodeBody[ChatMessage](t, rec)
	if rec.Code != http.StatusOK || len(updated.Reactions) != 1 ||
		updated.Reactions[0].Count != 1 || !updated.Reactions[0].Me {
		t.Fatalf("other user's reaction: status %d reactions %+v", rec.Code, updated.Reactions)
	}
	// The author sees the same reaction with me = false.
	rec = request(t, h, http.MethodGet,
		"/api/v1/chat/conversations/"+itoa(conv.ID)+"/messages", nil, cookieA)
	viewed := decodeBody[shared.Page[ChatMessage]](t, rec).Items
	if len(viewed) == 0 || len(viewed[0].Reactions) != 1 || viewed[0].Reactions[0].Me {
		t.Fatalf("author's reaction view: %+v", viewed)
	}
	rec = request(t, h, http.MethodPost, reactPath, map[string]any{"reaction": "star"}, cookieA)
	updated = decodeBody[ChatMessage](t, rec)
	if len(updated.Reactions) != 2 {
		t.Fatalf("two reaction codes: %+v", updated.Reactions)
	}
	for _, reaction := range updated.Reactions {
		if reaction.Reaction == "star" && !reaction.Me {
			t.Fatalf("own star reaction not marked me: %+v", reaction)
		}
	}

	rec = request(t, h, http.MethodPost, reactPath, map[string]any{"reaction": "fire"}, cookieA)
	if rec.Code != http.StatusBadRequest || errorCode(t, rec) != codeValidation {
		t.Fatalf("invalid reaction code: status %d body %q", rec.Code, rec.Body.String())
	}
}

func TestSoftDeleteAuthorization(t *testing.T) {
	d, alice := testDeps(t)
	bob := insertUser(t, d, "bob")
	carol := insertUser(t, d, "carol")
	h := testRouter(d)
	cookieA := authCookie(t, d, alice)
	cookieB := authCookie(t, d, bob)
	cookieC := authCookie(t, d, carol)
	advance := withClock(t)

	server := createServer(t, h, cookieA, "Moderation Club")
	channelID := firstChannelID(t, d, server.ID)
	for _, cookie := range []*http.Cookie{cookieB, cookieC} {
		rec := request(t, h, http.MethodPost, "/api/v1/chat/servers/"+itoa(server.ID)+"/join", nil, cookie)
		if rec.Code != http.StatusOK {
			t.Fatalf("join: status %d", rec.Code)
		}
	}
	first := sendChannelMessage(t, h, cookieA, channelID, "first")
	advance(1100 * time.Millisecond)

	deletePath := "/api/v1/chat/messages/"
	rec := request(t, h, http.MethodDelete, deletePath+itoa(first.ID), nil, cookieB)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("plain member delete: status %d body %q", rec.Code, rec.Body.String())
	}
	if _, err := d.DB.Exec(`UPDATE chat_server_members SET role = 'moderator' WHERE server_id = ? AND user_id = ?`,
		server.ID, bob); err != nil {
		t.Fatal(err)
	}
	second := sendChannelMessage(t, h, cookieA, channelID, "second")
	advance(1100 * time.Millisecond)
	rec = request(t, h, http.MethodDelete, deletePath+itoa(second.ID), nil, cookieB)
	if rec.Code != http.StatusOK {
		t.Fatalf("server moderator delete: status %d body %q", rec.Code, rec.Body.String())
	}

	if _, err := d.DB.Exec(`INSERT INTO platform_moderators(user_id) VALUES(?)`, carol); err != nil {
		t.Fatal(err)
	}
	third := sendChannelMessage(t, h, cookieA, channelID, "third")
	advance(1100 * time.Millisecond)
	rec = request(t, h, http.MethodDelete, deletePath+itoa(third.ID), nil, cookieC)
	if rec.Code != http.StatusOK {
		t.Fatalf("platform moderator delete: status %d body %q", rec.Code, rec.Body.String())
	}
	fourth := sendChannelMessage(t, h, cookieB, channelID, "mine")
	advance(1100 * time.Millisecond)
	rec = request(t, h, http.MethodDelete, deletePath+itoa(fourth.ID), nil, cookieB)
	if rec.Code != http.StatusOK {
		t.Fatalf("author delete: status %d body %q", rec.Code, rec.Body.String())
	}

	rec = request(t, h, http.MethodGet,
		"/api/v1/chat/channels/"+itoa(channelID)+"/messages?limit=20", nil, cookieA)
	items := decodeBody[shared.Page[ChatMessage]](t, rec).Items
	states := map[int64]bool{}
	for _, item := range items {
		states[item.ID] = item.Deleted
	}
	if states[first.ID] || !states[second.ID] || !states[third.ID] || !states[fourth.ID] {
		t.Fatalf("deleted flags: %+v", states)
	}
}

func TestReportDedup(t *testing.T) {
	d, alice := testDeps(t)
	bob := insertUser(t, d, "bob")
	carol := insertUser(t, d, "carol")
	h := testRouter(d)
	cookieA := authCookie(t, d, alice)
	cookieB := authCookie(t, d, bob)
	cookieC := authCookie(t, d, carol)
	conv := createDirect(t, h, cookieA, bob)
	message := sendConversationMessage(t, h, cookieB, conv.ID, "report me")

	path := "/api/v1/chat/messages/" + itoa(message.ID) + "/report"
	rec := request(t, h, http.MethodPost, path, map[string]any{"reason": "spam"}, cookieA)
	if rec.Code != http.StatusCreated {
		t.Fatalf("report: status %d body %q", rec.Code, rec.Body.String())
	}
	report := decodeBody[Report](t, rec)
	if report.TargetType != "message" || report.TargetID != message.ID || report.Status != "open" {
		t.Fatalf("report shape: %+v", report)
	}
	rec = request(t, h, http.MethodPost, path, map[string]any{"reason": "harassment"}, cookieA)
	if rec.Code != http.StatusConflict || errorCode(t, rec) != codeAlreadyReport {
		t.Fatalf("duplicate report: status %d body %q", rec.Code, rec.Body.String())
	}
	rec = request(t, h, http.MethodPost, path, map[string]any{"reason": "not-a-reason"}, cookieA)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid reason: status %d", rec.Code)
	}
	rec = request(t, h, http.MethodPost, path, map[string]any{"reason": "spam"}, cookieC)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("non-member report: status %d body %q", rec.Code, rec.Body.String())
	}
	var count int
	_ = d.DB.QueryRow(`SELECT COUNT(*) FROM reports WHERE target_type = 'message'`).Scan(&count)
	if count != 1 {
		t.Fatalf("report rows = %d, want 1", count)
	}
}

func TestMessageRateLimits(t *testing.T) {
	d, alice := testDeps(t)
	bob := insertUser(t, d, "bob")
	h := testRouter(d)
	cookieA := authCookie(t, d, alice)
	advance := withClock(t)
	conv := createDirect(t, h, cookieA, bob)
	path := "/api/v1/chat/conversations/" + itoa(conv.ID) + "/messages"

	rec := request(t, h, http.MethodPost, path, map[string]any{"body": "first"}, cookieA)
	if rec.Code != http.StatusCreated {
		t.Fatalf("first message: status %d", rec.Code)
	}
	rec = request(t, h, http.MethodPost, path, map[string]any{"body": "instant replay"}, cookieA)
	if rec.Code != http.StatusTooManyRequests || errorCode(t, rec) != codeRateLimited {
		t.Fatalf("burst limit: status %d body %q", rec.Code, rec.Body.String())
	}
	advance(1100 * time.Millisecond)
	rec = request(t, h, http.MethodPost, path, map[string]any{"body": "second"}, cookieA)
	if rec.Code != http.StatusCreated {
		t.Fatalf("after burst window: status %d", rec.Code)
	}
	for i := 0; i < 18; i++ {
		advance(1100 * time.Millisecond)
		rec = request(t, h, http.MethodPost, path, map[string]any{"body": fmt.Sprintf("filler %d", i)}, cookieA)
		if rec.Code != http.StatusCreated {
			t.Fatalf("message %d: status %d body %q", i, rec.Code, rec.Body.String())
		}
	}
	advance(1100 * time.Millisecond)
	rec = request(t, h, http.MethodPost, path, map[string]any{"body": "over minute cap"}, cookieA)
	if rec.Code != http.StatusTooManyRequests || errorCode(t, rec) != codeRateLimited {
		t.Fatalf("minute cap: status %d body %q", rec.Code, rec.Body.String())
	}
}
