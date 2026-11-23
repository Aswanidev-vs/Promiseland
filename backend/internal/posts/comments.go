package posts

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
)

func (h *handler) commentByID(ctx context.Context, id int64) (Comment, error) {
	var c Comment
	var authorID int64
	var parentID sql.NullInt64
	var body sql.NullString
	var deletedAt sql.NullString
	err := h.d.DB.QueryRowContext(ctx, `
		SELECT id, post_id, author_id, parent_comment_id, body, created_at,
		       COALESCE(edited_at, ''), deleted_at
		FROM comments WHERE id = ?`, id).
		Scan(&c.ID, &c.PostID, &authorID, &parentID, &body, &c.CreatedAt, &c.EditedAt, &deletedAt)
	if err != nil {
		return c, err
	}
	c.Deleted = deletedAt.Valid
	if c.Deleted {
		// Hide the body but keep the placeholder so replies still make sense.
		c.Body = ""
	} else {
		c.Body = body.String
	}
	if parentID.Valid {
		p := parentID.Int64
		c.ParentID = &p
	}
	users := loadUsers(ctx, h.d, []int64{authorID})
	c.Author = users[authorID]
	if c.Author.ID == 0 {
		c.Author = UserSummary{ID: authorID, DisplayName: "unknown"}
	}
	return c, nil
}

func (h *handler) listComments(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	viewer := viewerID(r, h.d)
	_, communityID, _, deleted, err := h.rawPost(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) || deleted {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Post not found.")
		return
	}
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not load the post.")
		return
	}
	if communityID.Valid {
		visible, _, found := communityAccess(h.d, communityID.Int64, viewer)
		if !found || !visible {
			shared.WriteError(w, http.StatusForbidden, codeForbidden, "You cannot view these comments.")
			return
		}
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
		SELECT id FROM comments WHERE post_id = ? AND id < ? ORDER BY id DESC LIMIT ?`,
		id, cursor, p.Limit+1)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not list comments.")
		return
	}
	ids := []int64{}
	for rows.Next() {
		var commentID int64
		if err := rows.Scan(&commentID); err == nil {
			ids = append(ids, commentID)
		}
	}
	rows.Close()
	result := page[Comment]{Items: []Comment{}, NextCursor: ""}
	if len(ids) > p.Limit {
		ids = ids[:p.Limit]
		result.NextCursor = shared.EncodeIDCursor(ids[len(ids)-1])
	}
	for _, commentID := range ids {
		comment, err := h.commentByID(r.Context(), commentID)
		if err == nil {
			result.Items = append(result.Items, comment)
		}
	}
	shared.WriteJSON(w, http.StatusOK, result)
}

// commentDepth validates that a reply stays within the three-level nesting
// limit and that its parent belongs to the same post.
func (h *handler) commentDepth(ctx context.Context, postID, parentID int64) (int, error) {
	depth := 2
	current := parentID
	for i := 0; i < 8; i++ {
		var next sql.NullInt64
		if err := h.d.DB.QueryRowContext(ctx,
			`SELECT parent_comment_id FROM comments WHERE id = ? AND post_id = ?`,
			current, postID).Scan(&next); err != nil {
			return 0, errors.New("invalid parent")
		}
		if !next.Valid {
			return depth, nil
		}
		depth++
		if depth > 3 {
			return 0, errors.New("too deep")
		}
		current = next.Int64
	}
	return 0, errors.New("too deep")
}

func (h *handler) addComment(w http.ResponseWriter, r *http.Request) {
	postID, ok := parseID(w, r)
	if !ok {
		return
	}
	userID, _ := shared.UserID(r)
	authorID, communityID, locked, deleted, err := h.rawPost(r.Context(), postID)
	if errors.Is(err, sql.ErrNoRows) || deleted {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Post not found.")
		return
	}
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not load the post.")
		return
	}
	_ = authorID
	if communityID.Valid {
		visible, _, found := communityAccess(h.d, communityID.Int64, userID)
		if !found || !visible {
			shared.WriteError(w, http.StatusForbidden, codeForbidden, "You cannot comment on this post.")
			return
		}
	}
	if locked {
		shared.WriteError(w, http.StatusForbidden, codePostLocked, "Comments are locked on this post.")
		return
	}
	var in struct {
		Body     string `json:"body"`
		ParentID *int64 `json:"parentId"`
	}
	if !shared.DecodeJSON(w, r, &in) {
		return
	}
	in.Body = strings.TrimSpace(in.Body)
	if in.Body == "" || utf8.RuneCountInString(in.Body) > 2000 {
		writeValidation(w, map[string]string{"body": "Comment must be between 1 and 2000 characters."})
		return
	}
	if in.ParentID != nil {
		if *in.ParentID <= 0 {
			writeValidation(w, map[string]string{"parentId": "Invalid parent comment."})
			return
		}
		if _, err := h.commentDepth(r.Context(), postID, *in.ParentID); err != nil {
			writeValidation(w, map[string]string{"parentId": "Replies are limited to three levels."})
			return
		}
	}
	var parent sql.NullInt64
	if in.ParentID != nil {
		parent = sql.NullInt64{Int64: *in.ParentID, Valid: true}
	}
	res, err := h.d.DB.ExecContext(r.Context(), `
		INSERT INTO comments(post_id, author_id, parent_comment_id, body, created_at)
		VALUES(?,?,?,?,datetime('now'))`, postID, userID, parent, in.Body)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not publish the comment.")
		return
	}
	commentID, _ := res.LastInsertId()
	if in.ParentID != nil {
		var parentAuthor int64
		if err := h.d.DB.QueryRowContext(r.Context(),
			`SELECT author_id FROM comments WHERE id = ?`, *in.ParentID).Scan(&parentAuthor); err == nil {
			notifyUser(h.d, parentAuthor, userID, "reply", "comment", commentID)
		}
	}
	h.notifyMentions(userID, in.Body, "comment", commentID)
	comment, err := h.commentByID(r.Context(), commentID)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not load the comment.")
		return
	}
	shared.WriteJSON(w, http.StatusCreated, comment)
}
