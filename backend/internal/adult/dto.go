package adult

// DTOs in this file are the exact R5 wire shapes from docs/conventions.md.
// They intentionally contain no location, phone, contact, birthday or
// compatibility-score fields. Nothing here presents a scientific score.

// UserSummary is the exact R1 UserSummary shape used inside a match.
type UserSummary struct {
	ID          int64   `json:"id"`
	Username    string  `json:"username"`
	DisplayName string  `json:"display_name"`
	AvatarURL   *string `json:"avatar_url"`
}

// AdultStatus is the safe status shape. When the feature flag is off it is
// returned with HTTP 404 and carries no eligibility information.
type AdultStatus struct {
	Enabled      bool   `json:"enabled"`
	AgeAssurance string `json:"ageAssurance"`
	Eligible     bool   `json:"eligible"`
	Message      string `json:"message"`
}

// AdultProfile is an opt-in adults-only profile. It is private by design:
// there is no listing endpoint and it is never shown to non-matched users.
type AdultProfile struct {
	UserID      int64  `json:"user_id"`
	DisplayName string `json:"display_name"`
	Intent      string `json:"intent"`
	Bio         string `json:"bio"`
	Boundaries  string `json:"boundaries"`
	Visibility  string `json:"visibility"`
	CreatedAt   string `json:"created_at"`
}

// AdultLike is the result of a like/pass action.
type AdultLike struct {
	ActorID   int64  `json:"actor_id"`
	TargetID  int64  `json:"target_id"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

// AdultMatch is a mutual-like match. ConversationID is resolved from the
// existing direct-conversation table by stable pair key, or nil when the chat
// schema is unavailable.
type AdultMatch struct {
	ID             int64         `json:"id"`
	Users          []UserSummary `json:"users"`
	CreatedAt      string        `json:"created_at"`
	ConversationID *int64        `json:"conversation_id"`
	UnmatchedAt    *string       `json:"unmatched_at"`
}

// page is the standard Page<T> envelope with next_cursor always present.
type page[T any] struct {
	Items      []T    `json:"items"`
	NextCursor string `json:"next_cursor"`
}

// reportResponse mirrors the existing R1 Report wire shape. The adult module
// writes to the same reports table used by moderation.
type reportResponse struct {
	ID         int64   `json:"id"`
	ReporterID int64   `json:"reporter_id"`
	TargetType string  `json:"target_type"`
	TargetID   int64   `json:"target_id"`
	Reason     string  `json:"reason"`
	Details    *string `json:"details"`
	Status     string  `json:"status"`
	CreatedAt  string  `json:"created_at"`
}

// verificationResponse is only produced by the development-only verify
// endpoint. It never exposes a birthday or other personal data.
type verificationResponse struct {
	UserID     int64   `json:"user_id"`
	Status     string  `json:"status"`
	Method     string  `json:"method"`
	VerifiedAt *string `json:"verified_at"`
	ExpiresAt  *string `json:"expires_at"`
}
