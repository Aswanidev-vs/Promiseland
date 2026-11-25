import { createPinia } from 'pinia'
import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { listCreators } from '@/api/creators'
import { listCommunities } from '@/api/communities'
import { listEvents } from '@/api/events'
import DiscoverRail from './DiscoverRail.vue'
import type { Community, Creator, Event } from '@/types'

vi.mock('@/api/events', () => ({ listEvents: vi.fn() }))
vi.mock('@/api/creators', () => ({ listCreators: vi.fn() }))
vi.mock('@/api/communities', () => ({ listCommunities: vi.fn() }))

const Blank = { template: '<div />' }

const event = {
  id: 4,
  title: 'Harbour screening',
  starts_at: '2026-12-14T18:00:00Z',
  city: 'London',
  is_online: false,
} as unknown as Event

const community = {
  id: 7,
  name: 'Cosplay craft',
  slug: 'cosplay-craft',
  description: 'Props and sewing',
  icon_url: null,
  banner_url: null,
  visibility: 'public',
  member_count: 42,
  post_count: 9,
  role: 'member',
  is_member: true,
  created_at: '2026-10-01T10:00:00Z',
} as unknown as Community

const creator = {
  user: { id: 9, username: 'mira', display_name: 'Mira', avatar_url: null },
  categories: ['cosplay'],
  featured_posts: [],
  post_count: 3,
  follower_count: 12,
} as unknown as Creator

async function mountRail(sections?: Array<'events' | 'creators' | 'communities'>) {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/home', name: 'home', component: Blank },
      { path: '/events', name: 'events', component: Blank },
      { path: '/events/:id', name: 'event', component: Blank },
      { path: '/communities/:slug', name: 'community', component: Blank },
      { path: '/creators', name: 'creators', component: Blank },
      { path: '/u/:username', name: 'profile', component: Blank },
    ],
  })
  await router.push('/home')
  await router.isReady()

  const wrapper = mount(DiscoverRail, {
    props: sections ? { sections } : {},
    global: { plugins: [createPinia(), router] },
  })
  await flushPromises()
  return wrapper
}

describe('DiscoverRail', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(listEvents).mockResolvedValue({ items: [event], next_cursor: '' })
    vi.mocked(listCreators).mockResolvedValue({ items: [creator], next_cursor: '' })
    vi.mocked(listCommunities).mockResolvedValue({ items: [community], next_cursor: '' })
  })

  it('lists real events and creators with a way to browse each', async () => {
    const wrapper = await mountRail()

    expect(wrapper.text()).toContain('Upcoming events')
    expect(wrapper.text()).toContain('Harbour screening')
    expect(wrapper.text()).toContain('Mira')
    expect(wrapper.text()).toContain('@mira')

    const hrefs = wrapper.findAll('a').map((link) => link.attributes('href'))
    expect(hrefs).toContain('/events/4')
    expect(hrefs).toContain('/u/mira')
    expect(hrefs).toContain('/events')
    expect(hrefs).toContain('/creators')
  })

  it('says plainly when a section has nothing to show', async () => {
    vi.mocked(listEvents).mockResolvedValue({ items: [], next_cursor: '' })
    vi.mocked(listCreators).mockResolvedValue({ items: [], next_cursor: '' })

    const wrapper = await mountRail()

    expect(wrapper.text()).toContain('No upcoming events yet.')
    expect(wrapper.text()).toContain('No creators listed yet.')
  })

  it('drops a section quietly when its request fails', async () => {
    vi.mocked(listEvents).mockRejectedValue(new Error('offline'))
    vi.mocked(listCreators).mockRejectedValue(new Error('offline'))

    const wrapper = await mountRail()

    expect(wrapper.text()).not.toContain('offline')
    expect(wrapper.text()).toContain('No upcoming events yet.')
  })

  it('only fetches the sections it was asked for', async () => {
    const wrapper = await mountRail(['communities'])

    expect(listCommunities).toHaveBeenCalledTimes(1)
    expect(listEvents).not.toHaveBeenCalled()
    expect(listCreators).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('Cosplay craft')
    expect(wrapper.text()).toContain('42 members')
    expect(wrapper.text()).not.toContain('Upcoming events')
  })
})
