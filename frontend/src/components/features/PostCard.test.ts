import { defineComponent, h } from 'vue'
import { createPinia } from 'pinia'
import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { Post } from '@/types'
import PostCard from './PostCard.vue'

vi.mock('@/api/posts', () => ({
  votePost: vi.fn(async () => undefined),
  bookmarkPost: vi.fn(async () => undefined),
  unbookmarkPost: vi.fn(async () => undefined),
}))
vi.mock('@/api/moderation', () => ({ report: vi.fn(async () => undefined) }))

const RouterLinkStub = defineComponent({
  props: { to: { type: [String, Object], default: '' } },
  setup(props, { slots }) {
    return () => h('a', { href: String(props.to) }, slots.default?.())
  },
})

function makePost(overrides: Partial<Post> = {}): Post {
  return {
    id: 7,
    author: { id: 2, username: 'mira', display_name: 'Mira', avatar_url: null },
    community: null,
    type: 'text',
    title: 'Finale thoughts',
    body: 'The last episode reveals who wrote the letter.',
    link_url: null,
    media: [],
    anime: [],
    tags: [],
    spoiler: true,
    spoiler_note: 'season 1 finale',
    season: null,
    episode: null,
    vote_score: 4,
    my_vote: 0,
    comment_count: 2,
    bookmarked: false,
    created_at: '2026-11-20T12:00:00Z',
    updated_at: '2026-11-20T12:00:00Z',
    edited_at: null,
    ...overrides,
  } as unknown as Post
}

function mountCard(post: Post) {
  return mount(PostCard, {
    props: { post },
    global: {
      plugins: [createPinia()],
      stubs: { RouterLink: RouterLinkStub },
    },
  })
}

describe('PostCard', () => {
  beforeEach(() => vi.clearAllMocks())

  it('obscures spoiler body until the reader explicitly reveals it', async () => {
    const wrapper = mountCard(makePost())
    const toggle = wrapper.get('.spoiler__toggle')

    expect(toggle.attributes('aria-expanded')).toBe('false')
    expect(wrapper.text()).not.toContain('The last episode reveals who wrote the letter.')
    expect(wrapper.text()).toContain('Marked as a spoiler: season 1 finale')

    await toggle.trigger('click')

    expect(wrapper.get('.spoiler__toggle').attributes('aria-expanded')).toBe('true')
    expect(wrapper.text()).toContain('The last episode reveals who wrote the letter.')
  })

  it('does not shield a post that has no spoiler metadata', () => {
    const wrapper = mountCard(makePost({ spoiler: false, spoiler_note: null }))
    expect(wrapper.find('.spoiler').exists()).toBe(false)
    expect(wrapper.text()).toContain('The last episode reveals who wrote the letter.')
  })

  it('keeps vote state local while the request is in flight', async () => {
    const wrapper = mountCard(makePost())
    const upvote = wrapper.get('button[aria-label="Upvote post"]')

    await upvote.trigger('click')

    expect(wrapper.get('.post-card__score').text()).toBe('5')
    expect(upvote.attributes('aria-pressed')).toBe('true')
  })
})
