package chat

import (
	"net/http"
	"testing"
)

func TestServerCreateJoinMemberVisibility(t *testing.T) {
	d, alice := testDeps(t)
	bob := insertUser(t, d, "bob")
	carol := insertUser(t, d, "carol")
	h := testRouter(d)
	cookieA := authCookie(t, d, alice)
	cookieB := authCookie(t, d, bob)
	cookieC := authCookie(t, d, carol)

	server := createServer(t, h, cookieA, "Anime Club")
	if server.Owner.ID != alice || server.Owner.Username != "alice" {
		t.Fatalf("unexpected owner: %+v", server.Owner)
	}
	if server.Role == nil || *server.Role != "owner" {
		t.Fatalf("creator should be owner, got %v", server.Role)
	}
	if !slugPattern.MatchString(server.Slug) {
		t.Fatalf("invalid derived slug %q", server.Slug)
	}
	if server.MemberCount != 1 {
		t.Fatalf("member_count = %d, want 1", server.MemberCount)
	}
	if len(server.Channels) != 1 || server.Channels[0].Name != "general" || server.Channels[0].Type != "text" {
		t.Fatalf("default channel missing: %+v", server.Channels)
	}

	rec := request(t, h, http.MethodPost, "/api/v1/chat/servers", map[string]any{
		"name": "Other Club", "slug": server.Slug,
	}, cookieB)
	if rec.Code != http.StatusConflict || errorCode(t, rec) != codeSlugTaken {
		t.Fatalf("duplicate slug: status %d body %q", rec.Code, rec.Body.String())
	}

	rec = request(t, h, http.MethodGet, "/api/v1/chat/servers", nil, cookieC)
	if rec.Code != http.StatusOK {
		t.Fatalf("list servers: status %d body %q", rec.Code, rec.Body.String())
	}
	servers := decodeBody[[]ChatServer](t, rec)
	visible := false
	for _, item := range servers {
		if item.ID == server.ID {
			visible = true
			if item.Role != nil {
				t.Fatalf("non-member role = %q, want null", *item.Role)
			}
		}
	}
	if !visible {
		t.Fatal("public server not visible to a non-member")
	}

	channelsPath := "/api/v1/chat/servers/" + itoa(server.ID) + "/channels"
	rec = request(t, h, http.MethodGet, channelsPath, nil, cookieC)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("non-member channels: status %d body %q", rec.Code, rec.Body.String())
	}
	rec = request(t, h, http.MethodGet, channelsPath, nil, cookieB)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("another non-member channels: status %d", rec.Code)
	}

	rec = request(t, h, http.MethodPost, "/api/v1/chat/servers/"+itoa(server.ID)+"/join", nil, cookieC)
	if rec.Code != http.StatusOK {
		t.Fatalf("join: status %d body %q", rec.Code, rec.Body.String())
	}
	joined := decodeBody[ChatServer](t, rec)
	if joined.MemberCount != 2 || joined.Role == nil || *joined.Role != "member" {
		t.Fatalf("join result: %+v", joined)
	}

	rec = request(t, h, http.MethodGet, channelsPath, nil, cookieC)
	if rec.Code != http.StatusOK {
		t.Fatalf("member channels: status %d body %q", rec.Code, rec.Body.String())
	}
	channels := decodeBody[[]ChatChannel](t, rec)
	if len(channels) != 1 {
		t.Fatalf("member sees %d channels, want 1", len(channels))
	}

	rec = request(t, h, http.MethodPost, channelsPath, map[string]any{"name": "events"}, cookieC)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("member create channel: status %d body %q", rec.Code, rec.Body.String())
	}
	rec = request(t, h, http.MethodPost, channelsPath, map[string]any{
		"name": "events", "description": "Event planning",
	}, cookieA)
	if rec.Code != http.StatusCreated {
		t.Fatalf("owner create channel: status %d body %q", rec.Code, rec.Body.String())
	}
	created := decodeBody[ChatChannel](t, rec)
	if created.Type != "text" || created.ServerID != server.ID {
		t.Fatalf("created channel: %+v", created)
	}
	rec = request(t, h, http.MethodGet, channelsPath, nil, cookieC)
	if channels := decodeBody[[]ChatChannel](t, rec); len(channels) != 2 {
		t.Fatalf("member sees %d channels after creation, want 2", len(channels))
	}

	// Repeat joins are idempotent.
	rec = request(t, h, http.MethodPost, "/api/v1/chat/servers/"+itoa(server.ID)+"/join", nil, cookieC)
	if rec.Code != http.StatusOK {
		t.Fatalf("repeat join: status %d", rec.Code)
	}
	var members int
	if err := d.DB.QueryRow(`SELECT COUNT(*) FROM chat_server_members WHERE server_id = ?`, server.ID).Scan(&members); err != nil || members != 2 {
		t.Fatalf("member rows = %d, err = %v", members, err)
	}
}

func TestServerValidationAndMissingServer(t *testing.T) {
	d, alice := testDeps(t)
	h := testRouter(d)
	cookie := authCookie(t, d, alice)

	rec := request(t, h, http.MethodPost, "/api/v1/chat/servers", map[string]any{
		"name": "ab", "slug": "BAD SLUG",
	}, cookie)
	if rec.Code != http.StatusBadRequest || errorCode(t, rec) != codeValidation {
		t.Fatalf("invalid server: status %d body %q", rec.Code, rec.Body.String())
	}
	rec = request(t, h, http.MethodPost, "/api/v1/chat/servers/999/join", nil, cookie)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("join missing server: status %d body %q", rec.Code, rec.Body.String())
	}
	rec = request(t, h, http.MethodGet, "/api/v1/chat/servers/999/channels", nil, cookie)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing server channels: status %d", rec.Code)
	}
	rec = request(t, h, http.MethodGet, "/api/v1/chat/servers", nil, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated list servers: status %d", rec.Code)
	}
}
