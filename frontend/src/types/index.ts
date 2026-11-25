/**
 * Shared API DTOs. Framework-agnostic wire types for the Go API.
 *
 * Field names and shapes follow docs/conventions.md §"Backend JSON shapes"
 * exactly (snake_case). A few optional camelCase aliases are kept from R0 so
 * existing imports keep compiling; prefer the snake_case contract fields.
 */

export type Vote = -1 | 0 | 1
export type Visibility = 'public' | 'private'
export type CommunityRole = 'owner' | 'moderator' | 'member' | null
export type PostType = 'text' | 'link' | 'image' | 'question' | 'poll' | 'announcement'
export type SearchType = 'users' | 'communities' | 'posts'

export interface UserSummary {
  id: number
  username: string
  display_name: string
  avatar_url: string | null
}

export interface User {
  id: number
  /** Only present on self responses. */
  email?: string | null
  username: string
  display_name: string
  avatar_url: string | null
  banner_url: string | null
  bio: string | null
  location: string | null
  visibility: string
  discoverable: boolean
  created_at: string
  /** Platform staff grant: shows the admin console. */
  is_staff?: boolean
  /** @deprecated R0 alias — use display_name. */
  displayName?: string | null
  /** @deprecated R0 alias — use avatar_url. */
  avatarUrl?: string | null
}

export interface UserProfile extends User {
  interests: Interest[]
  anime_interests: AnimeInterest[]
  follower_count: number
  following_count: number
  post_count: number
  is_following: boolean
  is_blocked: boolean
}

export interface Interest {
  id: number
  name: string
  category: string
}

export interface Anime {
  id: number
  external_id: string
  title: string
  alternate_title: string | null
  format: string
}

export interface AnimeInterest {
  anime: Anime
  interest_type: string
  sort_order: number
}

export interface Genre {
  id: number
  name: string
}

export interface Community {
  id: number
  name: string
  slug: string
  description: string | null
  icon_url: string | null
  banner_url: string | null
  visibility: Visibility
  member_count: number
  post_count: number
  role: CommunityRole
  is_member: boolean
  created_at: string
}

export interface CommunityRule {
  id: number
  community_id: number
  title: string
  body: string
  sort_order: number
}

export interface Media {
  id: number
  url: string
  mime_type: string
  size_bytes: number
  width: number
  height: number
}

export interface PollOption {
  id: number
  text: string
  votes: number
}

export interface Poll {
  question: string
  options: PollOption[]
  total_votes: number
  my_vote: number | null
}

export interface Post {
  id: number
  author: UserSummary
  community: Community | null
  type: PostType
  title: string
  body: string
  link_url: string | null
  media: Media[]
  anime: Anime[]
  tags: string[]
  spoiler: boolean
  spoiler_note: string | null
  season: string | null
  episode: number | null
  vote_score: number
  my_vote: Vote
  comment_count: number
  bookmarked: boolean
  /** R2: creator portfolio category; '' when the post is not creator-tagged. */
  creator_category: string
  created_at: string
  updated_at: string
  edited_at: string | null
  poll?: Poll
  /** @deprecated R0 alias — use spoiler. */
  spoilerFlag?: boolean
}

export interface Comment {
  id: number
  post_id: number
  author: UserSummary
  parent_id: number | null
  body: string
  created_at: string
  edited_at: string | null
  deleted: boolean
}

export interface Notification {
  id: number
  type: string
  actor: UserSummary | null
  target_type: string
  target_id: number
  read_at: string | null
  created_at: string
  data: Record<string, unknown>
}

export interface Report {
  id: number
  reporter_id: number
  target_type: string
  target_id: number
  reason: string
  details: string | null
  status: string
  created_at: string
}

// ── Release 2 wire DTOs (snake_case — docs/conventions.md §R2) ──

export type EventType =
  | 'convention'
  | 'meetup'
  | 'screening'
  | 'photoshoot'
  | 'club'
  | 'online'
  | 'competition'
  | 'other'

export type RsvpStatus = 'going' | 'maybe'
export type TeamUpStatus = 'open' | 'filled' | 'closed'
export type ApplicationStatus = 'pending' | 'accepted' | 'rejected'
export type CreatorCategory =
  | 'cosplay'
  | 'crossplay'
  | 'fan_art'
  | 'photography'
  | 'edit'
  | 'collection'
  | 'other'

export interface Event {
  id: number
  organizer: UserSummary
  title: string
  description: string
  event_type: string
  starts_at: string
  ends_at: string
  timezone: string
  venue_text: string
  city: string
  country: string
  is_online: boolean
  capacity: number
  rsvp_count: number
  rsvp_status: string | null
  cover_url: string | null
  status: string
  created_at: string
}

export interface EventUpdate {
  id: number
  event_id: number
  author: UserSummary
  body: string
  created_at: string
}

export interface EventComment {
  id: number
  event_id: number
  author: UserSummary
  parent_id: number | null
  body: string
  created_at: string
  deleted: boolean
}

export interface TeamUp {
  id: number
  owner: UserSummary
  title: string
  description: string
  event: Event | null
  categories: string[]
  status: TeamUpStatus
  applicant_count: number
  my_application: string | null
  created_at: string
}

export interface TeamUpApplication {
  id: number
  team_up_id: number
  applicant: UserSummary
  message: string
  status: ApplicationStatus
  created_at: string
}

export interface Creator {
  user: UserSummary
  categories: string[]
  featured_posts: Post[]
  post_count: number
  follower_count: number
}

/** Cursor pagination contract. next_cursor is '' when there are no more pages. */
export interface Page<T> {
  items: T[]
  next_cursor: string
  /** @deprecated R0 alias — use next_cursor. */
  nextCursor?: string | null
}

/** Every non-2xx response from the API uses this envelope. */
export interface ApiErrorBody {
  error: {
    code: string
    message: string
    details?: Record<string, unknown>
  }
}

// ── Request inputs (camelCase on the client, mapped to wire shape in api/) ──

export interface RegisterInput {
  email: string
  password: string
  username: string
  displayName: string
}

export interface LoginInput {
  email: string
  password: string
}

export interface AnimeInterestInput {
  animeId: number
  interestType: string
}

export interface CreateCommunityInput {
  name: string
  slug: string
  description: string
  visibility: string
  rules?: string[]
  /** URL or uploaded /media path for the community icon. */
  icon_url?: string
  banner_url?: string
}

export interface FeedParams {
  scope?: 'for-you' | 'following'
  community?: string
  /** R2: additive filter, sent as the `creatorCategory` query parameter. */
  creatorCategory?: string
  limit?: number
  cursor?: string
}

export interface PaginationParams {
  limit?: number
  cursor?: string
}

export interface ListCommunitiesParams extends PaginationParams {
  q?: string
}

export interface CreatePostInput {
  communitySlug?: string
  type: PostType
  title: string
  body: string
  linkUrl?: string
  mediaIds?: number[]
  animeIds?: number[]
  tags?: string[]
  spoiler?: boolean
  spoilerNote?: string
  season?: string
  episode?: number
  /** R2: creator portfolio category, sent as `creator_category` in the body. */
  creatorCategory?: string
}

export interface SearchParams extends PaginationParams {
  q: string
  type?: SearchType
}

export interface SearchResults {
  users: UserSummary[]
  communities: Community[]
  posts: Post[]
}

export interface UpdateSettingsInput {
  visibility?: string
  discoverable?: boolean
}

export interface ReportInput {
  targetType: string
  targetId: number
  reason: string
  details?: string
}

export interface ResolveReportInput {
  action: string
  reason: string
}

// ── Release 2 request inputs (camelCase client → snake_case wire in api/) ──

export interface ListEventsParams extends PaginationParams {
  q?: string
  type?: EventType | string
  from?: string
  to?: string
  city?: string
  country?: string
  online?: boolean
}

export interface CreateEventInput {
  title: string
  description: string
  eventType: EventType | string
  startsAt: string
  endsAt: string
  timezone: string
  venueText: string
  city: string
  country: string
  isOnline: boolean
  capacity?: number
  coverMediaId?: number
}

/** Team-up browsing currently exposes pagination only. */
export type ListTeamUpsParams = PaginationParams & {
  status?: string
  category?: string
  q?: string
}

export interface CreateTeamUpInput {
  title: string
  description: string
  eventId?: number
  categories: string[]
}

export interface ListCreatorsParams extends PaginationParams {
  category?: CreatorCategory | string
  q?: string
}

/** Query parameter values accepted by buildQuery. */
export type QueryValue = string | number | boolean | null | undefined

// ── Release 3 wire DTOs (snake_case — docs/conventions.md §R3) ──

export type ChatRole = 'owner' | 'moderator' | 'member' | null
export type ChatChannelType = 'text'
export type ConversationType = 'direct' | 'group'
/** Chat reactions are short codes rendered with the icon set, never emoji. */
export type ChatReactionCode = 'like' | 'heart' | 'star' | 'laugh'

export interface ChatReaction {
  reaction: string
  count: number
  me: boolean
}

export interface ChatMessage {
  id: number
  channel_id: number | null
  conversation_id: number | null
  author: UserSummary
  parent_id: number | null
  body: string
  reactions: ChatReaction[]
  created_at: string
  edited_at: string | null
  deleted: boolean
}

export interface ChatChannel {
  id: number
  server_id: number
  name: string
  description: string | null
  type: ChatChannelType
  last_message: ChatMessage | null
  unread_count: number
}

export interface ChatServer {
  id: number
  owner: UserSummary
  name: string
  slug: string
  description: string | null
  icon_url: string | null
  member_count: number
  channels: ChatChannel[]
  role: ChatRole
  created_at: string
}

export interface Conversation {
  id: number
  type: ConversationType
  name: string
  members: UserSummary[]
  last_message: ChatMessage | null
  unread_count: number
  created_at: string
}

export type ChatEventType =
  | 'message.new'
  | 'message.deleted'
  | 'reaction.updated'
  | 'typing'
  | 'presence'

/**
 * Server-pushed realtime frame from `WS /ws` (internal/realtime.Event).
 * `payload` is deliberately loose because the Go hub types it as `any`.
 */
export interface ChatEvent {
  type: string
  payload: Record<string, unknown>
  channel_id?: number | null
  conversation_id?: number | null
  /** Used by presence/typing; empty means the frame is scoped by channel/conversation. */
  user_ids?: number[]
}

/** User id → presence status string, as returned by GET /chat/presence. */
export type PresenceMap = Record<number, string>

// ── Release 3 request inputs (camelCase client → snake_case wire in api/) ──

/** A chat target: exactly one of the two ids is normally supplied. */
export interface ChatScope {
  channelId?: number
  conversationId?: number
}

export type ChatMessageScope = ChatScope

export interface ListMessagesParams extends ChatScope, PaginationParams {}

export interface CreateServerInput {
  name: string
  description: string
}

export interface CreateChannelInput {
  name: string
  description: string
}

export interface CreateConversationInput {
  memberIds: number[]
  name?: string
}

export interface ReportMessageInput {
  reason: string
  details?: string
}

// ── Release 4 wire DTOs (snake_case — docs/conventions.md §R4) ──

/** The overlap that explains why a candidate was surfaced. No compatibility score. */
export interface SharedInterests {
  anime: string[]
  genres: string[]
  interests: string[]
  communities: string[]
}

export type ConnectionStatus = 'pending' | 'accepted' | 'rejected' | 'ignored'
/** Statuses a user may set when reviewing an incoming request. */
export type ConnectionReviewStatus = Exclude<ConnectionStatus, 'pending'>
export type ConnectionDirection = 'incoming' | 'outgoing' | 'all'
/** Status stamped on a discovery candidate relative to the current user. */
export type CandidateConnectionStatus = 'none' | 'pending_in' | 'pending_out' | 'accepted'

export interface DiscoveryCandidate {
  user: UserProfile
  shared: SharedInterests
  shared_count: number
  explanation: string
  connection_status: CandidateConnectionStatus
}

export interface Connection {
  id: number
  requester: UserSummary
  addressee: UserSummary
  message: string
  status: ConnectionStatus
  created_at: string
  updated_at: string
}

export interface DiscoverySettings {
  discoverable: boolean
}

// ── Release 4 request inputs (camelCase client → snake_case wire in api/) ──

export type ListFriendsParams = PaginationParams

export interface UpdateDiscoverySettingsInput {
  discoverable: boolean
}

export interface SendConnectionInput {
  userId: number
  message?: string
}

export interface ListConnectionsParams extends PaginationParams {
  status?: ConnectionStatus
  direction?: ConnectionDirection
}

// ── Release 5 wire DTOs (gated — docs/conventions.md §R5) ──

/** Age-assurance backend in use. The default implementation denies everyone. */
export type AdultAgeAssurance = 'none' | 'development_stub'
/** Separate friendship/dating intent; never inferred from ordinary activity. */
export type AdultIntent = 'friendship' | 'dating' | 'both'
export type AdultLikeStatus = 'like' | 'pass'

/**
 * Safe, gate-check response. `enabled` reflects FEATURE_ADULT_DATING;
 * `eligible` reflects the age-assurance check. Neither is implied by the other.
 */
export interface AdultStatus {
  enabled: boolean
  ageAssurance: AdultAgeAssurance
  eligible: boolean
  message: string
}

/** Opt-in profile. Visibility is fixed to `opt_in` by the server. */
export interface AdultProfile {
  user_id: number
  display_name: string
  intent: AdultIntent
  bio: string
  boundaries: string
  visibility: 'opt_in'
  created_at: string
}

export interface AdultLike {
  actor_id: number
  target_id: number
  status: AdultLikeStatus
  created_at: string
}

/** A match exists only after two positive actions. */
export interface AdultMatch {
  id: number
  users: [UserSummary, UserSummary]
  created_at: string
  conversation_id: number | null
  unmatched_at: string | null
}

// ── Release 5 request inputs (camelCase client → snake_case wire in api/) ──

export interface AdultProfileInput {
  displayName: string
  intent: AdultIntent
  bio: string
  boundaries: string
}

export interface AdultReportInput {
  targetType: string
  targetId: number
  reason: string
  details?: string
}
