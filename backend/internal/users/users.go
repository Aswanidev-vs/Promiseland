// Package users implements profiles, interests, follows and blocks for the R1
// social foundation.
package users

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
	"github.com/go-chi/chi/v5"
)

const userColumns = `
	SELECT u.id, u.email, u.username, u.display_name,
	       p.avatar_url, p.banner_url, p.bio, p.location,
	       COALESCE(p.visibility, 'public'), u.discoverable, u.created_at
	FROM users u
	LEFT JOIN user_profiles p ON p.user_id = u.id`

type service struct {
	d *shared.Deps
}

// Register mounts public profile routes under /users.
func Register(r chi.Router, d *shared.Deps) {
	s := &service{d: d}
	r.Get("/users/{username}", s.getProfile)
}

// RegisterProtected mounts routes requiring a valid session. The orchestrator
// passes a router already wrapped with shared.RequireAuth.
func RegisterProtected(r chi.Router, d *shared.Deps) {
	s := &service{d: d}
	r.Get("/users/me", s.getMe)
	r.Patch("/users/me", s.patchMe)
	r.Get("/users/me/interests", s.listInterests)
	r.Put("/users/me/interests", s.replaceInterests)
	r.Put("/users/me/anime-interests", s.replaceAnimeInterests)
	r.Post("/users/{id}/follow", s.follow)
	r.Delete("/users/{id}/follow", s.unfollow)
	r.Post("/users/{id}/block", s.block)
	r.Delete("/users/{id}/block", s.unblock)
}

func internalError(w http.ResponseWriter, d *shared.Deps, action string, err error) {
	if d != nil && d.Log != nil {
		d.Log.Error("users: "+action, "error", err)
	}
	shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError,
		"An unexpected error occurred.")
}

func (s *service) loadUser(ctx context.Context, id int64) (User, error) {
	var user User
	var avatar, banner, bio, location sql.NullString
	err := s.d.DB.QueryRowContext(ctx, userColumns+` WHERE u.id = ?`, id).Scan(
		&user.ID, &user.Email, &user.Username, &user.DisplayName,
		&avatar, &banner, &bio, &location,
		&user.Visibility, &user.Discoverable, &user.CreatedAt,
	)
	if err != nil {
		return User{}, err
	}
	user.AvatarURL = stringPtr(avatar)
	user.BannerURL = stringPtr(banner)
	user.Bio = stringPtr(bio)
	user.Location = stringPtr(location)
	return user, nil
}

func (s *service) userIDByUsername(ctx context.Context, username string) (int64, error) {
	var id int64
	err := s.d.DB.QueryRowContext(ctx,
		`SELECT id FROM users WHERE username = ? AND is_active = 1 LIMIT 1`,
		strings.ToLower(strings.TrimSpace(username))).Scan(&id)
	return id, err
}

// isBlocked reports whether a block exists in either direction.
func (s *service) isBlocked(ctx context.Context, a, b int64) (bool, error) {
	var one int
	err := s.d.DB.QueryRowContext(ctx, `
		SELECT 1 FROM blocks
		WHERE (blocker_id = ? AND blocked_id = ?) OR (blocker_id = ? AND blocked_id = ?)
		LIMIT 1`, a, b, b, a).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func parseID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("invalid id")
	}
	return id, nil
}

func (s *service) postCount(ctx context.Context, userID int64) int64 {
	// The posts table is owned by another module; if it is not present yet the
	// profile still renders with a zero count.
	var count int64
	if err := s.d.DB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM posts WHERE author_id = ?`, userID).Scan(&count); err != nil {
		return 0
	}
	return count
}
