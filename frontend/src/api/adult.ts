import { request } from './client'
import type {
  AdultLike,
  AdultLikeStatus,
  AdultMatch,
  AdultProfile,
  AdultProfileInput,
  AdultReportInput,
  AdultStatus,
} from '@/types'

/**
 * Release 5 — gated, adults-only surface. Every path lives in its own
 * `/adult/*` namespace so this area remains unlinked and unreachable from
 * ordinary discovery when the feature flag is off.
 *
 * When the gate is off the backend may return 404 not_found for these paths;
 * callers should treat that as "not available", not as a network failure.
 */

/** GET /adult/status — safe; returns the disabled state without private data. */
export async function getAdultStatus(): Promise<AdultStatus> {
  return request<AdultStatus>('/adult/status')
}

/** GET /adult/profile — requires eligibility and an existing opt-in. */
export async function getAdultProfile(): Promise<AdultProfile> {
  return request<AdultProfile>('/adult/profile')
}

/** PUT /adult/profile — create or replace the opt-in profile. */
export async function saveAdultProfile(input: AdultProfileInput): Promise<AdultProfile> {
  return request<AdultProfile>('/adult/profile', {
    method: 'PUT',
    body: JSON.stringify({
      display_name: input.displayName,
      intent: input.intent,
      bio: input.bio,
      boundaries: input.boundaries,
    }),
  })
}

/** POST /adult/like/{userId} — a mutual like creates the match server-side. */
export async function likeAdult(userId: number, status: AdultLikeStatus): Promise<AdultLike> {
  return request<AdultLike>(`/adult/like/${userId}`, {
    method: 'POST',
    body: JSON.stringify({ status }),
  })
}

/** GET /adult/matches — only matches the signed-in user is part of. */
export async function listAdultMatches(): Promise<AdultMatch[]> {
  return request<AdultMatch[]>('/adult/matches')
}

/** POST /adult/matches/{id}/unmatch — available to either user. */
export async function unmatch(id: number): Promise<void> {
  await request<void>(`/adult/matches/${id}/unmatch`, { method: 'POST' })
}

/** POST /adult/reports — writes a high-priority adult safety report. */
export async function reportAdult(input: AdultReportInput): Promise<void> {
  await request<void>('/adult/reports', {
    method: 'POST',
    body: JSON.stringify({
      target_type: input.targetType,
      target_id: input.targetId,
      reason: input.reason,
      details: input.details,
    }),
  })
}
