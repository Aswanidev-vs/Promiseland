import { createPinia, setActivePinia } from 'pinia'
import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { getAdminOverview, listAdminUsers } from '@/api/admin'
import { useAuthStore } from '@/stores/auth'
import AdminView from './AdminView.vue'
import type { User } from '@/types'

vi.mock('@/api/admin', () => ({
  getAdminOverview: vi.fn(),
  listAdminUsers: vi.fn(),
  listAdminPosts: vi.fn(),
  listAdminCommunities: vi.fn(),
  listAdminEvents: vi.fn(),
  setAdminUserActive: vi.fn(),
}))

const Blank = { template: '<div />' }

async function mountAdmin(staff: boolean) {
  const pinia = createPinia()
  setActivePinia(pinia)
  useAuthStore(pinia).user = {
    id: 1,
    username: 'keeper',
    display_name: 'Keeper',
    is_staff: staff,
  } as unknown as User

  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/admin', name: 'admin', component: AdminView },
      { path: '/home', name: 'home', component: Blank },
      { path: '/moderation', name: 'moderation', component: Blank },
      { path: '/u/:username', name: 'profile', component: Blank },
    ],
  })
  await router.push('/admin')
  await router.isReady()

  const wrapper = mount(AdminView, { global: { plugins: [pinia, router] } })
  await flushPromises()
  return wrapper
}

const counts = {
  users: 31,
  active_users: 30,
  suspended_users: 1,
  active_sessions: 32,
  posts: 0,
  comments: 0,
  communities: 5,
  events: 0,
  media: 3,
  media_bytes: 2048,
  open_reports: 2,
  pending_memberships: 0,
}

describe('AdminView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(getAdminOverview).mockResolvedValue({
      counts,
      recent_users: [{ id: 1, username: 'keeper', display_name: 'Keeper', created_at: '2026-11-25 10:00:00' }],
      recent_posts: [],
      open_reports: [],
    })
    vi.mocked(listAdminUsers).mockResolvedValue({ items: [], next_cursor: '' })
  })

  it('shows the console and its counts to staff', async () => {
    const wrapper = await mountAdmin(true)

    expect(getAdminOverview).toHaveBeenCalled()
    expect(wrapper.text()).toContain('Admin')
    expect(wrapper.text()).toContain('31')
    expect(wrapper.text()).toContain('2 KB')
    expect(wrapper.findAll('.admin__tabs button')).toHaveLength(5)
    expect(wrapper.text()).toContain('Open reports')
  })

  it('refuses the console to an ordinary account without asking the API', async () => {
    const wrapper = await mountAdmin(false)

    expect(wrapper.text()).toContain('This console is for staff')
    expect(getAdminOverview).not.toHaveBeenCalled()
  })

  it('loads the users table when that tab opens', async () => {
    const wrapper = await mountAdmin(true)

    await wrapper.findAll('.admin__tabs button').find((button) => button.text() === 'Users')!.trigger('click')
    await flushPromises()

    expect(listAdminUsers).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('No accounts match.')
  })
})
