import { createPinia } from 'pinia'
import { defineComponent, h } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ApiError } from '@/api/client'
import {
  deleteConnection,
  listConnections,
  reviewConnection,
} from '@/api/discovery'
import { useAuthStore } from '@/stores/auth'
import type { Connection, User } from '@/types'
import ConnectionsView from './ConnectionsView.vue'

vi.mock('@/api/discovery', () => ({
  listConnections: vi.fn(),
  reviewConnection: vi.fn(),
  deleteConnection: vi.fn(),
  getDiscoverySettings: vi.fn(),
  updateDiscoverySettings: vi.fn(),
  suppressCandidate: vi.fn(),
  unsuppressCandidate: vi.fn(),
  sendConnection: vi.fn(),
  listFriends: vi.fn(),
}))

const RouterLinkStub = defineComponent({
  props: { to: { type: [String, Object], default: '' } },
  setup(props, { slots }) {
    return () => h('a', { href: JSON.stringify(props.to) }, slots.default?.())
  },
})

const incoming: Connection = {
  id: 7,
  requester: { id: 2, username: 'mira', display_name: 'Mira', avatar_url: null },
  addressee: { id: 1, username: 'you', display_name: 'You', avatar_url: null },
  message: 'We both love Frieren — want to compare favourite arcs?',
  status: 'pending',
  created_at: '2026-11-22T10:00:00Z',
  updated_at: '2026-11-22T10:00:00Z',
} as Connection

async function setup() {
  const pinia = createPinia()
  const auth = useAuthStore(pinia)
  auth.user = {
    id: 1,
    username: 'you',
    display_name: 'You',
    avatar_url: null,
    banner_url: null,
    bio: null,
    location: null,
    visibility: 'public',
    discoverable: true,
    created_at: '2026-01-01T00:00:00Z',
  } as User

  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/connections', name: 'connections', component: ConnectionsView },
      { path: '/discover', name: 'discover', component: { template: '<div />' } },
      { path: '/home', name: 'home', component: { template: '<div />' } },
    ],
  })
  await router.push('/connections')
  await router.isReady()

  const wrapper = mount(ConnectionsView, {
    global: { plugins: [pinia, router], stubs: { RouterLink: RouterLinkStub } },
  })
  await flushPromises()
  return { wrapper, router }
}

function button(wrapper: ReturnType<typeof mount>, label: string) {
  const found = wrapper.findAll('button').find((item) => item.text() === label)
  if (!found) throw new Error(`Button not found: ${label}`)
  return found
}

describe('ConnectionsView', () => {
  beforeEach(() => {
    vi.mocked(listConnections).mockResolvedValue({ items: [incoming], next_cursor: '' })
    vi.mocked(reviewConnection).mockImplementation(async (_id, status) => ({
      ...incoming,
      status,
    }))
    vi.mocked(deleteConnection).mockResolvedValue(undefined)
  })

  it('shows incoming requests with the message and all three review actions', async () => {
    const { wrapper } = await setup()

    expect(wrapper.text()).toContain('Mira')
    expect(wrapper.text()).toContain(incoming.message)
    expect(wrapper.text()).toContain('Wants to connect')
    expect(listConnections).toHaveBeenCalledWith(
      expect.objectContaining({ direction: 'incoming', status: 'pending' }),
    )
    for (const label of ['Accept', 'Ignore', 'Reject']) {
      expect(button(wrapper, label).exists()).toBe(true)
    }
  })

  it('accepts, ignores and rejects incoming requests', async () => {
    for (const [label, status] of [
      ['Accept', 'accepted'],
      ['Ignore', 'ignored'],
      ['Reject', 'rejected'],
    ] as const) {
      vi.mocked(reviewConnection).mockClear()
      const { wrapper } = await setup()
      await button(wrapper, label).trigger('click')
      await flushPromises()
      expect(reviewConnection).toHaveBeenCalledWith(7, status)
    }
  })

  it('cancels an outgoing request from the Outgoing tab', async () => {
    const { wrapper } = await setup()

    const outgoingTab = wrapper.findAll('[role="tab"]').find((tab) => tab.text() === 'Outgoing')
    await outgoingTab!.trigger('click')
    await flushPromises()

    expect(listConnections).toHaveBeenCalledWith(
      expect.objectContaining({ direction: 'outgoing', status: 'pending' }),
    )
    expect(wrapper.find('.connection-row').exists()).toBe(true)
    await button(wrapper, 'Cancel request').trigger('click')
    await flushPromises()
    expect(deleteConnection).toHaveBeenCalledWith(7)
  })

  it('shows a permission-denied state when connections are forbidden', async () => {
    vi.mocked(listConnections).mockRejectedValueOnce(
      new ApiError(403, 'forbidden', 'Sign in to view connections.'),
    )
    const { wrapper } = await setup()

    expect(wrapper.text()).toContain('Connections are not available')
  })
})
