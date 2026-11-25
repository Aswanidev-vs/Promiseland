import { createPinia, setActivePinia } from 'pinia'
import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { getCommunity, listRules, updateCommunity } from '@/api/communities'
import CommunityManageView from './CommunityManageView.vue'
import type { Community } from '@/types'

vi.mock('@/api/communities', () => ({
  getCommunity: vi.fn(),
  listRules: vi.fn(),
  updateCommunity: vi.fn(),
  addRule: vi.fn(),
  addModerator: vi.fn(),
}))

const Blank = { template: '<div />' }

function makeCommunity(role: Community['role']): Community {
  return {
    id: 5,
    name: 'Cosplay craft',
    slug: 'cosplay-craft',
    description: 'Props and sewing',
    icon_url: null,
    banner_url: null,
    visibility: 'public',
    member_count: 42,
    post_count: 9,
    role,
    is_member: true,
    created_at: '2026-10-01T10:00:00Z',
  } as unknown as Community
}

async function mountManage() {
  const pinia = createPinia()
  setActivePinia(pinia)
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/communities/:slug/manage', name: 'community-manage', component: CommunityManageView },
      { path: '/communities/:slug', name: 'community', component: Blank },
      { path: '/communities', name: 'communities', component: Blank },
    ],
  })
  await router.push('/communities/cosplay-craft/manage')
  await router.isReady()

  const wrapper = mount(CommunityManageView, { global: { plugins: [pinia, router] } })
  await flushPromises()
  return wrapper
}

describe('CommunityManageView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(getCommunity).mockResolvedValue(makeCommunity('owner'))
    vi.mocked(listRules).mockResolvedValue([
      { id: 1, community_id: 5, title: 'Be respectful', body: 'No harassment.', sort_order: 0 },
    ])
    vi.mocked(updateCommunity).mockResolvedValue(makeCommunity('owner'))
  })

  it('shows the four staff sections to an owner', async () => {
    const wrapper = await mountManage()

    const headings = wrapper.findAll('.manage__heading').map((heading) => heading.text())
    expect(headings).toEqual(['Details', 'Appearance', 'Rules', 'Moderators'])
    expect(wrapper.text()).toContain('Cosplay craft')
    expect(wrapper.text()).toContain('Be respectful')
    expect(wrapper.findAll('.uploader')).toHaveLength(2)
  })

  it('saves details through the community endpoint', async () => {
    const wrapper = await mountManage()

    await wrapper.get('.manage__form').trigger('submit')
    await flushPromises()

    expect(updateCommunity).toHaveBeenCalledWith('cosplay-craft', {
      name: 'Cosplay craft',
      description: 'Props and sewing',
      visibility: 'public',
    })
  })

  it('refuses the page to a plain member', async () => {
    vi.mocked(getCommunity).mockResolvedValue(makeCommunity('member'))

    const wrapper = await mountManage()

    expect(wrapper.text()).toContain('Only community staff can manage this community')
    expect(wrapper.findAll('.manage__heading')).toHaveLength(0)
    expect(listRules).not.toHaveBeenCalled()
  })
})
