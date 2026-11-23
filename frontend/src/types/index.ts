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
}

export interface FeedParams {
  scope?: 'for-you' | 'following'
  community?: string
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

/** Query parameter values accepted by buildQuery. */
export type QueryValue = string | number | boolean | null | undefined
