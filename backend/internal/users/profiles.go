package users

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
	"github.com/go-chi/chi/v5"
)

func (s *service) loadProfile(ctx context.Context, id, viewerID int64) (UserProfile, error) {
	user, err := s.loadUser(ctx, id)
	if err != nil {
		return UserProfile{}, err
	}
	profile := UserProfile{
		ID:             user.ID,
		Username:       user.Username,
		DisplayName:    user.DisplayName,
		AvatarURL:      user.AvatarURL,
		BannerURL:      user.BannerURL,
		Bio:            user.Bio,
		Location:       user.Location,
		Visibility:     user.Visibility,
		Discoverable:   user.Discoverable,
		CreatedAt:      user.CreatedAt,
		Interests:      []Interest{},
		AnimeInterests: []AnimeInterest{},
		PostCount:      s.postCount(ctx, id),
	}
	if viewerID == id {
		profile.Email = user.Email
	}

	interestRows, err := s.d.DB.QueryContext(ctx, `
		SELECT i.id, i.name, i.category
		FROM user_interests ui
		JOIN interests i ON i.id = ui.interest_id
		WHERE ui.user_id = ?
		ORDER BY ui.sort_order, i.id`, id)
	if err != nil {
		return UserProfile{}, err
	}
	for interestRows.Next() {
		var interest Interest
		if err := interestRows.Scan(&interest.ID, &interest.Name, &interest.Category); err != nil {
			_ = interestRows.Close()
			return UserProfile{}, err
		}
		profile.Interests = append(profile.Interests, interest)
	}
	if err := interestRows.Err(); err != nil {
		_ = interestRows.Close()
		return UserProfile{}, err
	}
	_ = interestRows.Close()

	animeRows, err := s.d.DB.QueryContext(ctx, `
		SELECT a.id, COALESCE(a.external_id, ''), a.title,
		       COALESCE(a.alternate_title, ''), COALESCE(a.format, ''),
		       ai.interest_type, ai.sort_order
		FROM anime_interests ai
		JOIN anime_titles a ON a.id = ai.anime_title_id
		WHERE ai.user_id = ?
		ORDER BY ai.sort_order, ai.anime_title_id, ai.interest_type`, id)
	if err != nil {
		return UserProfile{}, err
	}
	for animeRows.Next() {
		var item AnimeInterest
		if err := animeRows.Scan(
			&item.Anime.ID, &item.Anime.ExternalID, &item.Anime.Title,
			&item.Anime.AlternateTitle, &item.Anime.Format,
			&item.InterestType, &item.SortOrder,
		); err != nil {
			_ = animeRows.Close()
			return UserProfile{}, err
		}
		profile.AnimeInterests = append(profile.AnimeInterests, item)
	}
	if err := animeRows.Err(); err != nil {
		_ = animeRows.Close()
		return UserProfile{}, err
	}
	_ = animeRows.Close()

	if err := s.d.DB.QueryRowContext(ctx, `
		SELECT
			(SELECT COUNT(*) FROM follows WHERE followed_id = ?),
			(SELECT COUNT(*) FROM follows WHERE follower_id = ?)`, id, id).
		Scan(&profile.FollowerCount, &profile.FollowingCount); err != nil {
		return UserProfile{}, err
	}

	if viewerID > 0 {
		var one int
		err := s.d.DB.QueryRowContext(ctx,
			`SELECT 1 FROM follows WHERE follower_id = ? AND followed_id = ? LIMIT 1`,
			viewerID, id).Scan(&one)
		if err == nil {
			profile.IsFollowing = true
		} else if !errors.Is(err, sql.ErrNoRows) {
			return UserProfile{}, err
		}
		blocked, err := s.isBlocked(ctx, viewerID, id)
		if err != nil {
			return UserProfile{}, err
		}
		profile.IsBlocked = blocked
	}
	return profile, nil
}

func (s *service) getProfile(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimSpace(chi.URLParam(r, "username"))
	targetID, err := s.userIDByUsername(r.Context(), username)
	if errors.Is(err, sql.ErrNoRows) {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "User not found.")
		return
	}
	if err != nil {
		internalError(w, s.d, "get profile: lookup user", err)
		return
	}

	viewerID, _ := s.d.SessionUserID(r)
	if viewerID > 0 {
		blocked, err := s.isBlocked(r.Context(), viewerID, targetID)
		if err != nil {
			internalError(w, s.d, "get profile: check block", err)
			return
		}
		if blocked {
			shared.WriteError(w, http.StatusForbidden, codeBlocked, "This profile is not available.")
			return
		}
	}
	user, err := s.loadUser(r.Context(), targetID)
	if err != nil {
		internalError(w, s.d, "get profile: load user", err)
		return
	}
	if user.Visibility == "private" && viewerID != targetID {
		shared.WriteError(w, http.StatusForbidden, codeProfilePrivate, "This profile is private.")
		return
	}
	profile, err := s.loadProfile(r.Context(), targetID, viewerID)
	if err != nil {
		internalError(w, s.d, "get profile: load profile", err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, profile)
}

func (s *service) getMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := shared.UserID(r)
	if !ok {
		shared.WriteError(w, http.StatusUnauthorized, shared.CodeAuthRequired, "Sign in to continue.")
		return
	}
	user, err := s.loadUser(r.Context(), userID)
	if errors.Is(err, sql.ErrNoRows) {
		shared.WriteError(w, http.StatusUnauthorized, shared.CodeAuthRequired, "Sign in to continue.")
		return
	}
	if err != nil {
		internalError(w, s.d, "get me: load user", err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, user)
}

func (s *service) patchMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := shared.UserID(r)
	if !ok {
		shared.WriteError(w, http.StatusUnauthorized, shared.CodeAuthRequired, "Sign in to continue.")
		return
	}
	var in profilePatch
	if !shared.DecodeJSON(w, r, &in) {
		return
	}
	details := make(validationErrors)
	validateProfilePatch(in, details)
	if len(details) > 0 {
		writeValidation(w, details)
		return
	}

	var userSets, profileSets []string
	var userArgs, profileArgs []any
	if in.DisplayName != nil {
		userSets = append(userSets, "display_name = ?")
		userArgs = append(userArgs, strings.TrimSpace(*in.DisplayName))
	}
	if in.Discoverable != nil {
		userSets = append(userSets, "discoverable = ?")
		userArgs = append(userArgs, *in.Discoverable)
	}
	if in.Bio != nil {
		profileSets = append(profileSets, "bio = ?")
		profileArgs = append(profileArgs, nullableString(*in.Bio))
	}
	if in.Location != nil {
		profileSets = append(profileSets, "location = ?")
		profileArgs = append(profileArgs, nullableString(*in.Location))
	}
	if in.AvatarURL != nil {
		profileSets = append(profileSets, "avatar_url = ?")
		profileArgs = append(profileArgs, nullableString(*in.AvatarURL))
	}
	if in.BannerURL != nil {
		profileSets = append(profileSets, "banner_url = ?")
		profileArgs = append(profileArgs, nullableString(*in.BannerURL))
	}
	if in.Visibility != nil {
		profileSets = append(profileSets, "visibility = ?")
		profileArgs = append(profileArgs, *in.Visibility)
	}

	if len(userSets) > 0 || len(profileSets) > 0 {
		tx, err := s.d.DB.BeginTx(r.Context(), nil)
		if err != nil {
			internalError(w, s.d, "patch me: begin transaction", err)
			return
		}
		defer func() { _ = tx.Rollback() }()
		if len(userSets) > 0 {
			query := `UPDATE users SET ` + strings.Join(userSets, ", ") + `, updated_at = datetime('now') WHERE id = ?`
			if _, err := tx.ExecContext(r.Context(), query, append(userArgs, userID)...); err != nil {
				internalError(w, s.d, "patch me: update user", err)
				return
			}
		}
		if len(profileSets) > 0 {
			if _, err := tx.ExecContext(r.Context(),
				`INSERT OR IGNORE INTO user_profiles (user_id) VALUES (?)`, userID); err != nil {
				internalError(w, s.d, "patch me: ensure profile", err)
				return
			}
			query := `UPDATE user_profiles SET ` + strings.Join(profileSets, ", ") +
				`, updated_at = datetime('now') WHERE user_id = ?`
			if _, err := tx.ExecContext(r.Context(), query, append(profileArgs, userID)...); err != nil {
				internalError(w, s.d, "patch me: update profile", err)
				return
			}
		}
		if err := tx.Commit(); err != nil {
			internalError(w, s.d, "patch me: commit", err)
			return
		}
	}

	user, err := s.loadUser(r.Context(), userID)
	if err != nil {
		internalError(w, s.d, "patch me: reload user", err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, user)
}
