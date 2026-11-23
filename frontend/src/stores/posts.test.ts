import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { Post } from '@/types'

vi.mock('@/api/posts', () => ({
  getFeed: vi.fn(),
  getPost: vi.fn(),
  votePost: vi.fn(),
  bookmarkPost: vi.fn(),
  unbookmarkPost: vi.fn(),
}))

import * as api from '@/api/posts'
import { usePostsStore } from './posts'

function makePost(): Post {
  return {
    id: 7,
    author: { id: 1, username: 'kai', display_name: 'Kai', avatar_url: null },
    community: null,
    type: 'text',
    title: 'A discussion',
    body: 'Body',
    link_url: null,
    media: [],
    anime: [],
    tags: [],
    spoiler: false,
    spoiler_note: null,
    season: null,
    episode: null,
    vote_score: 3,
    my_vote: 0,
    comment_count: 0,
    bookmarked: false,
    creator_category: '',
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    edited_at: null,
  }
}

beforeEach(() => {
  setActivePinia(createPinia())
  vi.clearAllMocks()
})

describe('posts store', () => {
  it('optimistically applies a vote and rolls it back on failure', async () => {
    const post = makePost()
    const store = usePostsStore()
    store.byId[post.id] = post
    vi.mocked(api.votePost).mockRejectedValueOnce(new Error('offline'))

    await expect(store.vote(post.id, 1)).resolves.toBeNull()

    expect(post.my_vote).toBe(0)
    expect(post.vote_score).toBe(3)
    expect(store.error).toContain('offline')
  })

  it('keeps the server result on a successful vote', async () => {
    const post = makePost()
    const store = usePostsStore()
    store.byId[post.id] = post
    const voted = { ...post, my_vote: -1 as const, vote_score: 2 }
    vi.mocked(api.votePost).mockResolvedValueOnce(voted)

    await expect(store.vote(post.id, -1)).resolves.toEqual(voted)
    expect(store.byId[post.id]?.vote_score).toBe(2)
  })

  it('restores bookmark state when the request fails', async () => {
    const post = makePost()
    const store = usePostsStore()
    store.byId[post.id] = post
    vi.mocked(api.bookmarkPost).mockRejectedValueOnce(new Error('offline'))

    await expect(store.toggleBookmark(post.id)).resolves.toBeNull()
    expect(post.bookmarked).toBe(false)
  })
})
