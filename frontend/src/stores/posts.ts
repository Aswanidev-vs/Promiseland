import { ref } from 'vue'
import { defineStore } from 'pinia'
import { errorMessage } from '@/api/client'
import * as api from '@/api/posts'
import type { FeedParams, Post, Vote } from '@/types'

export const usePostsStore = defineStore('posts', () => {
  const feed = ref<Post[]>([])
  const byId = ref<Record<number, Post>>({})
  const loading = ref(false)
  const error = ref<string | null>(null)
  const nextCursor = ref('')

  function remember(post: Post): Post {
    byId.value = { ...byId.value, [post.id]: post }
    const index = feed.value.findIndex((item) => item.id === post.id)
    if (index !== -1) feed.value.splice(index, 1, post)
    return post
  }

  async function loadFeed(p: FeedParams = {}, append = false): Promise<void> {
    loading.value = true
    error.value = null
    try {
      const page = await api.getFeed(p)
      feed.value = append ? [...feed.value, ...page.items] : page.items
      page.items.forEach(remember)
      nextCursor.value = page.next_cursor
    } catch (cause) {
      error.value = errorMessage(cause)
    } finally {
      loading.value = false
    }
  }

  async function loadPost(id: number): Promise<Post | null> {
    try {
      return remember(await api.getPost(id))
    } catch (cause) {
      error.value = errorMessage(cause)
      return null
    }
  }

  function applyVote(post: Post, from: Vote, to: Vote): void {
    post.my_vote = to
    post.vote_score += to - from
  }

  /** Applies the new vote immediately and restores the old one on failure. */
  async function vote(id: number, value: Vote): Promise<Post | null> {
    const post = byId.value[id] ?? feed.value.find((item) => item.id === id)
    if (!post) return null
    const previous = post.my_vote
    applyVote(post, previous, value)
    try {
      return remember(await api.votePost(id, value))
    } catch (cause) {
      applyVote(post, value, previous)
      error.value = errorMessage(cause)
      return null
    }
  }

  /** Returns the resulting bookmarked state, or null when the request failed. */
  async function toggleBookmark(id: number): Promise<boolean | null> {
    const post = byId.value[id] ?? feed.value.find((item) => item.id === id)
    if (!post) return null
    const previous = post.bookmarked
    post.bookmarked = !previous
    try {
      if (post.bookmarked) await api.bookmarkPost(id)
      else await api.unbookmarkPost(id)
      return post.bookmarked
    } catch (cause) {
      post.bookmarked = previous
      error.value = errorMessage(cause)
      return null
    }
  }

  return { feed, byId, loading, error, nextCursor, loadFeed, loadPost, vote, toggleBookmark }
})
