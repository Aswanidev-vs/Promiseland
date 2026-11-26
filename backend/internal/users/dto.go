package users

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
)

// User and UserProfile are the exact R1 wire shapes from docs/conventions.md.
type User struct {
	ID           int64   `json:"id"`
	Email        string  `json:"email"`
	Username     string  `json:"username"`
	DisplayName  string  `json:"display_name"`
	AvatarURL    *string `json:"avatar_url"`
	BannerURL    *string `json:"banner_url"`
	Bio          *string `json:"bio"`
	Location     *string `json:"location"`
	Visibility   string  `json:"visibility"`
	Discoverable bool    `json:"discoverable"`
	CreatedAt    string  `json:"created_at"`
}

type UserProfile struct {
	ID             int64           `json:"id"`
	Email          string          `json:"email,omitempty"` // self only
	Username       string          `json:"username"`
	DisplayName    string          `json:"display_name"`
	AvatarURL      *string         `json:"avatar_url"`
	BannerURL      *string         `json:"banner_url"`
	Bio            *string         `json:"bio"`
	Location       *string         `json:"location"`
	Visibility     string          `json:"visibility"`
	Discoverable   bool            `json:"discoverable"`
	CreatedAt      string          `json:"created_at"`
	Interests      []Interest      `json:"interests"`
	AnimeInterests []AnimeInterest `json:"anime_interests"`
	FollowerCount  int64           `json:"follower_count"`
	FollowingCount int64           `json:"following_count"`
	PostCount      int64           `json:"post_count"`
	IsFollowing    bool            `json:"is_following"`
	IsBlocked      bool            `json:"is_blocked"`
}

type Interest struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
}

type Anime struct {
	ID             int64  `json:"id"`
	ExternalID     string `json:"external_id"`
	Title          string `json:"title"`
	AlternateTitle string `json:"alternate_title"`
	Format         string `json:"format"`
}

type AnimeInterest struct {
	Anime        Anime  `json:"anime"`
	InterestType string `json:"interest_type"`
	SortOrder    int    `json:"sort_order"`
}

type animeInterestInput struct {
	AnimeID      int64  `json:"animeId"`
	InterestType string `json:"interestType"`
}

// UnmarshalJSON accepts the camelCase keys from the R1 conventions and the
// snake_case keys the frontend client sends (src/api/users.ts). Without this
// the interest list decoded to zero values and nothing was saved.
func (a *animeInterestInput) UnmarshalJSON(b []byte) error {
	type plain animeInterestInput
	var raw struct {
		plain
		AnimeIDSnake      int64  `json:"anime_id"`
		InterestTypeSnake string `json:"interest_type"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	*a = animeInterestInput(raw.plain)
	if a.AnimeID == 0 {
		a.AnimeID = raw.AnimeIDSnake
	}
	if a.InterestType == "" {
		a.InterestType = raw.InterestTypeSnake
	}
	return nil
}

type profilePatch struct {
	DisplayName  *string `json:"display_name"`
	Bio          *string `json:"bio"`
	Location     *string `json:"location"`
	AvatarURL    *string `json:"avatar_url"`
	BannerURL    *string `json:"banner_url"`
	Visibility   *string `json:"visibility"`
	Discoverable *bool   `json:"discoverable"`
}

const (
	codeValidation     = "validation_error"
	codeProfilePrivate = "profile_private"
	codeBlocked        = "blocked"
	codeInvalidID      = "invalid_id"
	codeInvalidCursor  = "invalid_cursor"
)

type validationErrors map[string]string

func (v validationErrors) add(field, message string) {
	if _, exists := v[field]; !exists {
		v[field] = message
	}
}

func writeValidation(w http.ResponseWriter, details validationErrors) {
	shared.WriteErrorDetails(w, http.StatusBadRequest, codeValidation,
		"Please correct the highlighted fields.", map[string]string(details))
}

func stringPtr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	value := ns.String
	return &value
}

func nullableString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func validateProfilePatch(in profilePatch, details validationErrors) {
	if in.DisplayName != nil {
		name := strings.TrimSpace(*in.DisplayName)
		switch {
		case name == "":
			details.add("display_name", "Display name cannot be empty.")
		case utf8.RuneCountInString(name) > 50:
			details.add("display_name", "Display name must be at most 50 characters.")
		}
	}
	if in.Bio != nil && utf8.RuneCountInString(*in.Bio) > 500 {
		details.add("bio", "Bio must be at most 500 characters.")
	}
	if in.Location != nil && utf8.RuneCountInString(*in.Location) > 100 {
		details.add("location", "Location must be at most 100 characters.")
	}
	if in.AvatarURL != nil && !shared.ValidProfileImage(*in.AvatarURL) {
		details.add("avatar_url", "Upload an image or enter an http(s) URL.")
	}
	if in.BannerURL != nil && !shared.ValidProfileImage(*in.BannerURL) {
		details.add("banner_url", "Upload an image or enter an http(s) URL.")
	}
	if in.Visibility != nil && *in.Visibility != "public" && *in.Visibility != "private" {
		details.add("visibility", "Visibility must be 'public' or 'private'.")
	}
}
