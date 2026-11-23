// Package discovery implements R4 interest-based friend discovery and
// connection requests. All routes are authenticated; the router mounts this
// module on both the public and authenticated subrouters, and every handler
// resolves the viewer from the request context or session cookie.
package discovery

// UserSummary is the exact R1 UserSummary wire shape.
type UserSummary struct {
	ID          int64   `json:"id"`
	Username    string  `json:"username"`
	DisplayName string  `json:"display_name"`
	AvatarURL   *string `json:"avatar_url"`
}

// Interest is the exact R1 Interest wire shape.
type Interest struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
}

// Anime is the exact R1 Anime wire shape.
type Anime struct {
	ID             int64  `json:"id"`
	ExternalID     string `json:"external_id"`
	Title          string `json:"title"`
	AlternateTitle string `json:"alternate_title"`
	Format         string `json:"format"`
}

// AnimeInterest is the exact R1 AnimeInterest wire shape.
type AnimeInterest struct {
	Anime        Anime  `json:"anime"`
	InterestType string `json:"interest_type"`
	SortOrder    int    `json:"sort_order"`
}

// UserProfile is the R1 profile shape embedded in a discovery candidate.
// Email and location are deliberately absent: discovery never reveals private
// account or precise-location data. Both fields use omitempty and are left
// empty so they never appear on the wire.
type UserProfile struct {
	ID             int64           `json:"id"`
	Email          string          `json:"email,omitempty"`
	Username       string          `json:"username"`
	DisplayName    string          `json:"display_name"`
	AvatarURL      *string         `json:"avatar_url"`
	BannerURL      *string         `json:"banner_url"`
	Bio            *string         `json:"bio"`
	Location       *string         `json:"location,omitempty"`
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

// SharedInterests holds readable names for the overlap groups that have a
// dedicated wire array. Each array is capped at five entries.
type SharedInterests struct {
	Anime       []string `json:"anime"`
	Genres      []string `json:"genres"`
	Interests   []string `json:"interests"`
	Communities []string `json:"communities"`
}

// DiscoveryCandidate is the exact R4 wire shape from docs/conventions.md.
type DiscoveryCandidate struct {
	User             UserProfile     `json:"user"`
	Shared           SharedInterests `json:"shared"`
	SharedCount      int             `json:"shared_count"`
	Explanation      string          `json:"explanation"`
	ConnectionStatus string          `json:"connection_status"`
}

// Connection is the exact R4 wire shape from docs/conventions.md.
type Connection struct {
	ID        int64       `json:"id"`
	Requester UserSummary `json:"requester"`
	Addressee UserSummary `json:"addressee"`
	Message   string      `json:"message"`
	Status    string      `json:"status"`
	CreatedAt string      `json:"created_at"`
	UpdatedAt string      `json:"updated_at"`
}

// DiscoverySettings is the exact R4 wire shape from docs/conventions.md.
type DiscoverySettings struct {
	Discoverable bool `json:"discoverable"`
}

// page is the R1 Page<T> envelope. Unlike shared.Page, next_cursor is always
// present (empty string when there are no more pages) as the R4 contract asks.
type page[T any] struct {
	Items      []T    `json:"items"`
	NextCursor string `json:"next_cursor"`
}
