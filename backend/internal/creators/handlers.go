package creators

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Aswanidev-vs/promiseland/backend/internal/feed/shape"
	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
	"github.com/go-chi/chi/v5"
)

func (s *service) list(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.d == nil || s.d.DB == nil {
		shared.WriteError(w, http.StatusServiceUnavailable, shared.CodeDBUnavailable, "Creator discovery is temporarily unavailable.")
		return
	}
	p := shared.ParsePagination(r)
	cursor := int64(math.MaxInt64)
	if p.Cursor != "" {
		var err error
		cursor, err = shared.DecodeIDCursor(p.Cursor)
		if err != nil {
			shared.WriteErrorDetails(w, http.StatusBadRequest, codeValidation,
				"Invalid pagination cursor.", map[string]string{"cursor": "must be a valid cursor"})
			return
		}
	}

	category := normalizeCategory(r.URL.Query().Get("category"))
	if category != "" && !validCategories[category] {
		shared.WriteErrorDetails(w, http.StatusBadRequest, codeValidation,
			"Invalid creator category.", map[string]string{"category": "choose a valid creator category"})
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len([]rune(q)) > 100 {
		q = string([]rune(q)[:100])
	}

	viewer := viewerID(r, s.d)
	args := []any{cursor}
	query := `
		SELECT u.id, ` + s.usernameExpr() + `, ` + s.displayExpr() + `, up.avatar_url
		FROM users u
		LEFT JOIN user_profiles up ON up.user_id = u.id
		WHERE u.id < ?`
	available, availableArgs := s.creatorAvailable(viewer)
	if available != "" {
		query += " AND " + available
		args = append(args, availableArgs...)
	}
	query += ` AND EXISTS (SELECT 1 FROM posts cp
			WHERE cp.author_id = u.id AND ` + s.postAlive("cp") + `
			  AND COALESCE(cp.creator_category, '') <> '')`
	if category != "" {
		query += ` AND EXISTS (SELECT 1 FROM posts cpf
			WHERE cpf.author_id = u.id AND ` + s.postAlive("cpf") + `
			  AND cpf.creator_category = ?)`
		args = append(args, category)
	}
	if q != "" {
		query += ` AND (LOWER(` + s.usernameExpr() + `) LIKE ? ESCAPE '\'
			OR LOWER(` + s.displayExpr() + `) LIKE ? ESCAPE '\')`
		pattern := "%" + shape.EscapeLike(strings.ToLower(q)) + "%"
		args = append(args, pattern, pattern)
	}
	query += ` ORDER BY u.id DESC LIMIT ?`
	args = append(args, p.Limit+1)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	rows, err := s.d.DB.QueryContext(ctx, query, args...)
	if err != nil {
		s.writeInternal(w, "list creators", err)
		return
	}
	users, err := scanCreatorRows(rows)
	rows.Close()
	if err != nil {
		s.writeInternal(w, "scan creators", err)
		return
	}

	result := page[Creator]{Items: []Creator{}, NextCursor: ""}
	if len(users) > p.Limit {
		users = users[:p.Limit]
		result.NextCursor = shared.EncodeIDCursor(users[len(users)-1].ID)
	}
	creators, err := s.buildCreators(ctx, users, viewer)
	if err != nil {
		s.writeInternal(w, "build creators", err)
		return
	}
	result.Items = creators
	shared.WriteJSON(w, http.StatusOK, result)
}

func (s *service) getByUsername(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.d == nil || s.d.DB == nil {
		shared.WriteError(w, http.StatusServiceUnavailable, shared.CodeDBUnavailable, "Creator discovery is temporarily unavailable.")
		return
	}
	username := strings.TrimSpace(chi.URLParam(r, "username"))
	if username == "" {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Creator not found.")
		return
	}
	viewer := viewerID(r, s.d)
	args := []any{strings.ToLower(username)}
	query := `
		SELECT u.id, ` + s.usernameExpr() + `, ` + s.displayExpr() + `, up.avatar_url
		FROM users u
		LEFT JOIN user_profiles up ON up.user_id = u.id
		WHERE LOWER(` + s.usernameExpr() + `) = ?`
	available, availableArgs := s.creatorAvailable(viewer)
	if available != "" {
		query += " AND " + available
		args = append(args, availableArgs...)
	}
	query += ` AND EXISTS (SELECT 1 FROM posts cp
		WHERE cp.author_id = u.id AND ` + s.postAlive("cp") + `
		  AND COALESCE(cp.creator_category, '') <> '') LIMIT 1`

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	rows, err := s.d.DB.QueryContext(ctx, query, args...)
	if err != nil {
		s.writeInternal(w, "get creator", err)
		return
	}
	users, err := scanCreatorRows(rows)
	rows.Close()
	if err != nil {
		s.writeInternal(w, "scan creator", err)
		return
	}
	if len(users) == 0 {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Creator not found.")
		return
	}
	creators, err := s.buildCreators(ctx, users, viewer)
	if err != nil || len(creators) == 0 {
		s.writeInternal(w, "build creator", err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, creators[0])
}

func scanCreatorRows(rows *sql.Rows) ([]UserSummary, error) {
	out := []UserSummary{}
	for rows.Next() {
		var u UserSummary
		if err := rows.Scan(&u.ID, &u.Username, &u.DisplayName, &u.AvatarURL); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func parsePostID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "postId"), 10, 64)
	if err != nil || id <= 0 {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Post not found.")
		return 0, false
	}
	return id, true
}

// ownedPost verifies that the post exists, is not deleted and belongs to the
// signed-in user. Feature toggles are strictly owner-only.
func (s *service) ownedPost(ctx context.Context, postID, userID int64) error {
	var authorID int64
	var deleted sql.NullString
	err := s.d.DB.QueryRowContext(ctx,
		`SELECT author_id, deleted_at FROM posts WHERE id = ?`, postID).
		Scan(&authorID, &deleted)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && deleted.Valid) {
		return sql.ErrNoRows
	}
	if err != nil {
		return err
	}
	if authorID != userID {
		return errForbidden
	}
	return nil
}

var errForbidden = errors.New("forbidden")

func (s *service) feature(w http.ResponseWriter, r *http.Request) {
	userID, ok := shared.UserID(r)
	if !ok {
		shared.WriteError(w, http.StatusUnauthorized, shared.CodeAuthRequired, "Sign in to continue.")
		return
	}
	postID, ok := parsePostID(w, r)
	if !ok {
		return
	}
	if err := s.ownedPost(r.Context(), postID, userID); errors.Is(err, sql.ErrNoRows) {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Post not found.")
		return
	} else if errors.Is(err, errForbidden) {
		shared.WriteError(w, http.StatusForbidden, codeForbidden, "You can only feature your own posts.")
		return
	} else if err != nil {
		s.writeInternal(w, "feature ownership", err)
		return
	}

	tx, err := s.d.DB.BeginTx(r.Context(), nil)
	if err != nil {
		s.writeInternal(w, "feature begin", err)
		return
	}
	defer func() { _ = tx.Rollback() }()

	var count, maxSort int
	if err := tx.QueryRowContext(r.Context(),
		`SELECT COUNT(*), COALESCE(MAX(sort_order), 0) FROM featured_posts WHERE user_id = ?`, userID).
		Scan(&count, &maxSort); err != nil {
		s.writeInternal(w, "feature count", err)
		return
	}
	var already int
	if err := tx.QueryRowContext(r.Context(),
		`SELECT EXISTS(SELECT 1 FROM featured_posts WHERE user_id = ? AND post_id = ?)`, userID, postID).
		Scan(&already); err != nil {
		s.writeInternal(w, "feature exists", err)
		return
	}
	if already == 0 && count >= maxFeatured {
		shared.WriteError(w, http.StatusConflict, codeLimit,
			"You can feature at most six posts. Remove one first.")
		return
	}
	if already == 0 {
		if _, err := tx.ExecContext(r.Context(), `
			INSERT INTO featured_posts(user_id, post_id, sort_order, created_at)
			VALUES (?, ?, ?, datetime('now'))`, userID, postID, maxSort+1); err != nil {
			s.writeInternal(w, "feature insert", err)
			return
		}
	}
	if err := tx.Commit(); err != nil {
		s.writeInternal(w, "feature commit", err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, nil)
}

func (s *service) unfeature(w http.ResponseWriter, r *http.Request) {
	userID, ok := shared.UserID(r)
	if !ok {
		shared.WriteError(w, http.StatusUnauthorized, shared.CodeAuthRequired, "Sign in to continue.")
		return
	}
	postID, ok := parsePostID(w, r)
	if !ok {
		return
	}
	if err := s.ownedPost(r.Context(), postID, userID); errors.Is(err, sql.ErrNoRows) {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Post not found.")
		return
	} else if errors.Is(err, errForbidden) {
		shared.WriteError(w, http.StatusForbidden, codeForbidden, "You can only feature your own posts.")
		return
	} else if err != nil {
		s.writeInternal(w, "unfeature ownership", err)
		return
	}
	if _, err := s.d.DB.ExecContext(r.Context(),
		`DELETE FROM featured_posts WHERE user_id = ? AND post_id = ?`, userID, postID); err != nil {
		s.writeInternal(w, "unfeature delete", err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, nil)
}
