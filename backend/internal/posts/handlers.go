package posts

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/Aswanidev-vs/promiseland/backend/internal/communities"
	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
	"github.com/go-chi/chi/v5"
)

const (
	codeForbidden       = "forbidden"
	codeValidation      = "validation_error"
	codePostLocked      = "post_locked"
	codePostingDisabled = "posting_disabled"
)

var allowedTypes = map[string]bool{
	"text": true, "link": true, "image": true,
	"question": true, "poll": true, "announcement": true,
}

type handler struct{ d *shared.Deps }

// Register mounts public post read routes.
func Register(r chi.Router, d *shared.Deps) {
	h := &handler{d: d}
	r.Get("/posts", h.list)
	r.Get("/posts/{id}", h.get)
	r.Get("/posts/{id}/comments", h.listComments)
}

// RegisterProtected mounts authenticated post routes. The orchestrator passes
// a chi router wrapped with shared.RequireAuth. Method-level splitting keeps
// public and protected route registration from colliding.
func RegisterProtected(r chi.Router, d *shared.Deps) {
	h := &handler{d: d}
	r.Post("/posts", h.create)
	r.Patch("/posts/{id}", h.patch)
	r.Delete("/posts/{id}", h.remove)
	r.Post("/posts/{id}/comments", h.addComment)
	r.Post("/posts/{id}/vote", h.vote)
	r.Delete("/posts/{id}/vote", h.removeVote)
	// Aliases matching /posts/{id}/reactions in the API outline.
	r.Post("/posts/{id}/reactions", h.vote)
	r.Delete("/posts/{id}/reactions", h.removeVote)
	r.Post("/posts/{id}/bookmark", h.bookmark)
	r.Delete("/posts/{id}/bookmark", h.unbookmark)
	r.Post("/posts/{id}/poll-vote", h.pollVote)
}

func writeValidation(w http.ResponseWriter, details map[string]string) {
	shared.WriteErrorDetails(w, http.StatusBadRequest, codeValidation, "Please correct the highlighted fields.", details)
}

func parseID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Post not found.")
		return 0, false
	}
	return id, true
}

type pollOptionInput struct {
	ID   int64
	Text string
}

func (p *pollOptionInput) UnmarshalJSON(b []byte) error {
	var text string
	if err := json.Unmarshal(b, &text); err == nil {
		p.Text = text
		return nil
	}
	var object struct {
		ID   int64  `json:"id"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(b, &object); err != nil {
		return err
	}
	p.ID, p.Text = object.ID, object.Text
	return nil
}

type pollInput struct {
	Question string            `json:"question"`
	Options  []pollOptionInput `json:"options"`
}

type postInput struct {
	CommunitySlug   string     `json:"communitySlug"`
	Type            string     `json:"type"`
	Title           string     `json:"title"`
	Body            string     `json:"body"`
	LinkURL         string     `json:"linkUrl"`
	MediaIDs        []int64    `json:"mediaIds"`
	AnimeIDs        []int64    `json:"animeIds"`
	Tags            []string   `json:"tags"`
	Spoiler         bool       `json:"spoiler"`
	SpoilerNote     string     `json:"spoilerNote"`
	Season          int        `json:"season"`
	Episode         int        `json:"episode"`
	CreatorCategory string     `json:"creatorCategory"`
	Poll            *pollInput `json:"poll"`
}

type postPatch struct {
	Type            *string    `json:"type"`
	Title           *string    `json:"title"`
	Body            *string    `json:"body"`
	LinkURL         *string    `json:"linkUrl"`
	MediaIDs        *[]int64   `json:"mediaIds"`
	AnimeIDs        *[]int64   `json:"animeIds"`
	Tags            *[]string  `json:"tags"`
	Spoiler         *bool      `json:"spoiler"`
	SpoilerNote     *string    `json:"spoilerNote"`
	Season          *int       `json:"season"`
	Episode         *int       `json:"episode"`
	CreatorCategory *string    `json:"creatorCategory"`
	IsLocked        *bool      `json:"isLocked"`
	Poll            *pollInput `json:"poll"`
}

// UnmarshalJSON accepts both the canonical camelCase body key and the
// snake_case alias used by the R2 frontend client. camelCase wins when both
// are supplied; the Post wire field remains creator_category.
func (p *postInput) UnmarshalJSON(b []byte) error {
	type plain postInput
	var raw struct {
		plain
		CreatorCategorySnake string `json:"creator_category"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	*p = postInput(raw.plain)
	if p.CreatorCategory == "" {
		p.CreatorCategory = raw.CreatorCategorySnake
	}
	return nil
}

// UnmarshalJSON is the patch-side equivalent of postInput.UnmarshalJSON.
func (p *postPatch) UnmarshalJSON(b []byte) error {
	type plain postPatch
	var raw struct {
		plain
		CreatorCategorySnake *string `json:"creator_category"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	*p = postPatch(raw.plain)
	if p.CreatorCategory == nil {
		p.CreatorCategory = raw.CreatorCategorySnake
	}
	return nil
}

func dedupeIDs(ids []int64) []int64 {
	seen := map[int64]bool{}
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id > 0 && !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

func cleanTags(tags []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(strings.TrimPrefix(tag, "#"))
		if tag == "" || seen[tag] {
			continue
		}
		seen[tag] = true
		out = append(out, tag)
	}
	return out
}

func validatePostContent(in postInput) map[string]string {
	details := map[string]string{}
	if !allowedTypes[in.Type] {
		details["type"] = "Choose a valid post type."
	}
	title := utf8.RuneCountInString(strings.TrimSpace(in.Title))
	if title == 0 || title > 200 {
		details["title"] = "Title must be between 1 and 200 characters."
	}
	if utf8.RuneCountInString(in.Body) > 10000 {
		details["body"] = "Body must be at most 10000 characters."
	}
	if utf8.RuneCountInString(in.SpoilerNote) > 200 {
		details["spoilerNote"] = "Spoiler note must be at most 200 characters."
	}
	if len(dedupeIDs(in.MediaIDs)) > 4 {
		details["mediaIds"] = "A post can have at most 4 media attachments."
	}
	if len(dedupeIDs(in.AnimeIDs)) > 5 {
		details["animeIds"] = "A post can reference at most 5 anime titles."
	}
	if len(cleanTags(in.Tags)) > 5 {
		details["tags"] = "A post can have at most 5 tags."
	}
	for _, tag := range cleanTags(in.Tags) {
		if utf8.RuneCountInString(tag) > 40 {
			details["tags"] = "Each tag must be at most 40 characters."
			break
		}
	}
	if in.Season < 0 || in.Episode < 0 {
		details["episode"] = "Season and episode must not be negative."
	}
	if !validCreatorCategory(in.CreatorCategory) {
		details["creatorCategory"] = "Choose a valid creator category."
	} else {
		in.CreatorCategory = strings.ToLower(strings.TrimSpace(in.CreatorCategory))
	}
	if in.Type == "poll" {
		if in.Poll == nil || len(in.Poll.Options) < 2 || len(in.Poll.Options) > 6 {
			details["poll"] = "Polls need between 2 and 6 options."
		} else {
			for _, option := range in.Poll.Options {
				if strings.TrimSpace(option.Text) == "" || utf8.RuneCountInString(option.Text) > 200 {
					details["poll"] = "Each poll option needs text of at most 200 characters."
					break
				}
			}
		}
	}
	return details
}

func (h *handler) mediaBelongsToAuthor(ids []int64, authorID int64) bool {
	for _, id := range ids {
		var ownerID int64
		err := h.d.DB.QueryRow(`SELECT owner_id FROM media WHERE id = ?`, id).Scan(&ownerID)
		if err != nil || ownerID != authorID {
			return false
		}
	}
	return true
}

func (h *handler) insertRelations(tx *sql.Tx, postID int64, in postInput) error {
	for _, id := range dedupeIDs(in.MediaIDs) {
		if _, err := tx.Exec(`INSERT INTO post_media(post_id, media_id) VALUES(?,?)`, postID, id); err != nil {
			return err
		}
	}
	for _, id := range dedupeIDs(in.AnimeIDs) {
		if _, err := tx.Exec(`INSERT INTO post_anime(post_id, anime_title_id) VALUES(?,?)`, postID, id); err != nil {
			return err
		}
	}
	for _, tag := range cleanTags(in.Tags) {
		if _, err := tx.Exec(`INSERT INTO post_tags(post_id, tag) VALUES(?,?)`, postID, tag); err != nil {
			return err
		}
	}
	if in.Poll != nil {
		for i, option := range in.Poll.Options {
			if _, err := tx.Exec(`INSERT INTO poll_options(post_id, text, sort_order) VALUES(?,?,?)`,
				postID, strings.TrimSpace(option.Text), i); err != nil {
				return err
			}
		}
	}
	return nil
}

func (h *handler) create(w http.ResponseWriter, r *http.Request) {
	userID, _ := shared.UserID(r)
	var in postInput
	if !shared.DecodeJSON(w, r, &in) {
		return
	}
	in.Title = strings.TrimSpace(in.Title)
	if in.Type == "" && in.Poll != nil {
		in.Type = "poll"
	}
	if details := validatePostContent(in); len(details) > 0 {
		writeValidation(w, details)
		return
	}

	var communityID sql.NullInt64
	if slug := strings.TrimSpace(in.CommunitySlug); slug != "" {
		var id int64
		var visibility string
		var posting int
		err := h.d.DB.QueryRowContext(r.Context(),
			`SELECT id, visibility, posting_enabled FROM communities WHERE slug = ?`, slug).
			Scan(&id, &visibility, &posting)
		if errors.Is(err, sql.ErrNoRows) {
			writeValidation(w, map[string]string{"communitySlug": "Community not found."})
			return
		}
		if err != nil {
			shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not load the community.")
			return
		}
		role := communities.RoleFor(h.d, id, userID)
		if role == "" {
			shared.WriteError(w, http.StatusForbidden, codeForbidden, "Join the community before posting.")
			return
		}
		if posting == 0 {
			shared.WriteError(w, http.StatusForbidden, codePostingDisabled, "Posting is disabled in this community.")
			return
		}
		if in.Type == "announcement" && role != "owner" && role != "moderator" {
			shared.WriteError(w, http.StatusForbidden, codeForbidden, "Only community staff can post announcements.")
			return
		}
		communityID = sql.NullInt64{Int64: id, Valid: true}
	}
	if !h.mediaBelongsToAuthor(dedupeIDs(in.MediaIDs), userID) {
		writeValidation(w, map[string]string{"mediaIds": "One or more media attachments do not belong to you."})
		return
	}

	tx, err := h.d.DB.BeginTx(r.Context(), nil)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not publish the post.")
		return
	}
	defer func() { _ = tx.Rollback() }()
	question := ""
	if in.Poll != nil {
		question = strings.TrimSpace(in.Poll.Question)
	}
	res, err := tx.Exec(`
		INSERT INTO posts(author_id, community_id, type, title, body, link_url, spoiler,
		                  spoiler_note, poll_question, season, episode, creator_category,
		                  vote_score, is_locked, deleted_at, created_at, updated_at, edited_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,0,0,NULL,datetime('now'),datetime('now'),NULL)`,
		userID, communityID, in.Type, in.Title, in.Body, in.LinkURL,
		boolInt(in.Spoiler), in.SpoilerNote, nullableString(question), in.Season, in.Episode, in.CreatorCategory)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not publish the post.")
		return
	}
	postID, _ := res.LastInsertId()
	if err := h.insertRelations(tx, postID, in); err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not publish the post.")
		return
	}
	if err := tx.Commit(); err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not publish the post.")
		return
	}
	post, err := h.loadPost(r.Context(), postID, userID)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not load the post.")
		return
	}
	h.notifyMentions(userID, in.Body, "post", postID)
	shared.WriteJSON(w, http.StatusCreated, post)
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func nullableString(v string) any {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return v
}

func (h *handler) loadPost(ctx context.Context, id, viewerID int64) (Post, error) {
	rows, err := h.d.DB.QueryContext(ctx, postSelect+` WHERE p.id = ? AND p.deleted_at IS NULL`, id)
	if err != nil {
		return Post{}, err
	}
	defer rows.Close()
	postRows, err := scanPostRows(rows)
	if err != nil {
		return Post{}, err
	}
	if len(postRows) == 0 {
		return Post{}, sql.ErrNoRows
	}
	built, err := buildPosts(ctx, h.d, postRows, viewerID)
	if err != nil {
		return Post{}, err
	}
	return built[0], nil
}

func (h *handler) get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	viewer := viewerID(r, h.d)
	post, err := h.loadPost(r.Context(), id, viewer)
	if errors.Is(err, sql.ErrNoRows) {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Post not found.")
		return
	}
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not load the post.")
		return
	}
	if !h.canView(r.Context(), id, post.Community, viewer) {
		shared.WriteError(w, http.StatusForbidden, codeForbidden, "You cannot view this post.")
		return
	}
	shared.WriteJSON(w, http.StatusOK, post)
}

func (h *handler) canView(ctx context.Context, postID int64, community *Community, viewerID int64) bool {
	var authorID int64
	var communityID sql.NullInt64
	if err := h.d.DB.QueryRowContext(ctx,
		`SELECT author_id, community_id FROM posts WHERE id = ? AND deleted_at IS NULL`, postID).
		Scan(&authorID, &communityID); err != nil {
		return false
	}
	if isBlocked(h.d, authorID, viewerID) {
		return false
	}
	if communityID.Valid {
		visible, _, found := communityAccess(h.d, communityID.Int64, viewerID)
		if !found || !visible {
			return false
		}
	}
	_ = community
	return true
}

func (h *handler) list(w http.ResponseWriter, r *http.Request) {
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
	args := []any{cursor, viewer, viewer}
	query := postSelect + ` WHERE p.deleted_at IS NULL AND p.id < ?
		AND (p.community_id IS NULL
		     OR NOT EXISTS (SELECT 1 FROM communities c WHERE c.id = p.community_id AND c.visibility = 'private')
		     OR EXISTS (SELECT 1 FROM community_memberships m
		                WHERE m.community_id = p.community_id AND m.user_id = ? AND m.status = 'active')
		     OR EXISTS (SELECT 1 FROM communities c2 WHERE c2.id = p.community_id AND c2.owner_id = ?))`
	if slug := strings.TrimSpace(r.URL.Query().Get("community")); slug != "" {
		var communityID int64
		var visibility string
		err := h.d.DB.QueryRowContext(r.Context(),
			`SELECT id, visibility FROM communities WHERE slug = ?`, slug).Scan(&communityID, &visibility)
		if errors.Is(err, sql.ErrNoRows) {
			shared.WriteJSON(w, http.StatusOK, page[Post]{Items: []Post{}, NextCursor: ""})
			return
		}
		if err != nil {
			shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not load the community.")
			return
		}
		if visibility == "private" && communities.RoleFor(h.d, communityID, viewer) == "" {
			shared.WriteError(w, http.StatusForbidden, codeForbidden, "This community is private.")
			return
		}
		query += ` AND p.community_id = ?`
		args = append(args, communityID)
	}
	creatorCategory := r.URL.Query().Get("creatorCategory")
	if creatorCategory == "" {
		creatorCategory = r.URL.Query().Get("creator_category")
	}
	if creatorCategory != "" {
		category := strings.ToLower(strings.TrimSpace(creatorCategory))
		if !validCreatorCategory(category) {
			writeValidation(w, map[string]string{"creatorCategory": "Choose a valid creator category."})
			return
		}
		if category == "" {
			query += ` AND COALESCE(p.creator_category, '') = ''`
		} else {
			query += ` AND p.creator_category = ?`
			args = append(args, category)
		}
	}
	clause, blockArgs := blockClause(h.d, viewer)
	query += clause + ` ORDER BY p.id DESC LIMIT ?`
	args = append(args, blockArgs...)
	args = append(args, p.Limit+1)
	rows, err := h.d.DB.QueryContext(r.Context(), query, args...)
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

func (h *handler) mayModerate(communityID sql.NullInt64, authorID, userID int64) bool {
	if userID == authorID || isPlatformModerator(h.d, userID) {
		return true
	}
	if !communityID.Valid {
		return false
	}
	role := communities.RoleFor(h.d, communityID.Int64, userID)
	return role == "owner" || role == "moderator"
}

func (h *handler) rawPost(ctx context.Context, id int64) (authorID int64, communityID sql.NullInt64, locked, deleted bool, err error) {
	var lockInt int
	var deletedAt sql.NullString
	err = h.d.DB.QueryRowContext(ctx,
		`SELECT author_id, community_id, is_locked, deleted_at FROM posts WHERE id = ?`, id).
		Scan(&authorID, &communityID, &lockInt, &deletedAt)
	locked = lockInt == 1
	deleted = deletedAt.Valid
	return
}
