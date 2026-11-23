import { defineComponent, h } from 'vue'
import { createPinia } from 'pinia'
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { useAuthStore } from '@/stores/auth'
import type { TeamUp, User } from '@/types'
import TeamUpCard from './TeamUpCard.vue'

const RouterLinkStub = defineComponent({
  props: { to: { type: [String, Object], default: '' } },
  setup(props, { slots }) {
    return () => h('a', { href: String(props.to) }, slots.default?.())
  },
})

function makeTeamUp(overrides: Partial<TeamUp> = {}): TeamUp {
  return {
    id: 4,
    owner: { id: 2, username: 'mira', display_name: 'Mira', avatar_url: null },
    title: 'Convention group',
    description: 'Looking for a group to explore the dealer room.',
    event: null,
    categories: ['cosplay', 'photography'],
    status: 'open',
    applicant_count: 3,
    my_application: null,
    created_at: '2026-11-01T10:00:00Z',
    ...overrides,
  } as TeamUp
}

function mountCard(teamUp: TeamUp, viewerId: number) {
  const pinia = createPinia()
  const auth = useAuthStore(pinia)
  auth.user = { id: viewerId, username: 'viewer', display_name: 'Viewer' } as User
  return mount(TeamUpCard, {
    props: { teamUp },
    global: { plugins: [pinia], stubs: { RouterLink: RouterLinkStub } },
  })
}

describe('TeamUpCard', () => {
  it('shows the owner a review action', () => {
    const wrapper = mountCard(makeTeamUp(), 2)

    expect(wrapper.text()).toContain('Review applications')
    expect(wrapper.text()).toContain('Open')
    expect(wrapper.text()).toContain('3 applicants')
    expect(wrapper.text()).toContain('Cosplay')
    expect(wrapper.text()).toContain('Photography')
  })

  it('hides owner actions from other members', () => {
    const wrapper = mountCard(makeTeamUp(), 99)

    expect(wrapper.text()).not.toContain('Review applications')
    expect(wrapper.text()).toContain('View request')
  })

  it('surfaces the linked event and an existing application', () => {
    const wrapper = mountCard(
      makeTeamUp({
        event: {
          id: 7,
          title: 'Winter meetup',
          organizer: { id: 2, username: 'mira', display_name: 'Mira', avatar_url: null },
          description: '',
          event_type: 'meetup',
          starts_at: '2026-11-21T12:00:00Z',
          ends_at: '2026-11-21T15:00:00Z',
          timezone: 'UTC',
          venue_text: '',
          city: '',
          country: '',
          is_online: false,
          capacity: 0,
          rsvp_count: 0,
          rsvp_status: null,
          cover_url: null,
          status: 'scheduled',
          created_at: '2026-11-01T10:00:00Z',
        },
        my_application: 'pending',
      }),
      99,
    )

    expect(wrapper.text()).toContain('Winter meetup')
    expect(wrapper.text()).toContain('Application pending')
  })
})
