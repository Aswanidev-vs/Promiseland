// Package communities implements community discovery, membership and settings.
// It also serves the community-scoped post list; the feed itself is owned by
// the feed module.
package communities

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
)

// UserSummary is the exact R1 public user shape.
type UserSummary struct {
	ID          int64   `json:"id"`
	Username    string  `json:"username"`
	DisplayName string  `json:"display_name"`
	AvatarURL   *string `json:"avatar_url"`
}

// Community is the exact R1 community shape. Role is null for non-members.
type Community struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	Description string  `json:"description"`
	IconURL     *string `json:"icon_url"`
	BannerURL   *string `json:"banner_url"`
	Visibility  string  `json:"visibility"`
	MemberCount int     `json:"member_count"`
	PostCount   int     `json:"post_count"`
	Role        *string `json:"role"`
	IsMember    bool    `json:"is_member"`
	CreatedAt   string  `json:"created_at"`
}

// CommunityRule is the exact R1 rule shape.
type CommunityRule struct {
	ID          int64  `json:"id"`
	CommunityID int64  `json:"community_id"`
	Title       string `json:"title"`
	Body        string `json:"body"`
	SortOrder   int    `json:"sort_order"`
}

// Media, Anime, PollOption, Poll and Post mirror the R1 post shape. They are
// declared here (rather than imported from posts) because domain packages must
// not import each other; only RoleFor is the sanctioned exception, used by
// posts. Keeping the DTOs local keeps the dependency direction one-way.
type Media struct {
	ID        int64  `json:"id"`
	URL       string `json:"url"`
	MimeType  string `json:"mime_type"`
	SizeBytes int64  `json:"size_bytes"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
}

type Anime struct {
	ID             int64   `json:"id"`
	ExternalID     string  `json:"external_id"`
	Title          string  `json:"title"`
	AlternateTitle *string `json:"alternate_title"`
	Format         string  `json:"format"`
}

type PollOption struct {
	ID    int64  `json:"id"`
	Text  string `json:"text"`
	Votes int    `json:"votes"`
}

type Poll struct {
	Question   string       `json:"question"`
	Options    []PollOption `json:"options"`
	TotalVotes int          `json:"total_votes"`
	MyVote     *int64       `json:"my_vote"`
}

type Post struct {
	ID              int64       `json:"id"`
	Author          UserSummary `json:"author"`
	Community       *Community  `json:"community"`
	Type            string      `json:"type"`
	Title           string      `json:"title"`
	Body            string      `json:"body"`
	LinkURL         string      `json:"link_url"`
	CreatorCategory string      `json:"creator_category"`
	Media           []Media     `json:"media"`
	Anime           []Anime     `json:"anime"`
	Tags            []string    `json:"tags"`
	Spoiler         bool        `json:"spoiler"`
	SpoilerNote     string      `json:"spoiler_note"`
	Season          int         `json:"season"`
	Episode         int         `json:"episode"`
	VoteScore       int         `json:"vote_score"`
	MyVote          int         `json:"my_vote"`
	CommentCount    int         `json:"comment_count"`
	Bookmarked      bool        `json:"bookmarked"`
	CreatedAt       string      `json:"created_at"`
	UpdatedAt       string      `json:"updated_at"`
	EditedAt        string      `json:"edited_at"`
	Poll            *Poll       `json:"poll,omitempty"`
}

// page serialisation always includes next_cursor (empty string when done).
type page[T any] struct {
	Items      []T    `json:"items"`
	NextCursor string `json:"next_cursor"`
}

type communityRow struct {
	Community
	OwnerID        int64
	PostingEnabled bool
}

// viewerID accepts both routers with and without OptionalAuth: protected
// routes already carry the ID in context, public routes fall back to the
// session cookie so privacy filters still see signed-in members.
func viewerID(r *http.Request, d *shared.Deps) int64 {
	if id, ok := shared.UserID(r); ok {
		return id
	}
	if d == nil {
		return 0
	}
	id, _ := d.SessionUserID(r)
	return id
}

// columnExists is used for defensive queries against tables owned by other
// agents (users.username, anime_titles.format, blocks, ...). Table names are
// always compile-time constants here, never user input.
func columnExists(d *shared.Deps, table, column string) bool {
	if d == nil || d.DB == nil {
		return false
	}
	var n int
	err := d.DB.QueryRow(fmt.Sprintf(
		`SELECT COUNT(*) FROM pragma_table_info('%s') WHERE name = ?`, table,
	), column).Scan(&n)
	return err == nil && n > 0
}

func tableExists(d *shared.Deps, name string) bool {
	if d == nil || d.DB == nil {
		return false
	}
	var n int
	err := d.DB.QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, name,
	).Scan(&n)
	return err == nil && n > 0
}

func nullString(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	s := v.String
	return &s
}

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}
