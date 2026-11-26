package users

import (
	"net/http"
	"testing"
)

// The frontend client sends snake_case keys (src/api/users.ts). Before the
// alias in animeInterestInput.UnmarshalJSON every item decoded to zero values,
// so the request returned 200 and saved nothing.
func TestAnimeInterestsAcceptFrontendSnakeCaseKeys(t *testing.T) {
	env := newTestEnv(t)
	alice := seedUser(t, env, "alice@example.com", "alice", "Alice", "public")
	token := seedSession(t, env, alice)

	var narutoID int64
	if err := env.d.DB.QueryRow(`SELECT id FROM anime_titles WHERE title = 'Naruto'`).Scan(&narutoID); err != nil {
		t.Fatalf("find seeded Naruto: %v", err)
	}

	status, body := env.do(t, token, http.MethodPut, "/api/v1/users/me/anime-interests",
		[]map[string]any{{"anime_id": narutoID, "interest_type": "watching"}})
	if status != http.StatusOK {
		t.Fatalf("snake_case anime interests = %d (%s)", status, body)
	}
	profile := decodeJSON[UserProfile](t, body)
	if len(profile.AnimeInterests) != 1 {
		t.Fatalf("anime interests dropped: %+v", profile.AnimeInterests)
	}
	if got := profile.AnimeInterests[0]; got.Anime.Title != "Naruto" || got.InterestType != "watching" {
		t.Fatalf("anime interest = %+v", got)
	}
}
