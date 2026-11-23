package moderation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/Aswanidev-vs/promiseland/backend/internal/feed/shape/testutil"
	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
	"github.com/go-chi/chi/v5"
)

func itoa(value int64) string { return strconv.FormatInt(value, 10) }

func moderationRouter(d *shared.Deps) chi.Router {
	r := chi.NewRouter()
	Register(r, d)
	return r
}

func modRequest(t *testing.T, r chi.Router, method, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
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

func grantModerator(t *testing.T, d *shared.Deps, userID int64) {
	t.Helper()
	testutil.Exec(t, d, `INSERT OR IGNORE INTO platform_moderators (user_id) VALUES (?)`, userID)
	testutil.Exec(t, d, `UPDATE users SET is_moderator = 1 WHERE id = ?`, userID)
}

func reportBody(targetType string, targetID int64, reason string) string {
	return fmt.Sprintf(`{"targetType":%q,"targetId":%d,"reason":%q,"details":"spam link"}`,
		targetType, targetID, reason)
}

func TestReportValidationAndDeduplication(t *testing.T) {
	d := testutil.Deps(t)
	reporterID, reporterToken := testutil.User(t, d, "reporter")
	_, _ = testutil.User(t, d, "author")
	r := moderationRouter(d)

	rec := modRequest(t, r, http.MethodPost, "/reports", reporterToken, reportBody("post", 42, "spam"))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d body=%s", rec.Code, rec.Body.String())
	}
	var created Report
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.ReporterID != reporterID || created.TargetType != "post" || created.TargetID != 42 ||
		created.Reason != "spam" || created.Status != "open" || created.Details == nil {
		t.Fatalf("report shape = %+v", created)
	}

	rec = modRequest(t, r, http.MethodPost, "/reports", reporterToken, reportBody("post", 42, "harassment"))
	if rec.Code != http.StatusConflict {
		t.Fatalf("duplicate status = %d body=%s", rec.Code, rec.Body.String())
	}
	var envelope struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil || envelope.Error.Code != errAlreadyReport {
		t.Fatalf("duplicate error = %+v %v", envelope, err)
	}

	rec = modRequest(t, r, http.MethodPost, "/reports", reporterToken, reportBody("post", 99, "not_a_reason"))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid reason status = %d", rec.Code)
	}
	var detailEnvelope struct {
		Error struct {
			Code    string         `json:"code"`
			Details map[string]any `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &detailEnvelope); err != nil || detailEnvelope.Error.Details["reason"] == nil {
		t.Fatalf("validation details = %+v %v", detailEnvelope, err)
	}
}

func TestModeratorOnlyAccessResolveAliasesAndAuditLog(t *testing.T) {
	d := testutil.Deps(t)
	_, reporterToken := testutil.User(t, d, "reporter")
	authorID, _ := testutil.User(t, d, "author")
	moderatorID, moderatorToken := testutil.User(t, d, "moderator")
	grantModerator(t, d, moderatorID)
	r := moderationRouter(d)

	if rec := modRequest(t, r, http.MethodGet, "/moderation/reports", reporterToken, ""); rec.Code != http.StatusForbidden {
		t.Fatalf("regular user reports status = %d", rec.Code)
	}
	if rec := modRequest(t, r, http.MethodGet, "/moderation/actions", reporterToken, ""); rec.Code != http.StatusForbidden {
		t.Fatalf("regular user audit status = %d", rec.Code)
	}

	// A post and two reports against it.
	res, err := d.DB.Exec(`
		INSERT INTO posts (author_id, type, title, body) VALUES (?, 'text', 'bad', 'body')`, authorID)
	if err != nil {
		t.Fatal(err)
	}
	postID, _ := res.LastInsertId()
	if rec := modRequest(t, r, http.MethodPost, "/reports", reporterToken, reportBody("post", postID, "spam")); rec.Code != http.StatusCreated {
		t.Fatalf("report status = %d", rec.Code)
	}
	var report Report
	// The reporter and moderator both have perspective; the reporter can read its own report.
	// Use the list as the moderator to obtain the id.
	listRec := modRequest(t, r, http.MethodGet, "/moderation/reports", moderatorToken, "")
	var page struct {
		Items []Report `json:"items"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &page); err != nil || len(page.Items) != 1 {
		t.Fatalf("moderator report list = %s %v", listRec.Body.String(), err)
	}
	report = page.Items[0]

	// UI alias: remove_content must remove the post and return the updated Report.
	rec := modRequest(t, r, http.MethodPost, "/moderation/reports/"+itoa(report.ID)+"/resolve",
		moderatorToken, `{"action":"remove_content","reason":"spam confirmed"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("resolve status = %d body=%s", rec.Code, rec.Body.String())
	}
	var resolved Report
	if err := json.Unmarshal(rec.Body.Bytes(), &resolved); err != nil {
		t.Fatal(err)
	}
	if resolved.Status != "resolved" || resolved.ID != report.ID {
		t.Fatalf("resolved report = %+v", resolved)
	}
	var deleted string
	if err := d.DB.QueryRow(`SELECT deleted_at FROM posts WHERE id = ?`, postID).Scan(&deleted); err != nil || deleted == "" {
		t.Fatalf("post not soft-deleted: %q %v", deleted, err)
	}

	// UI alias: warn_user records a warning against the target author.
	warnRec := modRequest(t, r, http.MethodPost, "/reports", reporterToken, reportBody("user", authorID, "harassment"))
	if warnRec.Code != http.StatusCreated {
		t.Fatalf("user report status = %d", warnRec.Code)
	}
	var warnPage struct {
		Items []Report `json:"items"`
	}
	_ = json.Unmarshal(modRequest(t, r, http.MethodGet, "/moderation/reports", moderatorToken, "").Body.Bytes(), &warnPage)
	var warnReportID int64
	for _, item := range warnPage.Items {
		if item.TargetType == "user" {
			warnReportID = item.ID
		}
	}
	if warnReportID == 0 {
		t.Fatal("user report not found")
	}
	rec = modRequest(t, r, http.MethodPost, "/moderation/reports/"+itoa(warnReportID)+"/resolve",
		moderatorToken, `{"action":"warn_user","reason":"warning issued"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("warn resolve status = %d body=%s", rec.Code, rec.Body.String())
	}

	// Original vocabulary remains valid.
	res, err = d.DB.Exec(`INSERT INTO posts (author_id, type, title, body) VALUES (?, 'text', 'second', 'body')`, authorID)
	if err != nil {
		t.Fatal(err)
	}
	secondPost, _ := res.LastInsertId()
	if rec := modRequest(t, r, http.MethodPost, "/reports", reporterToken, reportBody("post", secondPost, "spam")); rec.Code != http.StatusCreated {
		t.Fatalf("second report status = %d", rec.Code)
	}
	_ = json.Unmarshal(modRequest(t, r, http.MethodGet, "/moderation/reports", moderatorToken, "").Body.Bytes(), &warnPage)
	var dismissID int64
	for _, item := range warnPage.Items {
		if item.TargetID == secondPost {
			dismissID = item.ID
		}
	}
	rec = modRequest(t, r, http.MethodPost, "/moderation/reports/"+itoa(dismissID)+"/resolve",
		moderatorToken, `{"action":"dismiss","reason":"no violation"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("dismiss status = %d", rec.Code)
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resolved); err != nil || resolved.Status != "dismissed" {
		t.Fatalf("dismissed report = %+v %v", resolved, err)
	}

	// Audit log contains both actions with canonical names and the moderator id.
	auditRec := modRequest(t, r, http.MethodGet, "/moderation/actions", moderatorToken, "")
	if auditRec.Code != http.StatusOK {
		t.Fatalf("audit status = %d", auditRec.Code)
	}
	var audit struct {
		Items []ModerationAction `json:"items"`
	}
	if err := json.Unmarshal(auditRec.Body.Bytes(), &audit); err != nil || len(audit.Items) < 3 {
		t.Fatalf("audit log = %s %v", auditRec.Body.String(), err)
	}
	actions := map[string]bool{}
	for _, item := range audit.Items {
		actions[item.Action] = true
		if item.ModeratorID == nil || *item.ModeratorID != moderatorID || item.Scope != "platform" || item.Reason == nil {
			t.Errorf("bad audit entry: %+v", item)
		}
	}
	if !actions["remove"] || !actions["warn"] || !actions["dismiss"] {
		t.Errorf("audit actions = %v", actions)
	}
}

func TestSuspendUserAndBootstrap(t *testing.T) {
	d := testutil.Deps(t)
	authorID, authorToken := testutil.User(t, d, "author")
	moderatorID, moderatorToken := testutil.User(t, d, "moderator")
	grantModerator(t, d, moderatorID)
	r := moderationRouter(d)

	if rec := modRequest(t, r, http.MethodPost, "/users/"+itoa(authorID)+"/suspend", authorToken, `{"reason":"abuse"}`); rec.Code != http.StatusForbidden {
		t.Fatalf("regular user suspend status = %d", rec.Code)
	}
	if rec := modRequest(t, r, http.MethodPost, "/users/"+itoa(authorID)+"/suspend", moderatorToken, `{"reason":"abuse"}`); rec.Code != http.StatusNoContent {
		t.Fatalf("suspend status = %d body=%s", rec.Code, rec.Body.String())
	}
	if testutil.QueryInt(t, d, `SELECT COUNT(*) FROM suspensions WHERE user_id = ? AND is_active = 1`, authorID) != 1 {
		t.Fatal("suspension row missing")
	}

	// Dev bootstrap via env is scoped to non-production.
	t.Setenv("PLATFORM_MODERATOR_EMAIL", "author@example.test")
	fresh := newModule(context.Background(), d)
	granted, err := fresh.isModerator(context.Background(), authorID)
	if err != nil || !granted {
		t.Fatalf("bootstrap grant = %v, %v", granted, err)
	}
}
