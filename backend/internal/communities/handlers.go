package communities

import (
	"database/sql"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/Aswanidev-vs/promiseland/backend/internal/feed/shape"
	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
	"github.com/go-chi/chi/v5"
)

const (
	codeForbidden  = "forbidden"
	codeValidation = "validation_error"
	codeSlugTaken  = "slug_taken"
	codeOwnerLeave = "owner_cannot_leave"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9-]{3,40}$`)

type handler struct{ d *shared.Deps }

// Register mounts public community routes.
func Register(r chi.Router, d *shared.Deps) {
	h := &handler{d: d}
	r.Get("/communities", h.listCommunities)
	r.Get("/communities/{slug}", h.get)
	r.Get("/communities/{slug}/members", h.listMembers)
	r.Get("/communities/{slug}/rules", h.listRules)
	r.Get("/communities/{slug}/posts", h.listPosts)
}

// RegisterProtected mounts authenticated community routes. The orchestrator
// passes a chi router wrapped with shared.RequireAuth. Routes are split by
// HTTP method so public and protected registration never collide.
func RegisterProtected(r chi.Router, d *shared.Deps) {
	h := &handler{d: d}
	r.Post("/communities", h.create)
	r.Patch("/communities/{slug}", h.update)
	r.Post("/communities/{slug}/join", h.join)
	r.Delete("/communities/{slug}/membership", h.leave)
	r.Post("/communities/{slug}/moderators", h.addModerator)
	r.Post("/communities/{slug}/rules", h.addRule)
}

func writeValidation(w http.ResponseWriter, details map[string]string) {
	shared.WriteErrorDetails(w, http.StatusBadRequest, codeValidation, "Please correct the highlighted fields.", details)
}

func isUniqueSlugError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed: communities.slug")
}

func blockClause(d *shared.Deps, viewerID int64) (string, []any) {
	if viewerID <= 0 || !tableExists(d, "blocks") {
		return "", nil
	}
	return ` AND NOT EXISTS (SELECT 1 FROM blocks b WHERE b.blocker_id = p.author_id AND b.blocked_id = ?)` +
			` AND NOT EXISTS (SELECT 1 FROM blocks b WHERE b.blocker_id = ? AND b.blocked_id = p.author_id)`,
		[]any{viewerID, viewerID}
}

// optionalString distinguishes an absent JSON field from an explicit null.
type optionalString struct {
	Set   bool
	Value *string
}

func (o *optionalString) UnmarshalJSON(b []byte) error {
	o.Set = true
	if string(b) == "null" {
		o.Value = nil
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	o.Value = &s
	return nil
}

type ruleInput struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

type communityInput struct {
	Name        string         `json:"name"`
	Slug        string         `json:"slug"`
	Description string         `json:"description"`
	IconURL     optionalString `json:"icon_url"`
	BannerURL   optionalString `json:"banner_url"`
	Visibility  string         `json:"visibility"`
}

func validateCommunity(in communityInput) map[string]string {
	details := map[string]string{}
	n := utf8.RuneCountInString(strings.TrimSpace(in.Name))
	if n < 3 || n > 60 {
		details["name"] = "Name must be between 3 and 60 characters."
	}
	if !slugPattern.MatchString(strings.TrimSpace(in.Slug)) {
		details["slug"] = "Slug must be 3-40 lowercase letters, digits or hyphens."
	}
	if utf8.RuneCountInString(in.Description) > 2000 {
		details["description"] = "Description must be at most 2000 characters."
	}
	if in.Visibility != "" && in.Visibility != "public" && in.Visibility != "private" {
		details["visibility"] = "Visibility must be public or private."
	}
	if in.IconURL.Set && in.IconURL.Value != nil && !shared.ValidProfileImage(*in.IconURL.Value) {
		details["icon_url"] = "Upload an image or enter an http(s) URL."
	}
	if in.BannerURL.Set && in.BannerURL.Value != nil && !shared.ValidProfileImage(*in.BannerURL.Value) {
		details["banner_url"] = "Upload an image or enter an http(s) URL."
	}
	return details
}

var defaultRules = []ruleInput{
	{Title: "Be respectful", Body: "Treat other members with kindness. Harassment and hateful content are not allowed."},
	{Title: "Mark spoilers", Body: "Use spoiler tags and include the relevant season and episode when discussing story details."},
}

func insertRules(tx *sql.Tx, communityID int64, rules []ruleInput) error {
	for i, rule := range rules {
		if _, err := tx.Exec(`INSERT INTO community_rules(community_id, title, body, sort_order) VALUES(?,?,?,?)`,
			communityID, rule.Title, rule.Body, i); err != nil {
			return err
		}
	}
	return nil
}

func (h *handler) create(w http.ResponseWriter, r *http.Request) {
	userID, _ := shared.UserID(r)
	var in communityInput
	if !shared.DecodeJSON(w, r, &in) {
		return
	}
	if details := validateCommunity(in); len(details) > 0 {
		writeValidation(w, details)
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	in.Slug = strings.TrimSpace(in.Slug)
	if in.Visibility == "" {
		in.Visibility = "public"
	}

	tx, err := h.d.DB.BeginTx(r.Context(), nil)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not create the community.")
		return
	}
	defer func() { _ = tx.Rollback() }()

	var existing int
	if err := tx.QueryRow(`SELECT 1 FROM communities WHERE slug = ?`, in.Slug).Scan(&existing); err == nil {
		shared.WriteError(w, http.StatusConflict, codeSlugTaken, "That slug is already in use.")
		return
	} else if !errors.Is(err, sql.ErrNoRows) {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not create the community.")
		return
	}

	res, err := tx.Exec(`
		INSERT INTO communities(owner_id, name, slug, description, icon_url, banner_url,
		                        visibility, posting_enabled, created_at, updated_at)
		VALUES(?,?,?,?,?,?,?,1,datetime('now'),datetime('now'))`,
		userID, in.Name, in.Slug, in.Description, in.IconURL.Value, in.BannerURL.Value, in.Visibility)
	if err != nil {
		if isUniqueSlugError(err) {
			shared.WriteError(w, http.StatusConflict, codeSlugTaken, "That slug is already in use.")
			return
		}
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not create the community.")
		return
	}
	id, _ := res.LastInsertId()
	if _, err := tx.Exec(`
		INSERT INTO community_memberships(community_id, user_id, role, status, joined_at)
		VALUES(?,?,'owner','active',datetime('now'))`, id, userID); err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not create the community.")
		return
	}
	// Every new community starts with the same two baseline rules.
	if err := insertRules(tx, id, defaultRules); err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not create the community.")
		return
	}
	if err := tx.Commit(); err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not create the community.")
		return
	}
	row, err := communityBySlug(r.Context(), h.d, in.Slug, userID)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not load the community.")
		return
	}
	shared.WriteJSON(w, http.StatusCreated, row.Community)
}

func (h *handler) listCommunities(w http.ResponseWriter, r *http.Request) {
	viewer := viewerID(r, h.d)
	p := shared.ParsePagination(r)
	cursor := int64(math.MaxInt64)
	if p.Cursor != "" {
		var err error
		cursor, err = shared.DecodeIDCursor(p.Cursor)
		if err != nil {
			writeValidation(w, map[string]string{"cursor": "Invalid cursor."})
			return
		}
	}
	// Escape LIKE wildcards so a search for "100%" matches the text, not a
	// pattern that sweeps the whole catalogue.
	q := shape.EscapeLike(strings.TrimSpace(r.URL.Query().Get("q")))
	query := communitySelect + `
		WHERE c.id < ?
		  AND (c.visibility = 'public' OR EXISTS(
		        SELECT 1 FROM community_memberships m
		        WHERE m.community_id = c.id AND m.user_id = ? AND m.status = 'active'))
		  AND (? = '' OR c.name LIKE '%' || ? || '%' ESCAPE '\'
		              OR c.slug LIKE '%' || ? || '%' ESCAPE '\')
		ORDER BY c.id DESC LIMIT ?`
	rows, err := h.d.DB.QueryContext(r.Context(), query, viewer, viewer, viewer, cursor, viewer, q, q, q, p.Limit+1)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not list communities.")
		return
	}
	defer rows.Close()
	result := page[Community]{Items: []Community{}, NextCursor: ""}
	lastID := int64(0)
	for rows.Next() {
		row, err := scanCommunity(rows, communityRow{}, viewer)
		if err != nil {
			shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not list communities.")
			return
		}
		if len(result.Items) == p.Limit {
			result.NextCursor = shared.EncodeIDCursor(lastID)
			break
		}
		result.Items = append(result.Items, row.Community)
		lastID = row.ID
	}
	if err := rows.Err(); err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not list communities.")
		return
	}
	shared.WriteJSON(w, http.StatusOK, result)
}

func (h *handler) get(w http.ResponseWriter, r *http.Request) {
	viewer := viewerID(r, h.d)
	row, err := communityBySlug(r.Context(), h.d, chi.URLParam(r, "slug"), viewer)
	if errors.Is(err, sql.ErrNoRows) {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Community not found.")
		return
	}
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not load the community.")
		return
	}
	if row.Visibility == "private" && !row.IsMember {
		shared.WriteError(w, http.StatusForbidden, codeForbidden, "This community is private.")
		return
	}
	shared.WriteJSON(w, http.StatusOK, row.Community)
}

func (h *handler) update(w http.ResponseWriter, r *http.Request) {
	viewer, _ := shared.UserID(r)
	row, err := communityBySlug(r.Context(), h.d, chi.URLParam(r, "slug"), viewer)
	if errors.Is(err, sql.ErrNoRows) {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Community not found.")
		return
	}
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not load the community.")
		return
	}
	if RoleFor(h.d, row.ID, viewer) != "owner" {
		shared.WriteError(w, http.StatusForbidden, codeForbidden, "Only the owner can change community settings.")
		return
	}

	var patch struct {
		Name           *string        `json:"name"`
		Description    *string        `json:"description"`
		IconURL        optionalString `json:"icon_url"`
		BannerURL      optionalString `json:"banner_url"`
		Visibility     *string        `json:"visibility"`
		PostingEnabled *bool          `json:"postingEnabled"`
		// posting_enabled is what the frontend client sends; postingEnabled is
		// the conventions key. Accept both rather than dropping the toggle.
		PostingEnabledSnake *bool        `json:"posting_enabled"`
		Rules               *[]ruleInput `json:"rules"`
	}
	if !shared.DecodeJSON(w, r, &patch) {
		return
	}
	if patch.PostingEnabled == nil {
		patch.PostingEnabled = patch.PostingEnabledSnake
	}
	details := map[string]string{}
	if patch.Name != nil {
		n := utf8.RuneCountInString(strings.TrimSpace(*patch.Name))
		if n < 3 || n > 60 {
			details["name"] = "Name must be between 3 and 60 characters."
		}
	}
	if patch.Description != nil && utf8.RuneCountInString(*patch.Description) > 2000 {
		details["description"] = "Description must be at most 2000 characters."
	}
	if patch.Visibility != nil && *patch.Visibility != "public" && *patch.Visibility != "private" {
		details["visibility"] = "Visibility must be public or private."
	}
	if patch.Rules != nil {
		for i, rule := range *patch.Rules {
			if strings.TrimSpace(rule.Title) == "" || utf8.RuneCountInString(rule.Title) > 120 {
				details["rules"] = "Each rule needs a title of at most 120 characters."
				break
			}
			if utf8.RuneCountInString(rule.Body) > 2000 || i > 50 {
				details["rules"] = "Rules must have a body of at most 2000 characters (max 50 rules)."
				break
			}
		}
	}
	if len(details) > 0 {
		writeValidation(w, details)
		return
	}

	sets := []string{}
	args := []any{}
	if patch.Name != nil {
		sets = append(sets, "name = ?")
		args = append(args, strings.TrimSpace(*patch.Name))
	}
	if patch.Description != nil {
		sets = append(sets, "description = ?")
		args = append(args, *patch.Description)
	}
	if patch.IconURL.Set {
		sets = append(sets, "icon_url = ?")
		args = append(args, patch.IconURL.Value)
	}
	if patch.BannerURL.Set {
		sets = append(sets, "banner_url = ?")
		args = append(args, patch.BannerURL.Value)
	}
	if patch.Visibility != nil {
		sets = append(sets, "visibility = ?")
		args = append(args, *patch.Visibility)
	}
	if patch.PostingEnabled != nil {
		value := 0
		if *patch.PostingEnabled {
			value = 1
		}
		sets = append(sets, "posting_enabled = ?")
		args = append(args, value)
	}

	tx, err := h.d.DB.BeginTx(r.Context(), nil)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not update the community.")
		return
	}
	defer func() { _ = tx.Rollback() }()
	if len(sets) > 0 {
		sets = append(sets, "updated_at = datetime('now')")
		args = append(args, row.ID)
		if _, err := tx.Exec(`UPDATE communities SET `+strings.Join(sets, ", ")+` WHERE id = ?`, args...); err != nil {
			shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not update the community.")
			return
		}
	}
	if patch.Rules != nil {
		if _, err := tx.Exec(`DELETE FROM community_rules WHERE community_id = ?`, row.ID); err != nil {
			shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not update the rules.")
			return
		}
		if err := insertRules(tx, row.ID, *patch.Rules); err != nil {
			shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not update the rules.")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not update the community.")
		return
	}
	updated, err := communityBySlug(r.Context(), h.d, row.Slug, viewer)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not load the community.")
		return
	}
	shared.WriteJSON(w, http.StatusOK, updated.Community)
}

func (h *handler) join(w http.ResponseWriter, r *http.Request) {
	userID, _ := shared.UserID(r)
	row, err := communityBySlug(r.Context(), h.d, chi.URLParam(r, "slug"), userID)
	if errors.Is(err, sql.ErrNoRows) {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Community not found.")
		return
	}
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not load the community.")
		return
	}
	status := "active"
	if row.Visibility == "private" && RoleFor(h.d, row.ID, userID) != "owner" {
		status = "pending"
	}
	_, err = h.d.DB.ExecContext(r.Context(), `
		INSERT INTO community_memberships(community_id, user_id, role, status, joined_at)
		VALUES(?,?,'member',?,datetime('now'))
		ON CONFLICT(community_id, user_id) DO UPDATE SET status = excluded.status
		WHERE community_memberships.status = 'pending'`, row.ID, userID, status)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not join the community.")
		return
	}
	// Report the stored status so repeat joins do not claim "pending" for an
	// already-approved membership.
	var actualStatus string
	if err := h.d.DB.QueryRowContext(r.Context(),
		`SELECT status FROM community_memberships WHERE community_id = ? AND user_id = ?`,
		row.ID, userID).Scan(&actualStatus); err == nil {
		status = actualStatus
	}
	shared.WriteJSON(w, http.StatusOK, map[string]string{"status": status})
}

func (h *handler) leave(w http.ResponseWriter, r *http.Request) {
	userID, _ := shared.UserID(r)
	row, err := communityBySlug(r.Context(), h.d, chi.URLParam(r, "slug"), userID)
	if errors.Is(err, sql.ErrNoRows) {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Community not found.")
		return
	}
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not load the community.")
		return
	}
	role := RoleFor(h.d, row.ID, userID)
	if role == "owner" {
		shared.WriteError(w, http.StatusConflict, codeOwnerLeave, "Transfer ownership before leaving this community.")
		return
	}
	if _, err := h.d.DB.ExecContext(r.Context(),
		`DELETE FROM community_memberships WHERE community_id = ? AND user_id = ?`, row.ID, userID); err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not leave the community.")
		return
	}
	shared.WriteJSON(w, http.StatusOK, nil)
}

func (h *handler) listMembers(w http.ResponseWriter, r *http.Request) {
	viewer := viewerID(r, h.d)
	row, err := communityBySlug(r.Context(), h.d, chi.URLParam(r, "slug"), viewer)
	if errors.Is(err, sql.ErrNoRows) {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Community not found.")
		return
	}
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not load the community.")
		return
	}
	if row.Visibility == "private" && !row.IsMember {
		shared.WriteError(w, http.StatusForbidden, codeForbidden, "This community is private.")
		return
	}
	p := shared.ParsePagination(r)
	cursor := int64(math.MaxInt64)
	if p.Cursor != "" {
		var err error
		cursor, err = shared.DecodeIDCursor(p.Cursor)
		if err != nil {
			writeValidation(w, map[string]string{"cursor": "Invalid cursor."})
			return
		}
	}
	rows, err := h.d.DB.QueryContext(r.Context(), `
		SELECT user_id FROM community_memberships
		WHERE community_id = ? AND status = 'active' AND user_id < ?
		ORDER BY user_id DESC LIMIT ?`, row.ID, cursor, p.Limit+1)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not list members.")
		return
	}
	ids := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()
	result := page[UserSummary]{Items: []UserSummary{}, NextCursor: ""}
	if len(ids) > p.Limit {
		ids = ids[:p.Limit]
		result.NextCursor = shared.EncodeIDCursor(ids[len(ids)-1])
	}
	users := loadUsers(r.Context(), h.d, ids)
	for _, id := range ids {
		if u, ok := users[id]; ok {
			result.Items = append(result.Items, u)
		}
	}
	shared.WriteJSON(w, http.StatusOK, result)
}

func (h *handler) listRules(w http.ResponseWriter, r *http.Request) {
	viewer := viewerID(r, h.d)
	row, err := communityBySlug(r.Context(), h.d, chi.URLParam(r, "slug"), viewer)
	if errors.Is(err, sql.ErrNoRows) {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Community not found.")
		return
	}
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not load the community.")
		return
	}
	if row.Visibility == "private" && !row.IsMember {
		shared.WriteError(w, http.StatusForbidden, codeForbidden, "This community is private.")
		return
	}
	rows, err := h.d.DB.QueryContext(r.Context(), `
		SELECT id, community_id, title, body, sort_order FROM community_rules
		WHERE community_id = ? ORDER BY sort_order, id`, row.ID)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not list rules.")
		return
	}
	defer rows.Close()
	rules := []CommunityRule{}
	for rows.Next() {
		var rule CommunityRule
		if err := rows.Scan(&rule.ID, &rule.CommunityID, &rule.Title, &rule.Body, &rule.SortOrder); err == nil {
			rules = append(rules, rule)
		}
	}
	shared.WriteJSON(w, http.StatusOK, rules)
}

func (h *handler) addRule(w http.ResponseWriter, r *http.Request) {
	viewer, _ := shared.UserID(r)
	row, err := communityBySlug(r.Context(), h.d, chi.URLParam(r, "slug"), viewer)
	if errors.Is(err, sql.ErrNoRows) {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Community not found.")
		return
	}
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not load the community.")
		return
	}
	role := RoleFor(h.d, row.ID, viewer)
	if role != "owner" && role != "moderator" {
		shared.WriteError(w, http.StatusForbidden, codeForbidden, "Only community staff can change the rules.")
		return
	}
	var in ruleInput
	if !shared.DecodeJSON(w, r, &in) {
		return
	}
	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" || utf8.RuneCountInString(in.Title) > 120 || utf8.RuneCountInString(in.Body) > 2000 {
		writeValidation(w, map[string]string{"title": "A rule needs a title of at most 120 characters and a body of at most 2000 characters."})
		return
	}
	var sortOrder int
	_ = h.d.DB.QueryRowContext(r.Context(),
		`SELECT COALESCE(MAX(sort_order), -1) + 1 FROM community_rules WHERE community_id = ?`, row.ID).Scan(&sortOrder)
	res, err := h.d.DB.ExecContext(r.Context(), `
		INSERT INTO community_rules(community_id, title, body, sort_order) VALUES(?,?,?,?)`,
		row.ID, in.Title, in.Body, sortOrder)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not add the rule.")
		return
	}
	id, _ := res.LastInsertId()
	shared.WriteJSON(w, http.StatusCreated, CommunityRule{ID: id, CommunityID: row.ID, Title: in.Title, Body: in.Body, SortOrder: sortOrder})
}

func (h *handler) addModerator(w http.ResponseWriter, r *http.Request) {
	viewer, _ := shared.UserID(r)
	row, err := communityBySlug(r.Context(), h.d, chi.URLParam(r, "slug"), viewer)
	if errors.Is(err, sql.ErrNoRows) {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Community not found.")
		return
	}
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not load the community.")
		return
	}
	if RoleFor(h.d, row.ID, viewer) != "owner" {
		shared.WriteError(w, http.StatusForbidden, codeForbidden, "Only the owner can assign moderators.")
		return
	}
	var in struct {
		UserID int64 `json:"userId"`
		// user_id is what the frontend client sends (src/api/communities.ts).
		UserIDSnake int64 `json:"user_id"`
	}
	if !shared.DecodeJSON(w, r, &in) {
		return
	}
	if in.UserID == 0 {
		in.UserID = in.UserIDSnake
	}
	if in.UserID <= 0 || in.UserID == row.OwnerID {
		writeValidation(w, map[string]string{"userId": "Choose a different existing user."})
		return
	}
	var exists int
	if err := h.d.DB.QueryRowContext(r.Context(), `SELECT 1 FROM users WHERE id = ?`, in.UserID).Scan(&exists); err != nil {
		writeValidation(w, map[string]string{"userId": "Choose a different existing user."})
		return
	}
	_, err = h.d.DB.ExecContext(r.Context(), `
		INSERT INTO community_memberships(community_id, user_id, role, status, joined_at)
		VALUES(?,?,'moderator','active',datetime('now'))
		ON CONFLICT(community_id, user_id) DO UPDATE SET role = 'moderator', status = 'active'`,
		row.ID, in.UserID)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not assign the moderator.")
		return
	}
	shared.WriteJSON(w, http.StatusOK, nil)
}

func (h *handler) listPosts(w http.ResponseWriter, r *http.Request) {
	viewer := viewerID(r, h.d)
	row, err := communityBySlug(r.Context(), h.d, chi.URLParam(r, "slug"), viewer)
	if errors.Is(err, sql.ErrNoRows) {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Community not found.")
		return
	}
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not load the community.")
		return
	}
	if row.Visibility == "private" && !row.IsMember {
		shared.WriteError(w, http.StatusForbidden, codeForbidden, "This community is private.")
		return
	}
	p := shared.ParsePagination(r)
	cursor := int64(math.MaxInt64)
	if p.Cursor != "" {
		var err error
		cursor, err = shared.DecodeIDCursor(p.Cursor)
		if err != nil {
			writeValidation(w, map[string]string{"cursor": "Invalid cursor."})
			return
		}
	}
	clause, args := blockClause(h.d, viewer)
	query := postSelect + ` WHERE p.community_id = ? AND p.deleted_at IS NULL AND p.id < ?` + clause +
		` ORDER BY p.id DESC LIMIT ?`
	queryArgs := append([]any{row.ID, cursor}, args...)
	queryArgs = append(queryArgs, p.Limit+1)
	rows, err := h.d.DB.QueryContext(r.Context(), query, queryArgs...)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not list posts.")
		return
	}
	postRows, err := scanPostRows(rows)
	rows.Close()
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not list posts.")
		return
	}
	result := page[Post]{Items: []Post{}, NextCursor: ""}
	if len(postRows) > p.Limit {
		postRows = postRows[:p.Limit]
		result.NextCursor = shared.EncodeIDCursor(postRows[len(postRows)-1].ID)
	}
	built, err := buildPosts(r.Context(), h.d, postRows, viewer)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not load posts.")
		return
	}
	result.Items = built
	shared.WriteJSON(w, http.StatusOK, result)
}
