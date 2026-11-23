package posts

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
)

func (h *handler) writablePost(ctx context.Context, id, userID int64) (bool, bool) {
	_, communityID, _, deleted, err := h.rawPost(ctx, id)
	if errors.Is(err, sql.ErrNoRows) || deleted {
		return false, false
	}
	if err != nil {
		return false, false
	}
	if communityID.Valid {
		visible, _, found := communityAccess(h.d, communityID.Int64, userID)
		if !found || !visible {
			return false, false
		}
	}
	return true, true
}

func (h *handler) applyVote(w http.ResponseWriter, r *http.Request, value int) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	userID, _ := shared.UserID(r)
	valid, _ := h.writablePost(r.Context(), id, userID)
	if !valid {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Post not found.")
		return
	}
	tx, err := h.d.DB.BeginTx(r.Context(), nil)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not record the vote.")
		return
	}
	defer func() { _ = tx.Rollback() }()
	if value == 0 {
		_, err = tx.Exec(`DELETE FROM votes WHERE user_id = ? AND post_id = ?`, userID, id)
	} else {
		_, err = tx.Exec(`
			INSERT INTO votes(user_id, post_id, value) VALUES(?,?,?)
			ON CONFLICT(user_id, post_id) DO UPDATE SET value = excluded.value`, userID, id, value)
	}
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not record the vote.")
		return
	}
	// Always recompute from the vote rows: a duplicate request can never add
	// to the cached score twice.
	_, err = tx.Exec(`
		UPDATE posts SET vote_score = (SELECT COALESCE(SUM(value), 0) FROM votes WHERE post_id = ?)
		WHERE id = ?`, id, id)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not record the vote.")
		return
	}
	if err := tx.Commit(); err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not record the vote.")
		return
	}
	post, err := h.loadPost(r.Context(), id, userID)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not load the post.")
		return
	}
	shared.WriteJSON(w, http.StatusOK, post)
}

func (h *handler) vote(w http.ResponseWriter, r *http.Request) {
	// Both spellings are accepted: {value} for /posts/{id}/vote and {vote}
	// for the frontend's /posts/{id}/reactions alias. Same handler, no
	// duplicated voting logic.
	var in struct {
		Value *int `json:"value"`
		Vote  *int `json:"vote"`
	}
	if !shared.DecodeJSON(w, r, &in) {
		return
	}
	chosen := in.Value
	if in.Vote != nil {
		chosen = in.Vote
	}
	if chosen == nil {
		writeValidation(w, map[string]string{"vote": "Vote must be -1, 0 or 1."})
		return
	}
	if *chosen < -1 || *chosen > 1 {
		writeValidation(w, map[string]string{"vote": "Vote must be -1, 0 or 1."})
		return
	}
	h.applyVote(w, r, *chosen)
}

func (h *handler) removeVote(w http.ResponseWriter, r *http.Request) {
	h.applyVote(w, r, 0)
}

func (h *handler) bookmark(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	userID, _ := shared.UserID(r)
	valid, _ := h.writablePost(r.Context(), id, userID)
	if !valid {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Post not found.")
		return
	}
	_, err := h.d.DB.ExecContext(r.Context(),
		`INSERT OR IGNORE INTO bookmarks(user_id, post_id, created_at) VALUES(?,?,datetime('now'))`, userID, id)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not bookmark the post.")
		return
	}
	shared.WriteJSON(w, http.StatusOK, nil)
}

func (h *handler) unbookmark(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	userID, _ := shared.UserID(r)
	_, err := h.d.DB.ExecContext(r.Context(),
		`DELETE FROM bookmarks WHERE user_id = ? AND post_id = ?`, userID, id)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not remove the bookmark.")
		return
	}
	shared.WriteJSON(w, http.StatusOK, nil)
}

func (h *handler) pollVote(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	userID, _ := shared.UserID(r)
	valid, _ := h.writablePost(r.Context(), id, userID)
	if !valid {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Post not found.")
		return
	}
	var postType string
	if err := h.d.DB.QueryRowContext(r.Context(), `SELECT type FROM posts WHERE id = ?`, id).Scan(&postType); err != nil {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Poll not found.")
		return
	}
	if postType != "poll" {
		writeValidation(w, map[string]string{"optionId": "This post is not a poll."})
		return
	}
	var in struct {
		OptionID int64 `json:"optionId"`
	}
	if !shared.DecodeJSON(w, r, &in) {
		return
	}
	if in.OptionID <= 0 {
		writeValidation(w, map[string]string{"optionId": "Choose a poll option."})
		return
	}
	var optionPostID int64
	if err := h.d.DB.QueryRowContext(r.Context(),
		`SELECT post_id FROM poll_options WHERE id = ?`, in.OptionID).Scan(&optionPostID); err != nil || optionPostID != id {
		writeValidation(w, map[string]string{"optionId": "Choose an option from this poll."})
		return
	}
	tx, err := h.d.DB.BeginTx(r.Context(), nil)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not record the vote.")
		return
	}
	defer func() { _ = tx.Rollback() }()
	_, err = tx.Exec(`
		INSERT OR IGNORE INTO poll_votes(post_id, user_id, option_id) VALUES(?,?,?)`,
		id, userID, in.OptionID)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not record the vote.")
		return
	}
	if err := tx.Commit(); err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not record the vote.")
		return
	}
	post, err := h.loadPost(r.Context(), id, userID)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not load the post.")
		return
	}
	shared.WriteJSON(w, http.StatusOK, post)
}
