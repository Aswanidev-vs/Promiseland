package shape

// UserSummary is the exact R1 UserSummary wire shape.
type UserSummary struct {
	ID          int64   `json:"id"`
	Username    string  `json:"username"`
	DisplayName string  `json:"display_name"`
	AvatarURL   *string `json:"avatar_url"`
}

// Community is the exact R1 Community wire shape.
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

// Media is the exact R1 Media wire shape.
type Media struct {
	ID        int64  `json:"id"`
	URL       string `json:"url"`
	MimeType  string `json:"mime_type"`
	SizeBytes int64  `json:"size_bytes"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
}

// Anime is the exact R1 Anime wire shape.
type Anime struct {
	ID             int64   `json:"id"`
	ExternalID     string  `json:"external_id"`
	Title          string  `json:"title"`
	AlternateTitle *string `json:"alternate_title"`
	Format         string  `json:"format"`
}

// PollOption and Poll implement the optional Post.poll object.
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

// Post is the exact R1 Post wire shape.
type Post struct {
	ID              int64       `json:"id"`
	Author          UserSummary `json:"author"`
	Community       *Community  `json:"community"`
	Type            string      `json:"type"`
	Title           string      `json:"title"`
	Body            string      `json:"body"`
	LinkURL         string      `json:"link_url"`
	Media           []Media     `json:"media"`
	Anime           []Anime     `json:"anime"`
	Tags            []string    `json:"tags"`
	Spoiler         bool        `json:"spoiler"`
	SpoilerNote     string      `json:"spoiler_note"`
	Season          int         `json:"season"`
	Episode         int         `json:"episode"`
	CreatorCategory string      `json:"creator_category"`
	VoteScore       int         `json:"vote_score"`
	MyVote          int         `json:"my_vote"`
	CommentCount    int         `json:"comment_count"`
	Bookmarked      bool        `json:"bookmarked"`
	CreatedAt       string      `json:"created_at"`
	UpdatedAt       string      `json:"updated_at"`
	EditedAt        string      `json:"edited_at"`
	Poll            *Poll       `json:"poll,omitempty"`
}

func newPost() Post {
	return Post{
		Media: []Media{},
		Anime: []Anime{},
		Tags:  []string{},
	}
}
