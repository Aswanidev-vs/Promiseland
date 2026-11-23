package events

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
)

func listEvents(t *testing.T, h http.Handler, path string, cookie *http.Cookie) page[Event] {
	t.Helper()
	rec := request(t, h, http.MethodGet, path, nil, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s: status %d body %q", path, rec.Code, rec.Body.String())
	}
	return decodeBody[page[Event]](t, rec)
}

func TestListFiltersAndPagination(t *testing.T) {
	d, alice := testDeps(t)
	h := testRouter(d)
	cookie := authCookie(t, d, alice)

	e1 := createEvent(t, h, cookie, sampleEvent(
		"Cosplay Meetup", "2027-01-05T10:00:00Z", "2027-01-05T12:00:00Z", "Austin", "meetup", false))
	e2 := createEvent(t, h, cookie, sampleEvent(
		"Anime Convention", "2027-02-10T09:00:00Z", "2027-02-10T17:00:00Z", "Osaka", "convention", false))
	third := sampleEvent("Screening Night", "2027-03-01T19:00:00Z", "2027-03-01T22:00:00Z", "Tokyo", "screening", true)
	third["country"] = "USA"
	e3 := createEvent(t, h, cookie, third)

	if got := listEvents(t, h, "/api/v1/events?q=cosplay", nil); len(got.Items) != 1 || got.Items[0].ID != e1.ID {
		t.Fatalf("q filter: got %+v", got.Items)
	}
	if got := listEvents(t, h, "/api/v1/events?type=convention", nil); len(got.Items) != 1 || got.Items[0].ID != e2.ID {
		t.Fatalf("type filter: got %+v", got.Items)
	}
	if got := listEvents(t, h, "/api/v1/events?from=2027-02-01&to=2027-02-28", nil); len(got.Items) != 1 || got.Items[0].ID != e2.ID {
		t.Fatalf("date filters: got %+v", got.Items)
	}
	if got := listEvents(t, h, "/api/v1/events?city=Osaka", nil); len(got.Items) != 1 || got.Items[0].ID != e2.ID {
		t.Fatalf("city filter: got %+v", got.Items)
	}
	if got := listEvents(t, h, "/api/v1/events?country=Japan", nil); len(got.Items) != 2 {
		t.Fatalf("country filter: got %+v", got.Items)
	}
	if got := listEvents(t, h, "/api/v1/events?online=true", nil); len(got.Items) != 1 || got.Items[0].ID != e3.ID {
		t.Fatalf("online filter: got %+v", got.Items)
	}
	if rec := request(t, h, http.MethodGet, "/api/v1/events?from=not-a-date", nil, nil); rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid date should return validation error: status %d", rec.Code)
	}

	// Cursor pagination is ordered by (starts_at, id); next_cursor must always
	// be present in the JSON, including on the last page.
	first := listEvents(t, h, "/api/v1/events?limit=1", nil)
	if len(first.Items) != 1 || first.Items[0].ID != e1.ID || first.NextCursor == "" {
		t.Fatalf("first page: %+v", first)
	}
	raw := decodeBody[map[string]json.RawMessage](t, request(t, h, http.MethodGet, "/api/v1/events?limit=1", nil, nil))
	if _, ok := raw["next_cursor"]; !ok {
		t.Fatal("next_cursor must always be present")
	}
	second := listEvents(t, h, "/api/v1/events?limit=1&cursor="+url.QueryEscape(first.NextCursor), nil)
	if len(second.Items) != 1 || second.Items[0].ID != e2.ID {
		t.Fatalf("second page: %+v", second)
	}
	thirdPage := listEvents(t, h, "/api/v1/events?limit=1&cursor="+url.QueryEscape(second.NextCursor), nil)
	if len(thirdPage.Items) != 1 || thirdPage.Items[0].ID != e3.ID || thirdPage.NextCursor != "" {
		t.Fatalf("last page: %+v", thirdPage)
	}
	lastRaw := decodeBody[map[string]json.RawMessage](t, request(t, h, http.MethodGet,
		"/api/v1/events?limit=1&cursor="+url.QueryEscape(second.NextCursor), nil, nil))
	if _, ok := lastRaw["next_cursor"]; !ok {
		t.Fatal("next_cursor must be present even when empty")
	}
}

func TestLikeWildcardEscaping(t *testing.T) {
	d, alice := testDeps(t)
	h := testRouter(d)
	cookie := authCookie(t, d, alice)
	createEvent(t, h, cookie, sampleEvent(
		"50% Off Meetup", "2027-01-05T10:00:00Z", "2027-01-05T12:00:00Z", "A_B", "meetup", false))

	if got := listEvents(t, h, "/api/v1/events?q="+url.QueryEscape("%"), nil); len(got.Items) != 1 {
		t.Fatalf("literal %% should match one event: %+v", got.Items)
	}
	if got := listEvents(t, h, "/api/v1/events?city="+url.QueryEscape("_"), nil); len(got.Items) != 1 {
		t.Fatalf("literal _ should match one event: %+v", got.Items)
	}
	if got := listEvents(t, h, "/api/v1/events?q=nomatch%25", nil); len(got.Items) != 0 {
		t.Fatalf("unexpected match: %+v", got.Items)
	}
}

func TestCreateValidationAndTimezoneFallback(t *testing.T) {
	d, alice := testDeps(t)
	h := testRouter(d)
	cookie := authCookie(t, d, alice)
	base := sampleEvent("Valid Event Title", "2027-01-05T10:00:00Z", "2027-01-05T12:00:00Z", "Osaka", "meetup", false)

	cases := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{"short title", func(b map[string]any) { b["title"] = "ab" }},
		{"long description", func(b map[string]any) { b["description"] = strings.Repeat("x", 5001) }},
		{"bad type", func(b map[string]any) { b["eventType"] = "concert" }},
		{"end before start", func(b map[string]any) { b["endsAt"] = "2027-01-05T09:00:00Z" }},
		{"bad start time", func(b map[string]any) { b["startsAt"] = "tomorrow" }},
		{"negative capacity", func(b map[string]any) { b["capacity"] = -1 }},
		{"long city", func(b map[string]any) { b["city"] = strings.Repeat("a", 81) }},
		{"long country", func(b map[string]any) { b["country"] = strings.Repeat("a", 81) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := map[string]any{}
			for k, v := range base {
				body[k] = v
			}
			tc.mutate(body)
			rec := request(t, h, http.MethodPost, "/api/v1/events", body, cookie)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status %d body %q", rec.Code, rec.Body.String())
			}
			errBody := decodeBody[apiErrorEnvelope](t, rec)
			if errBody.Error.Code != codeValidation || len(errBody.Error.Details) == 0 {
				t.Fatalf("unexpected error body: %+v", errBody)
			}
		})
	}

	body := map[string]any{}
	for k, v := range base {
		body[k] = v
	}
	body["timezone"] = "Mars/Olympus"
	created := createEvent(t, h, cookie, body)
	if created.Timezone != "UTC" {
		t.Fatalf("invalid IANA timezone should fall back to UTC: %q", created.Timezone)
	}
}

func TestOrganizerOnlyEditAndCancelledVisibility(t *testing.T) {
	d, alice := testDeps(t)
	h := testRouter(d)
	aliceCookie := authCookie(t, d, alice)
	bob := insertUser(t, d, "bob")
	bobCookie := authCookie(t, d, bob)

	event := createEvent(t, h, aliceCookie, sampleEvent(
		"Cosplay Meetup", "2027-01-05T10:00:00Z", "2027-01-05T12:00:00Z", "Austin", "meetup", false))
	path := "/api/v1/events/" + itoa(event.ID)

	if rec := request(t, h, http.MethodPatch, path, map[string]any{"title": "Hacked Title"}, bobCookie); rec.Code != http.StatusForbidden {
		t.Fatalf("non-organiser PATCH: status %d body %q", rec.Code, rec.Body.String())
	}
	if rec := request(t, h, http.MethodDelete, path, nil, bobCookie); rec.Code != http.StatusForbidden {
		t.Fatalf("non-organiser DELETE: status %d body %q", rec.Code, rec.Body.String())
	}
	rec := request(t, h, http.MethodPatch, path, map[string]any{"title": "Updated Meetup Title"}, aliceCookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("organiser PATCH: status %d body %q", rec.Code, rec.Body.String())
	}
	if got := decodeBody[Event](t, rec); got.Title != "Updated Meetup Title" {
		t.Fatalf("title not updated: %+v", got)
	}

	rec = request(t, h, http.MethodDelete, path, nil, aliceCookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("organiser DELETE: status %d body %q", rec.Code, rec.Body.String())
	}
	if got := decodeBody[Event](t, rec); got.Status != "cancelled" {
		t.Fatalf("DELETE must cancel rather than hard-delete: %+v", got)
	}

	if rec := request(t, h, http.MethodGet, path, nil, bobCookie); rec.Code != http.StatusNotFound {
		t.Fatalf("cancelled detail for non-owner: status %d body %q", rec.Code, rec.Body.String())
	}
	if rec := request(t, h, http.MethodGet, path, nil, aliceCookie); rec.Code != http.StatusOK {
		t.Fatalf("cancelled detail for owner: status %d body %q", rec.Code, rec.Body.String())
	}
	if got := listEvents(t, h, "/api/v1/events", bobCookie); len(got.Items) != 0 {
		t.Fatalf("cancelled event visible to non-owner: %+v", got.Items)
	}
	if got := listEvents(t, h, "/api/v1/events", aliceCookie); len(got.Items) != 1 {
		t.Fatalf("cancelled event hidden from owner: %+v", got.Items)
	}
	if rec := request(t, h, http.MethodGet, path+"/updates", nil, bobCookie); rec.Code != http.StatusNotFound {
		t.Fatalf("cancelled updates for non-owner: status %d", rec.Code)
	}
	if rec := request(t, h, http.MethodGet, path+"/comments", nil, bobCookie); rec.Code != http.StatusNotFound {
		t.Fatalf("cancelled comments for non-owner: status %d", rec.Code)
	}

	// History survives cancellation: the organiser can still post updates.
	if rec := request(t, h, http.MethodPost, path+"/updates", map[string]any{"body": "Cancelled, sorry."}, aliceCookie); rec.Code != http.StatusCreated {
		t.Fatalf("update on cancelled event: status %d body %q", rec.Code, rec.Body.String())
	}
}

func TestRSVPResponseShapesCapacityAndSwitching(t *testing.T) {
	d, alice := testDeps(t)
	h := testRouter(d)
	aliceCookie := authCookie(t, d, alice)
	bob := insertUser(t, d, "bob")
	bobCookie := authCookie(t, d, bob)

	created := sampleEvent("Small Meetup", "2027-01-05T10:00:00Z", "2027-01-05T12:00:00Z", "Osaka", "meetup", false)
	created["capacity"] = 1
	event := createEvent(t, h, aliceCookie, created)
	path := "/api/v1/events/" + itoa(event.ID) + "/rsvp"

	rsvp := func(cookie *http.Cookie, status string) *httptest.ResponseRecorder {
		return request(t, h, http.MethodPost, path, map[string]any{"status": status}, cookie)
	}
	rec := rsvp(aliceCookie, "going")
	if rec.Code != http.StatusOK {
		t.Fatalf("alice going: status %d body %q", rec.Code, rec.Body.String())
	}
	first := decodeBody[Event](t, rec)
	if first.RSVPCount != 1 || first.RSVPStatus == nil || *first.RSVPStatus != "going" {
		t.Fatalf("POST rsvp must return updated Event: %+v", first)
	}
	raw := decodeBody[map[string]json.RawMessage](t, rec)
	for _, key := range []string{"rsvp_count", "rsvp_status", "capacity", "status"} {
		if _, ok := raw[key]; !ok {
			t.Fatalf("POST rsvp response missing %q", key)
		}
	}

	rec = rsvp(bobCookie, "going")
	if rec.Code != http.StatusConflict {
		t.Fatalf("full event should reject going: status %d body %q", rec.Code, rec.Body.String())
	}
	if decodeBody[apiErrorEnvelope](t, rec).Error.Code != codeEventFull {
		t.Fatalf("unexpected error body: %s", rec.Body.String())
	}
	rec = rsvp(bobCookie, "maybe")
	if got := decodeBody[Event](t, rec); got.RSVPCount != 1 || got.RSVPStatus == nil || *got.RSVPStatus != "maybe" {
		t.Fatalf("maybe must not consume capacity: %+v", got)
	}
	rec = rsvp(aliceCookie, "maybe")
	if got := decodeBody[Event](t, rec); got.RSVPCount != 0 {
		t.Fatalf("going -> maybe must free a slot: %+v", got)
	}
	rec = rsvp(bobCookie, "going")
	if got := decodeBody[Event](t, rec); got.RSVPCount != 1 {
		t.Fatalf("freed slot must be usable: %+v", got)
	}
	rec = rsvp(aliceCookie, "going")
	if rec.Code != http.StatusConflict {
		t.Fatalf("capacity must be enforced again: status %d", rec.Code)
	}

	// DELETE returns the updated Event too, with a null rsvp_status.
	rec = request(t, h, http.MethodDelete, path, nil, bobCookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("DELETE rsvp: status %d body %q", rec.Code, rec.Body.String())
	}
	after := decodeBody[Event](t, rec)
	if after.RSVPCount != 0 || after.RSVPStatus != nil {
		t.Fatalf("DELETE rsvp must return refreshed Event: %+v", after)
	}
	afterRaw := decodeBody[map[string]json.RawMessage](t, rec)
	if value, ok := afterRaw["rsvp_status"]; !ok || string(value) != "null" {
		t.Fatalf("rsvp_status must be present and null, got %s", afterRaw["rsvp_status"])
	}

	if rec := rsvp(aliceCookie, "maybe-later"); rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid status: status %d", rec.Code)
	}
	if rec := rsvp(nil, "going"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated rsvp: status %d", rec.Code)
	}

	cancelled := createEvent(t, h, aliceCookie, sampleEvent(
		"Another Meetup", "2027-02-05T10:00:00Z", "2027-02-05T12:00:00Z", "Osaka", "meetup", false))
	if rec := request(t, h, http.MethodDelete, "/api/v1/events/"+itoa(cancelled.ID), nil, aliceCookie); rec.Code != http.StatusOK {
		t.Fatalf("cancel event: status %d", rec.Code)
	}
	if rec := request(t, h, http.MethodPost,
		"/api/v1/events/"+itoa(cancelled.ID)+"/rsvp", map[string]any{"status": "going"}, bobCookie); rec.Code != http.StatusConflict {
		t.Fatalf("rsvp on cancelled event: status %d body %q", rec.Code, rec.Body.String())
	}
}

func TestUpdateAndCommentResponseShapes(t *testing.T) {
	d, alice := testDeps(t)
	h := testRouter(d)
	aliceCookie := authCookie(t, d, alice)
	bob := insertUser(t, d, "bob")
	bobCookie := authCookie(t, d, bob)
	event := createEvent(t, h, aliceCookie, sampleEvent(
		"Anime Meetup", "2027-01-05T10:00:00Z", "2027-01-05T12:00:00Z", "Austin", "meetup", false))
	base := "/api/v1/events/" + itoa(event.ID)

	if rec := request(t, h, http.MethodPost, base+"/updates", map[string]any{"body": "Not allowed"}, bobCookie); rec.Code != http.StatusForbidden {
		t.Fatalf("non-organiser update: status %d body %q", rec.Code, rec.Body.String())
	}
	if rec := request(t, h, http.MethodPost, base+"/updates",
		map[string]any{"body": strings.Repeat("x", 2001)}, aliceCookie); rec.Code != http.StatusBadRequest {
		t.Fatalf("long update: status %d", rec.Code)
	}
	rec := request(t, h, http.MethodPost, base+"/updates", map[string]any{"body": "Venue changed!"}, aliceCookie)
	if rec.Code != http.StatusCreated {
		t.Fatalf("add update: status %d body %q", rec.Code, rec.Body.String())
	}
	update := decodeBody[EventUpdate](t, rec)
	if update.ID <= 0 || update.EventID != event.ID || update.Author.ID != alice || update.Body != "Venue changed!" || update.CreatedAt == "" {
		t.Fatalf("addEventUpdate must return EventUpdate: %+v", update)
	}
	updateRaw := decodeBody[map[string]json.RawMessage](t, rec)
	for _, key := range []string{"id", "event_id", "author", "body", "created_at"} {
		if _, ok := updateRaw[key]; !ok {
			t.Fatalf("update response missing %q", key)
		}
	}
	if updates := decodeBody[[]EventUpdate](t, request(t, h, http.MethodGet, base+"/updates", nil, nil)); len(updates) != 1 {
		t.Fatalf("list updates: %+v", updates)
	}

	if rec := request(t, h, http.MethodPost, base+"/comments", map[string]any{"body": "Anyone going?"}, nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated comment: status %d", rec.Code)
	}
	rec = request(t, h, http.MethodPost, base+"/comments", map[string]any{"body": "Anyone going?"}, bobCookie)
	if rec.Code != http.StatusCreated {
		t.Fatalf("add comment: status %d body %q", rec.Code, rec.Body.String())
	}
	root := decodeBody[EventComment](t, rec)
	if root.ID <= 0 || root.EventID != event.ID || root.Author.ID != bob || root.ParentID != nil || root.Deleted || root.Body != "Anyone going?" {
		t.Fatalf("addEventComment must return EventComment: %+v", root)
	}
	commentRaw := decodeBody[map[string]json.RawMessage](t, rec)
	for _, key := range []string{"id", "event_id", "author", "parent_id", "body", "created_at", "deleted"} {
		if _, ok := commentRaw[key]; !ok {
			t.Fatalf("comment response missing %q", key)
		}
	}

	child := decodeBody[EventComment](t, request(t, h, http.MethodPost, base+"/comments",
		map[string]any{"body": "Yes!", "parentId": root.ID}, aliceCookie))
	if child.ParentID == nil || *child.ParentID != root.ID {
		t.Fatalf("reply parent: %+v", child)
	}
	grandchild := decodeBody[EventComment](t, request(t, h, http.MethodPost, base+"/comments",
		map[string]any{"body": "Me too", "parent_id": child.ID}, bobCookie))
	if grandchild.ParentID == nil || *grandchild.ParentID != child.ID {
		t.Fatalf("nested reply parent: %+v", grandchild)
	}
	if rec := request(t, h, http.MethodPost, base+"/comments",
		map[string]any{"body": "Too deep", "parentId": grandchild.ID}, aliceCookie); rec.Code != http.StatusBadRequest {
		t.Fatalf("depth limit: status %d body %q", rec.Code, rec.Body.String())
	}

	firstPage := decodeBody[page[EventComment]](t, request(t, h, http.MethodGet, base+"/comments?limit=1", nil, nil))
	if len(firstPage.Items) != 1 || firstPage.NextCursor == "" {
		t.Fatalf("comments first page: %+v", firstPage)
	}
	secondPage := decodeBody[page[EventComment]](t, request(t, h, http.MethodGet,
		base+"/comments?limit=1&cursor="+url.QueryEscape(firstPage.NextCursor), nil, nil))
	if len(secondPage.Items) != 1 {
		t.Fatalf("comments second page: %+v", secondPage)
	}

	if _, err := d.DB.Exec(`UPDATE event_comments SET deleted_at = datetime('now') WHERE id = ?`, root.ID); err != nil {
		t.Fatal(err)
	}
	withDeleted := decodeBody[page[EventComment]](t, request(t, h, http.MethodGet, base+"/comments", nil, nil))
	found := false
	for _, c := range withDeleted.Items {
		if c.ID == root.ID {
			found = true
			if !c.Deleted || c.Body != "" {
				t.Fatalf("deleted comment should hide its body: %+v", c)
			}
		}
	}
	if !found {
		t.Fatal("deleted comment placeholder is missing")
	}
}

func TestUnauthorizedAndErrorEnvelope(t *testing.T) {
	d, alice := testDeps(t)
	h := testRouter(d)
	cookie := authCookie(t, d, alice)

	if rec := request(t, h, http.MethodGet, "/api/v1/events", nil, nil); rec.Code != http.StatusOK {
		t.Fatalf("public list: status %d", rec.Code)
	}
	if rec := request(t, h, http.MethodPost, "/api/v1/events",
		sampleEvent("Some Event", "2027-01-05T10:00:00Z", "2027-01-05T12:00:00Z", "Osaka", "meetup", false), nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated create: status %d body %q", rec.Code, rec.Body.String())
	}
	event := createEvent(t, h, cookie, sampleEvent(
		"Some Event", "2027-01-05T10:00:00Z", "2027-01-05T12:00:00Z", "Osaka", "meetup", false))
	path := "/api/v1/events/" + itoa(event.ID)
	if rec := request(t, h, http.MethodPatch, path, map[string]any{"title": "New Title"}, nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated patch: status %d", rec.Code)
	}
	if rec := request(t, h, http.MethodGet, "/api/v1/events/999999", nil, nil); rec.Code != http.StatusNotFound {
		t.Fatalf("missing event: status %d", rec.Code)
	}
	rec := request(t, h, http.MethodPost, "/api/v1/events", "{broken", cookie)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("malformed JSON: status %d", rec.Code)
	}
	if got := decodeBody[apiErrorEnvelope](t, rec); got.Error.Code != shared.CodeInvalidJSON {
		t.Fatalf("unexpected error body: %+v", got)
	}
}
