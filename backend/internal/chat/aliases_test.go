package chat

import (
	"net/http"
	"testing"
	"time"
)

// F1 uses snake_case request keys while the conventions document camelCase.
// Both spellings must be accepted; response shapes remain exactly as
// specified in the R3 contract.
func TestFrontendSnakeCaseInputAliases(t *testing.T) {
	d, alice := testDeps(t)
	bob := insertUser(t, d, "bob")
	h := testRouter(d)
	cookieA := authCookie(t, d, alice)
	cookieB := authCookie(t, d, bob)
	advance := withClock(t)

	rec := request(t, h, http.MethodPost, "/api/v1/chat/conversations",
		map[string]any{"member_ids": []int64{bob}}, cookieA)
	if rec.Code != http.StatusCreated {
		t.Fatalf("conversation with member_ids: status %d body %q", rec.Code, rec.Body.String())
	}
	conv := decodeBody[Conversation](t, rec)
	if conv.Type != "direct" || len(conv.Members) != 2 {
		t.Fatalf("conversation shape: %+v", conv)
	}

	root := sendConversationMessage(t, h, cookieA, conv.ID, "root")
	advance(1100 * time.Millisecond)
	rec = request(t, h, http.MethodPost,
		"/api/v1/chat/conversations/"+itoa(conv.ID)+"/messages",
		map[string]any{"body": "reply", "parent_id": root.ID}, cookieA)
	if rec.Code != http.StatusCreated {
		t.Fatalf("message with parent_id: status %d body %q", rec.Code, rec.Body.String())
	}
	reply := decodeBody[ChatMessage](t, rec)
	if reply.ParentID == nil || *reply.ParentID != root.ID {
		t.Fatalf("parent_id not honoured: %+v", reply.ParentID)
	}

	reactionPath := "/api/v1/chat/messages/" + itoa(root.ID) + "/reactions"
	rec = request(t, h, http.MethodPost, reactionPath, map[string]any{"reaction": "heart"}, cookieB)
	if rec.Code != http.StatusOK {
		t.Fatalf("reaction: status %d body %q", rec.Code, rec.Body.String())
	}
	reacted := decodeBody[ChatMessage](t, rec)
	if len(reacted.Reactions) != 1 || reacted.Reactions[0].Reaction != "heart" {
		t.Fatalf("reaction shape: %+v", reacted.Reactions)
	}

	rec = request(t, h, http.MethodPost, "/api/v1/chat/typing",
		map[string]any{"conversation_id": conv.ID, "is_typing": true}, cookieB)
	if rec.Code != http.StatusOK {
		t.Fatalf("typing with snake keys: status %d body %q", rec.Code, rec.Body.String())
	}
	var count int
	_ = d.DB.QueryRow(`SELECT COUNT(*) FROM typing_state WHERE user_id = ? AND conversation_id = ?`, bob, conv.ID).Scan(&count)
	if count != 1 {
		t.Fatalf("typing rows = %d, want 1", count)
	}
	advance(2100 * time.Millisecond)
	rec = request(t, h, http.MethodPost, "/api/v1/chat/typing",
		map[string]any{"conversation_id": conv.ID, "is_typing": false}, cookieB)
	if rec.Code != http.StatusOK {
		t.Fatalf("typing stop: status %d body %q", rec.Code, rec.Body.String())
	}
	_ = d.DB.QueryRow(`SELECT COUNT(*) FROM typing_state WHERE user_id = ? AND conversation_id = ?`, bob, conv.ID).Scan(&count)
	if count != 0 {
		t.Fatalf("typing rows after stop = %d, want 0", count)
	}

	rec = request(t, h, http.MethodGet,
		"/api/v1/chat/presence?ids="+itoa(alice)+","+itoa(bob), nil, cookieA)
	if rec.Code != http.StatusOK {
		t.Fatalf("presence: status %d body %q", rec.Code, rec.Body.String())
	}
	statuses := decodeBody[map[int64]string](t, rec)
	if len(statuses) != 2 || statuses[alice] == "" || statuses[bob] == "" {
		t.Fatalf("presence snapshot: %+v", statuses)
	}
}

func TestCamelCaseInputsStillAccepted(t *testing.T) {
	d, alice := testDeps(t)
	bob := insertUser(t, d, "bob")
	h := testRouter(d)
	cookieA := authCookie(t, d, alice)
	cookieB := authCookie(t, d, bob)
	advance := withClock(t)

	rec := request(t, h, http.MethodPost, "/api/v1/chat/conversations",
		map[string]any{"memberIds": []int64{bob}}, cookieA)
	if rec.Code != http.StatusCreated {
		t.Fatalf("conversation with memberIds: status %d body %q", rec.Code, rec.Body.String())
	}
	conv := decodeBody[Conversation](t, rec)
	root := sendConversationMessage(t, h, cookieA, conv.ID, "root")
	advance(1100 * time.Millisecond)
	rec = request(t, h, http.MethodPost,
		"/api/v1/chat/conversations/"+itoa(conv.ID)+"/messages",
		map[string]any{"body": "reply", "parentId": root.ID}, cookieA)
	if rec.Code != http.StatusCreated {
		t.Fatalf("message with parentId: status %d body %q", rec.Code, rec.Body.String())
	}
	rec = request(t, h, http.MethodPost, "/api/v1/chat/typing",
		map[string]any{"conversationId": conv.ID, "isTyping": true}, cookieB)
	if rec.Code != http.StatusOK {
		t.Fatalf("typing with camel keys: status %d body %q", rec.Code, rec.Body.String())
	}

	// When both spellings are supplied the camelCase value wins.
	rec = request(t, h, http.MethodPost, "/api/v1/chat/conversations",
		map[string]any{"memberIds": []int64{bob}, "member_ids": []int64{999999}}, cookieA)
	if rec.Code != http.StatusOK {
		t.Fatalf("mixed spelling conversation: status %d body %q", rec.Code, rec.Body.String())
	}
}
