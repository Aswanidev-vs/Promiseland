import { createPinia, setActivePinia } from 'pinia'
import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { joinCommunity, listCommunities } from '@/api/communities'
import { getFeed } from '@/api/posts'
import HomeView from './HomeView.vue'
import type { Community } from '@/types'

vi.mock('@/api/posts', () => ({ getFeed: vi.fn(), uploadMedia: vi.fn() }))
vi.mock('@/api/communities', () => ({ listCommunities: vi.fn(), joinCommunity: vi.fn() }))

const Blank = { template: '<div />' }

function community(id: number, name: string, slug: string, memberCount: number, isMember = false) {
  return {
    id,
    name,
    slug,
    description: null,
    icon_url: null,
    banner_url: null,
    visibility: 'public',
    member_count: memberCount,
    post_count: 0,
    role: 'member',
    is_member: isMember,
    created_at: '2026-10-01T10:00:00Z',
  } as unknown as Community
}

async function mountHome() {
  const pinia = createPinia()
  setActivePinia(pinia)
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/home', name: 'home', component: HomeView },
      { path: '/communities', name: 'communities', component: Blank },
      { path: '/posts/new', name: 'create-post', component: Blank },
      { path: '/discover', name: 'discover', component: Blank },
    ],
  })
  await router.push('/home')
  await router.isReady()

  const wrapper = mount(HomeView, { global: { plugins: [pinia, router] } })
  await flushPromises()
  return wrapper
}

describe('HomeView empty feed', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(getFeed).mockResolvedValue({ items: [], next_cursor: '' })
    vi.mocked(listCommunities).mockResolvedValue({
      items: [
        community(1, 'Cosplay craft', 'cosplay-craft', 42),
        community(2, 'Frieren', 'frieren', 1205),
        community(3, 'Already joined', 'already-joined', 7, true),
      ],
      next_cursor: '',
    })
    vi.mocked(joinCommunity).mockResolvedValue(undefined)
  })

  it('offers real communities to join instead of a bigger placeholder', async () => {
    const wrapper = await mountHome()

    expect(wrapper.text()).toContain('Your feed is quiet')
    expect(wrapper.text()).toContain('Communities to start with')
    expect(wrapper.text()).toContain('Cosplay craft')
    expect(wrapper.text()).toContain('42 members')
    // Already a member: not a suggestion.
    expect(wrapper.text()).not.toContain('Already joined')
  })

  it('joins a suggested community in place', async () => {
    const wrapper = await mountHome()

    const joinButton = wrapper.findAll('button').find((button) => button.text() === 'Join')!
    await joinButton.trigger('click')
    await flushPromises()

    expect(joinCommunity).toHaveBeenCalledWith('cosplay-craft')
    expect(wrapper.text()).not.toContain('Cosplay craft')
  })
})
