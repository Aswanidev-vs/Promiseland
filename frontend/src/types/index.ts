/**
 * Shared API DTOs. Keep these framework-agnostic: they describe the wire
 * format from the Go API (see docs/conventions.md), not UI view models.
 */

export interface User {
  id: string
  username: string
  displayName?: string | null
  avatarUrl?: string | null
  bio?: string | null
}

export interface Community {
  id: string
  name: string
  slug: string
  description?: string | null
  visibility?: 'public' | 'private'
  memberCount?: number
}

export interface Post {
  id: string
  authorId: string
  communityId?: string | null
  type?: 'text' | 'link' | 'image' | 'question' | 'poll' | 'announcement'
  title?: string | null
  body?: string | null
  spoilerFlag: boolean
  createdAt: string
}

/** Cursor pagination contract from docs/conventions.md. */
export interface Page<T> {
  items: T[]
  nextCursor: string | null
}

/** Every non-2xx response from the API uses this envelope. */
export interface ApiErrorBody {
  error: {
    code: string
    message: string
    details?: Record<string, unknown>
  }
}
