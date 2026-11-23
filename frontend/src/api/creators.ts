import { buildQuery, request } from './client'
import type { Creator, ListCreatorsParams, Page } from '@/types'

/** Filter order follows the R2 endpoint contract: category,q,limit,cursor. */
export async function listCreators(p: ListCreatorsParams = {}): Promise<Page<Creator>> {
  return request<Page<Creator>>(
    `/creators${buildQuery({
      category: p.category,
      q: p.q,
      limit: p.limit,
      cursor: p.cursor,
    })}`,
  )
}

export async function getCreator(username: string): Promise<Creator> {
  return request<Creator>(`/creators/${encodeURIComponent(username)}`)
}

/** Featured posts are toggled on the current user via /users/me/featured/{postId}. */
export async function featurePost(postId: number): Promise<void> {
  await request<void>(`/users/me/featured/${postId}`, { method: 'POST' })
}

export async function unfeaturePost(postId: number): Promise<void> {
  await request<void>(`/users/me/featured/${postId}`, { method: 'DELETE' })
}
