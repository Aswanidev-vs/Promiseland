import { createPinia } from 'pinia'
import { defineComponent, h } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  getDiscoverySettings,
  listFriends,
  sendConnection,
  suppressCandidate,
  unsuppressCandidate,
  updateDiscoverySettings,
} from '@/api/discovery'
import type { DiscoveryCandidate } from '@/types'
import DiscoverView from './DiscoverView.vue'

vi.mock('@/api/discovery', () => ({
  listFriends: vi.fn(),
  getDiscoverySettings: vi.fn(),
  updateDiscoverySettings: vi.fn(),
  suppressCandidate: vi.fn(),
  unsuppressCandidate: vi.fn(),
  sendConnection: vi.fn(),
  listConnections: vi.fn(),
  reviewConnection: vi.fn(),
  deleteConnection: vi.fn(),
}))

const RouterLinkStub = defineComponent({
  props: { to: { type: [String, Object], default: '' } },
  setup(props, { slots }) {
    return () => h('a', { href: JSON.stringify(props.to) }, slots.default?.())
  },
})

const candidate: DiscoveryCandidate = {
  user: {
    id: 12,
    username: 'mira',
    display_name: 'Mira',
    avatar_url: null,
    banner_url: null,
    bio: null,
    location: null,
    visibility: 'public',
    discoverable: true,
    created_at: '2026-01-01T00:00:00Z',
    interests: [],
    anime_interests: [],
    follower_count: 3,
    following_count: 4,
    post_count: 2,
    is_following: false,
    is_blocked: false,
  },
  shared: { anime: ['Frieren'], genres: ['Isekai'], interests: [], communities: [] },
  shared_count: 2,
  explanation: 'You both follow Frieren and like Isekai',
  connection_status: 'none',
} as DiscoveryCandidate

async function setup() {
  const pinia = createPinia()
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/discover', name: 'discover', component: DiscoverView },
      { path: '/connections', name: 'connections', component: { template: '<div />' } },
      { path: '/settings', name: 'settings', component: { template: '<div />' } },
      { path: '/home', name: 'home', component: { template: '<div />' } },
    ],
  })
  await router.push('/discover')
  await router.isReady()
  const wrapper = mount(DiscoverView, {
    global: { plugins: [pinia, router], stubs: { RouterLink: RouterLinkStub } },
  })
  await flushPromises()
  return { wrapper }
}

function button(wrapper: ReturnType<typeof mount>, label: string) {
  const found = wrapper.findAll('button').find((item) => item.text() === label)
  if (!found) throw new Error(`Button not found: ${label}`)
  return found
}

describe('DiscoverView', () => {
  beforeEach(() => {
    vi.mocked(listFriends).mockResolvedValue({ items: [candidate], next_cursor: '' })
    vi.mocked(getDiscoverySettings).mockResolvedValue({ discoverable: true })
    vi.mocked(updateDiscoverySettings).mockImplementation(async (patch) => patch)
    vi.mocked(suppressCandidate).mockResolvedValue(undefined)
    vi.mocked(unsuppressCandidate).mockResolvedValue(undefined)
    vi.mocked(sendConnection).mockResolvedValue({
      id: 1,
      requester: candidate.user,
      addressee: candidate.user,
      message: '',
      status: 'pending',
      created_at: '2026-11-23T00:00:00Z',
      updated_at: '2026-11-23T00:00:00Z',
    })
  })

  it('renders the promise and the compatibility explanation', async () => {
    const { wrapper } = await setup()

    expect(wrapper.get('h1').text()).toBe('Your fandom. Your people. Your place.')
    expect(wrapper.text()).toContain(candidate.explanation)
    expect(wrapper.text()).toContain('Frieren')
  })

  it('wires the discoverability toggle to the settings endpoint', async () => {
    const { wrapper } = await setup()
    const toggle = wrapper.get('[role="switch"]')

    expect(toggle.attributes('aria-checked')).toBe('true')
    await toggle.trigger('click')
    await flushPromises()

    expect(updateDiscoverySettings).toHaveBeenCalledWith({ discoverable: false })
    expect(wrapper.get('[role="switch"]').attributes('aria-checked')).toBe('false')
  })

  it('suppresses a suggestion and restores it from the undo toast', async () => {
    const { wrapper } = await setup()

    await button(wrapper, 'Not interested').trigger('click')
    await flushPromises()

    expect(suppressCandidate).toHaveBeenCalledWith(12)
    expect(wrapper.text()).not.toContain(candidate.explanation)
    expect(wrapper.text()).toContain('Suggestion hidden.')

    await button(wrapper, 'Undo').trigger('click')
    await flushPromises()

    expect(unsuppressCandidate).toHaveBeenCalledWith(12)
    expect(wrapper.text()).toContain(candidate.explanation)
  })

  it('sends a connection request with the message from the dialog', async () => {
    const { wrapper } = await setup()

    await button(wrapper, 'Connect').trigger('click')
    await wrapper.get('#connect-message').setValue('Both Frieren fans — hello!')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(sendConnection).toHaveBeenCalledWith({
      userId: 12,
      message: 'Both Frieren fans — hello!',
    })
    expect(wrapper.text()).toContain('Request sent')
  })
})
