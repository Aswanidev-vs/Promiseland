package users

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
)

// notifyFollow writes a notification when the notifications table is already
// available. Its migration is owned by another module, so a missing table (or
// an incompatible early schema) is logged and ignored rather than failing the
// follow request.
func (s *service) notifyFollow(actorID, targetID int64) {
	var exists int
	err := s.d.DB.QueryRow(`
		SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = 'notifications' LIMIT 1`).Scan(&exists)
	if err != nil {
		return
	}
	_, err = s.d.DB.Exec(`
		INSERT INTO notifications (user_id, type, actor_id, target_type, target_id, created_at)
		VALUES (?, 'follow', ?, 'user', ?, datetime('now'))`, targetID, actorID, targetID)
	if err != nil && s.d.Log != nil {
		s.d.Log.Warn("users: could not create follow notification", "error", err)
	}
}

func (s *service) follow(w http.ResponseWriter, r *http.Request) {
	actorID, ok := shared.UserID(r)
	if !ok {
		shared.WriteError(w, http.StatusUnauthorized, shared.CodeAuthRequired, "Sign in to continue.")
		return
	}
	targetID, err := parseID(r)
	if err != nil {
		shared.WriteError(w, http.StatusBadRequest, codeInvalidID, "Invalid user id.")
		return
	}
	if targetID == actorID {
		shared.WriteError(w, http.StatusBadRequest, "cannot_follow_self", "You cannot follow yourself.")
		return
	}
	user, err := s.loadUser(r.Context(), targetID)
	if errors.Is(err, sql.ErrNoRows) {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "User not found.")
		return
	}
	if err != nil {
		internalError(w, s.d, "follow: load target", err)
		return
	}
	if user.Visibility == "private" {
		shared.WriteError(w, http.StatusForbidden, codeProfilePrivate, "This profile is private.")
		return
	}
	blocked, err := s.isBlocked(r.Context(), actorID, targetID)
	if err != nil {
		internalError(w, s.d, "follow: check block", err)
		return
	}
	if blocked {
		shared.WriteError(w, http.StatusForbidden, codeBlocked, "You cannot follow this user.")
		return
	}
	result, err := s.d.DB.ExecContext(r.Context(),
		`INSERT OR IGNORE INTO follows (follower_id, followed_id, created_at)
		 VALUES (?, ?, datetime('now'))`, actorID, targetID)
	if err != nil {
		internalError(w, s.d, "follow: insert", err)
		return
	}
	// Only a new relationship produces a notification.
	if affected, rowsErr := result.RowsAffected(); rowsErr == nil && affected > 0 {
		s.notifyFollow(actorID, targetID)
	}
	shared.WriteJSON(w, http.StatusOK, nil)
}

func (s *service) unfollow(w http.ResponseWriter, r *http.Request) {
	actorID, ok := shared.UserID(r)
	if !ok {
		shared.WriteError(w, http.StatusUnauthorized, shared.CodeAuthRequired, "Sign in to continue.")
		return
	}
	targetID, err := parseID(r)
	if err != nil {
		shared.WriteError(w, http.StatusBadRequest, codeInvalidID, "Invalid user id.")
		return
	}
	if targetID == actorID {
		shared.WriteError(w, http.StatusBadRequest, "cannot_follow_self", "You cannot unfollow yourself.")
		return
	}
	if _, err := s.d.DB.ExecContext(r.Context(),
		`DELETE FROM follows WHERE follower_id = ? AND followed_id = ?`, actorID, targetID); err != nil {
		internalError(w, s.d, "unfollow: delete", err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, nil)
}

func (s *service) block(w http.ResponseWriter, r *http.Request) {
	actorID, ok := shared.UserID(r)
	if !ok {
		shared.WriteError(w, http.StatusUnauthorized, shared.CodeAuthRequired, "Sign in to continue.")
		return
	}
	targetID, err := parseID(r)
	if err != nil {
		shared.WriteError(w, http.StatusBadRequest, codeInvalidID, "Invalid user id.")
		return
	}
	if targetID == actorID {
		shared.WriteError(w, http.StatusBadRequest, "cannot_block_self", "You cannot block yourself.")
		return
	}
	var exists int
	err = s.d.DB.QueryRowContext(r.Context(),
		`SELECT 1 FROM users WHERE id = ? AND is_active = 1 LIMIT 1`, targetID).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "User not found.")
		return
	}
	if err != nil {
		internalError(w, s.d, "block: lookup target", err)
		return
	}
	if _, err := s.d.DB.ExecContext(r.Context(),
		`INSERT OR IGNORE INTO blocks (blocker_id, blocked_id, created_at)
		 VALUES (?, ?, datetime('now'))`, actorID, targetID); err != nil {
		internalError(w, s.d, "block: insert", err)
		return
	}
	// Blocking removes any existing relationship in either direction.
	if _, err := s.d.DB.ExecContext(r.Context(), `
		DELETE FROM follows
		WHERE (follower_id = ? AND followed_id = ?) OR (follower_id = ? AND followed_id = ?)`,
		actorID, targetID, targetID, actorID); err != nil {
		internalError(w, s.d, "block: remove follows", err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, nil)
}

func (s *service) unblock(w http.ResponseWriter, r *http.Request) {
	actorID, ok := shared.UserID(r)
	if !ok {
		shared.WriteError(w, http.StatusUnauthorized, shared.CodeAuthRequired, "Sign in to continue.")
		return
	}
	targetID, err := parseID(r)
	if err != nil {
		shared.WriteError(w, http.StatusBadRequest, codeInvalidID, "Invalid user id.")
		return
	}
	if _, err := s.d.DB.ExecContext(r.Context(),
		`DELETE FROM blocks WHERE blocker_id = ? AND blocked_id = ?`, actorID, targetID); err != nil {
		internalError(w, s.d, "unblock: delete", err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, nil)
}
