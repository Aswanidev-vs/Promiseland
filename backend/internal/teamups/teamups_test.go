package teamups

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/Aswanidev-vs/promiseland/backend/internal/db"
	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
	"github.com/Aswanidev-vs/promiseland/backend/internal/testutil"
	"github.com/go-chi/chi/v5"
)

type testEnv struct {
	d *shared.Deps
	h http.Handler
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	database, _ := testutil.TempDB(t, "teamups.db")
	if err := db.Migrate(database, nil); err != nil {
		t.Fatal(err)
	}
	d := &shared.Deps{DB: database, Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	root := chi.NewRouter()
	root.Use(shared.OptionalAuth(d))
	root.Route("/api/v1", func(r chi.Router) {
		Register(r, d)
		RegisterProtected(r.With(shared.RequireAuth(d)), d)
	})
	return &testEnv{d: d, h: root}
}

func (e *testEnv) user(t *testing.T, username string) (int64, *http.Cookie) {
	t.Helper()
	res, err := e.d.DB.Exec(`
		INSERT INTO users(email, password_hash, display_name, username)
		VALUES (?, 'hash', ?, ?)`, username+"@example.test", username, username)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := res.LastInsertId()
	if _, err := e.d.DB.Exec(`INSERT INTO user_profiles(user_id, visibility) VALUES (?, 'public')`, id); err != nil {
		t.Fatal(err)
	}
	token := "teamups-" + username
	sum := sha256.Sum256([]byte(token))
	if _, err := e.d.DB.Exec(`
		INSERT INTO sessions(token_hash, user_id, expires_at)
		VALUES (?, ?, datetime('now', '+1 day'))`, hex.EncodeToString(sum[:]), id); err != nil {
		t.Fatal(err)
	}
	return id, &http.Cookie{Name: shared.SessionCookieName, Value: token}
}

func (e *testEnv) do(t *testing.T, method, path string, body any, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, reader)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, req)
	return rec
}

func errorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode error %q: %v", rec.Body.String(), err)
	}
	return payload.Error.Code
}

func itoa(id int64) string { return strconv.FormatInt(id, 10) }

func TestTeamUpCRUDAndStatusTransitions(t *testing.T) {
	e := newTestEnv(t)
	owner, ownerCookie := e.user(t, "owner")
	_, otherCookie := e.user(t, "other")

	created := e.do(t, http.MethodPost, "/api/v1/team-ups", map[string]any{
		"title":       "Looking for cosplayers",
		"description": "Convention group",
		"categories":  []string{"cosplay", "convention"},
	}, ownerCookie)
	if created.Code != http.StatusCreated {
		t.Fatalf("create = %d: %s", created.Code, created.Body.String())
	}
	var teamUp TeamUp
	if err := json.Unmarshal(created.Body.Bytes(), &teamUp); err != nil {
		t.Fatal(err)
	}
	if teamUp.ID <= 0 || teamUp.Owner.ID != owner || teamUp.Status != "open" ||
		len(teamUp.Categories) != 2 || teamUp.Event != nil || teamUp.MyApplication != nil {
		t.Fatalf("created team up = %+v", teamUp)
	}

	invalid := e.do(t, http.MethodPost, "/api/v1/team-ups", map[string]any{
		"title": "no", "description": "x", "categories": []string{"not-a-category"},
	}, ownerCookie)
	if invalid.Code != http.StatusBadRequest || errorCode(t, invalid) != codeValidation {
		t.Errorf("invalid create = %d %q", invalid.Code, errorCode(t, invalid))
	}
	missingEvent := e.do(t, http.MethodPost, "/api/v1/team-ups", map[string]any{
		"title": "Valid title", "categories": []string{"cosplay"}, "eventId": 999,
	}, ownerCookie)
	if missingEvent.Code != http.StatusBadRequest {
		t.Errorf("missing event = %d", missingEvent.Code)
	}

	path := "/api/v1/team-ups/" + itoa(teamUp.ID)
	forbiddenPatch := e.do(t, http.MethodPatch, path, map[string]any{"status": "closed"}, otherCookie)
	if forbiddenPatch.Code != http.StatusForbidden || errorCode(t, forbiddenPatch) != codeForbidden {
		t.Errorf("foreign patch = %d %q", forbiddenPatch.Code, errorCode(t, forbiddenPatch))
	}
	patched := e.do(t, http.MethodPatch, path, map[string]any{
		"title": "Updated title", "categories": []string{"photography"}, "status": "filled",
	}, ownerCookie)
	if patched.Code != http.StatusOK {
		t.Fatalf("patch = %d: %s", patched.Code, patched.Body.String())
	}
	var updated TeamUp
	if err := json.Unmarshal(patched.Body.Bytes(), &updated); err != nil {
		t.Fatal(err)
	}
	if updated.Title != "Updated title" || updated.Status != "filled" ||
		len(updated.Categories) != 1 || updated.Categories[0] != "photography" {
		t.Errorf("updated team up = %+v", updated)
	}
	closed := e.do(t, http.MethodPatch, path, map[string]any{"status": "closed"}, ownerCookie)
	if closed.Code != http.StatusOK {
		t.Errorf("close = %d", closed.Code)
	}
	var closedTeamUp TeamUp
	if err := json.Unmarshal(closed.Body.Bytes(), &closedTeamUp); err != nil {
		t.Fatal(err)
	}
	if closedTeamUp.Status != "closed" {
		t.Errorf("closed status = %q", closedTeamUp.Status)
	}

	forbiddenDelete := e.do(t, http.MethodDelete, path, nil, otherCookie)
	if forbiddenDelete.Code != http.StatusForbidden {
		t.Errorf("foreign delete = %d", forbiddenDelete.Code)
	}
	deleted := e.do(t, http.MethodDelete, path, nil, ownerCookie)
	if deleted.Code != http.StatusOK {
		t.Errorf("delete = %d", deleted.Code)
	}
	missing := e.do(t, http.MethodGet, path, nil, nil)
	if missing.Code != http.StatusNotFound {
		t.Errorf("get after delete = %d", missing.Code)
	}
}

func TestTeamUpEmbedsEventDefensively(t *testing.T) {
	e := newTestEnv(t)
	owner, ownerCookie := e.user(t, "owner")
	attendee, attendeeCookie := e.user(t, "attendee")

	res, err := e.d.DB.Exec(`
		INSERT INTO events(organizer_id, title, description, event_type, starts_at, ends_at, timezone, venue_text, city, country, is_online)
		VALUES (?, 'Cosplay Convention', 'Annual meetup', 'convention',
		        '2026-12-01T10:00:00Z', '2026-12-01T18:00:00Z', 'UTC', 'Hall A', 'Tokyo', 'JP', 0)`, owner)
	if err != nil {
		t.Fatal(err)
	}
	eventID, _ := res.LastInsertId()
	if _, err := e.d.DB.Exec(`INSERT INTO event_rsvps(event_id, user_id, status) VALUES (?, ?, 'going')`,
		eventID, attendee); err != nil {
		t.Fatal(err)
	}

	created := e.do(t, http.MethodPost, "/api/v1/team-ups", map[string]any{
		"title": "Convention crew", "categories": []string{"convention"}, "eventId": eventID,
	}, ownerCookie)
	if created.Code != http.StatusCreated {
		t.Fatalf("create = %d: %s", created.Code, created.Body.String())
	}
	var teamUp TeamUp
	if err := json.Unmarshal(created.Body.Bytes(), &teamUp); err != nil {
		t.Fatal(err)
	}
	if teamUp.Event == nil || teamUp.Event.ID != eventID || teamUp.Event.Title != "Cosplay Convention" ||
		teamUp.Event.City != "Tokyo" || teamUp.Event.RSVPCount != 1 || teamUp.Event.Organizer.ID != owner {
		t.Fatalf("embedded event = %+v", teamUp.Event)
	}

	detail := e.do(t, http.MethodGet, "/api/v1/team-ups/"+itoa(teamUp.ID), nil, attendeeCookie)
	var viewed TeamUp
	if err := json.Unmarshal(detail.Body.Bytes(), &viewed); err != nil {
		t.Fatal(err)
	}
	if viewed.Event == nil || viewed.Event.RSVPStatus == nil || *viewed.Event.RSVPStatus != "going" {
		t.Errorf("viewer rsvp status = %+v", viewed.Event)
	}
}

func TestTeamUpApplicationsDedupAndOwnerReview(t *testing.T) {
	e := newTestEnv(t)
	owner, ownerCookie := e.user(t, "owner")
	applicant, applicantCookie := e.user(t, "applicant")
	_, strangerCookie := e.user(t, "stranger")

	created := e.do(t, http.MethodPost, "/api/v1/team-ups", map[string]any{
		"title": "Photography team", "categories": []string{"photography"},
	}, ownerCookie)
	if created.Code != http.StatusCreated {
		t.Fatalf("create = %s", created.Body.String())
	}
	var teamUp TeamUp
	if err := json.Unmarshal(created.Body.Bytes(), &teamUp); err != nil {
		t.Fatal(err)
	}
	path := "/api/v1/team-ups/" + itoa(teamUp.ID)

	selfApply := e.do(t, http.MethodPost, path+"/apply", map[string]any{"message": "mine"}, ownerCookie)
	if selfApply.Code != http.StatusBadRequest || errorCode(t, selfApply) != codeSelfApply {
		t.Errorf("self apply = %d %q", selfApply.Code, errorCode(t, selfApply))
	}
	apply := e.do(t, http.MethodPost, path+"/apply", map[string]any{"message": "I can help"}, applicantCookie)
	if apply.Code != http.StatusCreated {
		t.Fatalf("apply = %d: %s", apply.Code, apply.Body.String())
	}
	var application TeamUpApplication
	if err := json.Unmarshal(apply.Body.Bytes(), &application); err != nil {
		t.Fatal(err)
	}
	if application.ID <= 0 || application.Applicant.ID != applicant || application.Status != "pending" ||
		application.TeamUpID != teamUp.ID || application.Message != "I can help" {
		t.Fatalf("application = %+v", application)
	}
	duplicate := e.do(t, http.MethodPost, path+"/apply", map[string]any{"message": "again"}, applicantCookie)
	if duplicate.Code != http.StatusConflict || errorCode(t, duplicate) != codeApplied {
		t.Errorf("duplicate apply = %d %q", duplicate.Code, errorCode(t, duplicate))
	}

	detail := e.do(t, http.MethodGet, path, nil, applicantCookie)
	var mine TeamUp
	if err := json.Unmarshal(detail.Body.Bytes(), &mine); err != nil {
		t.Fatal(err)
	}
	if mine.MyApplication == nil || *mine.MyApplication != "pending" || mine.ApplicantCount != 1 {
		t.Errorf("my_application = %v, count = %d", mine.MyApplication, mine.ApplicantCount)
	}

	forbiddenList := e.do(t, http.MethodGet, path+"/applications", nil, strangerCookie)
	if forbiddenList.Code != http.StatusForbidden {
		t.Errorf("foreign applications list = %d", forbiddenList.Code)
	}
	listed := e.do(t, http.MethodGet, path+"/applications", nil, ownerCookie)
	var applications []TeamUpApplication
	if err := json.Unmarshal(listed.Body.Bytes(), &applications); err != nil {
		t.Fatal(err)
	}
	if len(applications) != 1 || applications[0].ID != application.ID {
		t.Errorf("applications = %+v", applications)
	}

	reviewPath := path + "/applications/" + itoa(application.ID)
	forbiddenReview := e.do(t, http.MethodPatch, reviewPath, map[string]any{"status": "accepted"}, strangerCookie)
	if forbiddenReview.Code != http.StatusForbidden {
		t.Errorf("foreign review = %d", forbiddenReview.Code)
	}
	invalidReview := e.do(t, http.MethodPatch, reviewPath, map[string]any{"status": "pending"}, ownerCookie)
	if invalidReview.Code != http.StatusBadRequest {
		t.Errorf("invalid review = %d", invalidReview.Code)
	}
	reviewed := e.do(t, http.MethodPatch, reviewPath, map[string]any{"status": "accepted"}, ownerCookie)
	if reviewed.Code != http.StatusOK {
		t.Fatalf("review = %d: %s", reviewed.Code, reviewed.Body.String())
	}
	var accepted TeamUpApplication
	if err := json.Unmarshal(reviewed.Body.Bytes(), &accepted); err != nil {
		t.Fatal(err)
	}
	if accepted.Status != "accepted" {
		t.Errorf("review status = %q", accepted.Status)
	}
	detail = e.do(t, http.MethodGet, path, nil, applicantCookie)
	if err := json.Unmarshal(detail.Body.Bytes(), &mine); err != nil {
		t.Fatal(err)
	}
	if mine.MyApplication == nil || *mine.MyApplication != "accepted" {
		t.Errorf("my_application after review = %v", mine.MyApplication)
	}
	if _, err := e.d.DB.Exec(`SELECT 1 FROM notifications WHERE user_id = ? AND type = 'team_up_application'`, owner); err != nil {
		t.Errorf("optimistic owner notification missing: %v", err)
	}
}
