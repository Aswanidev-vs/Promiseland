package shared

// User is the public representation of an account. Never add password hashes,
// email verification tokens or other security fields to this struct.
type User struct {
	ID          int64  `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url,omitempty"`
	CreatedAt   string `json:"created_at"`
}

// Session mirrors the sessions table. TokenHash is deliberately not serialised.
type Session struct {
	TokenHash string `json:"-"`
	UserID    int64  `json:"user_id"`
	ExpiresAt string `json:"expires_at"`
	CreatedAt string `json:"created_at"`
}

// Page is the standard paginated response envelope. Use NewPage to ensure
// Items serialises as [] rather than null.
type Page[T any] struct {
	Items      []T    `json:"items"`
	NextCursor string `json:"next_cursor,omitempty"`
}

// NewPage builds a page with a non-nil Items slice.
func NewPage[T any](items []T) Page[T] {
	if items == nil {
		items = []T{}
	}
	return Page[T]{Items: items}
}
