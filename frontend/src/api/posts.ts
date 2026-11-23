import { buildQuery, request, requestForm } from './client'
import type {
  Comment,
  CreatePostInput,
  FeedParams,
  Media,
  Page,
  PaginationParams,
  Post,
  Vote,
} from '@/types'

/** No upload endpoint is listed in the v1 surface; media is its own resource. */
export const MEDIA_UPLOAD_PATH = '/media'

export async function getFeed(p: FeedParams = {}): Promise<Page<Post>> {
  return request<Page<Post>>(
    `/posts${buildQuery({
      scope: p.scope,
      community: p.community,
      creatorCategory: p.creatorCategory,
      limit: p.limit,
      cursor: p.cursor,
    })}`,
  )
}

export async function getPost(id: number): Promise<Post> {
  return request<Post>(`/posts/${id}`)
}

export async function createPost(input: CreatePostInput): Promise<Post> {
  return request<Post>('/posts', {
    method: 'POST',
    body: JSON.stringify({
      community_slug: input.communitySlug,
      type: input.type,
      title: input.title,
      body: input.body,
      link_url: input.linkUrl,
      media_ids: input.mediaIds,
      anime_ids: input.animeIds,
      tags: input.tags,
      spoiler: input.spoiler,
      spoiler_note: input.spoilerNote,
      season: input.season,
      episode: input.episode,
      creator_category: input.creatorCategory,
    }),
  })
}

export async function updatePost(id: number, patch: Partial<Post>): Promise<Post> {
  return request<Post>(`/posts/${id}`, { method: 'PATCH', body: JSON.stringify(patch) })
}

export async function deletePost(id: number): Promise<void> {
  await request<void>(`/posts/${id}`, { method: 'DELETE' })
}

export async function listComments(
  id: number,
  p: PaginationParams = {},
): Promise<Page<Comment>> {
  return request<Page<Comment>>(`/posts/${id}/comments${buildQuery({ ...p })}`)
}

export async function addComment(id: number, body: string, parentId?: number): Promise<Comment> {
  return request<Comment>(`/posts/${id}/comments`, {
    method: 'POST',
    body: JSON.stringify({ body, parent_id: parentId }),
  })
}

/**
 * One vote per user per post. Removing a vote (0) uses DELETE; up/down votes
 * use the reactions endpoint from the v1 surface with the signed vote value.
 */
export async function votePost(id: number, vote: Vote): Promise<Post> {
  if (vote === 0) {
    await request<void>(`/posts/${id}/reactions`, { method: 'DELETE' })
    return getPost(id)
  }
  return request<Post>(`/posts/${id}/reactions`, {
    method: 'POST',
    body: JSON.stringify({ vote }),
  })
}

export async function bookmarkPost(id: number): Promise<void> {
  await request<void>(`/posts/${id}/bookmark`, { method: 'POST' })
}

export async function unbookmarkPost(id: number): Promise<void> {
  await request<void>(`/posts/${id}/bookmark`, { method: 'DELETE' })
}

export async function uploadMedia(file: File): Promise<Media> {
  const form = new FormData()
  form.append('file', file)
  return requestForm<Media>(MEDIA_UPLOAD_PATH, form)
}
