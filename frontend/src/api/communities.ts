import { buildQuery, request } from './client'
import type {
  Community,
  CommunityRule,
  CreateCommunityInput,
  ListCommunitiesParams,
  Page,
  PaginationParams,
  Post,
  UserSummary,
} from '@/types'

export async function listCommunities(
  p: ListCommunitiesParams = {},
): Promise<Page<Community>> {
  return request<Page<Community>>(`/communities${buildQuery({ ...p })}`)
}

export async function getCommunity(slug: string): Promise<Community> {
  return request<Community>(`/communities/${encodeURIComponent(slug)}`)
}

export async function createCommunity(input: CreateCommunityInput): Promise<Community> {
  return request<Community>('/communities', { method: 'POST', body: JSON.stringify(input) })
}

export async function updateCommunity(
  slug: string,
  patch: Partial<Community>,
): Promise<Community> {
  return request<Community>(`/communities/${encodeURIComponent(slug)}`, {
    method: 'PATCH',
    body: JSON.stringify(patch),
  })
}

export async function joinCommunity(slug: string): Promise<void> {
  // The API joins at POST /join and leaves at DELETE /membership.
  await request<void>(`/communities/${encodeURIComponent(slug)}/join`, { method: 'POST' })
}

export async function leaveCommunity(slug: string): Promise<void> {
  await request<void>(`/communities/${encodeURIComponent(slug)}/membership`, { method: 'DELETE' })
}

export async function listRules(slug: string): Promise<CommunityRule[]> {
  return request<CommunityRule[]>(`/communities/${encodeURIComponent(slug)}/rules`)
}

export async function listMembers(
  slug: string,
  p: PaginationParams = {},
): Promise<Page<UserSummary>> {
  return request<Page<UserSummary>>(
    `/communities/${encodeURIComponent(slug)}/members${buildQuery({ ...p })}`,
  )
}

export async function listCommunityPosts(
  slug: string,
  p: PaginationParams = {},
): Promise<Page<Post>> {
  return request<Page<Post>>(
    `/communities/${encodeURIComponent(slug)}/posts${buildQuery({ ...p })}`,
  )
}

export async function addRule(
  slug: string,
  rule: { title: string; body: string },
): Promise<CommunityRule> {
  return request<CommunityRule>(`/communities/${encodeURIComponent(slug)}/rules`, {
    method: 'POST',
    body: JSON.stringify(rule),
  })
}

export async function addModerator(slug: string, userId: number): Promise<void> {
  await request<void>(`/communities/${encodeURIComponent(slug)}/moderators`, {
    method: 'POST',
    body: JSON.stringify({ user_id: userId }),
  })
}
