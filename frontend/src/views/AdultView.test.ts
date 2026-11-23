import { createPinia } from 'pinia'
import { defineComponent, h } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ApiError } from '@/api/client'
import {
  getAdultProfile,
  getAdultStatus,
  listAdultMatches,
  reportAdult,
  saveAdultProfile,
  unmatch,
} from '@/api/adult'
import { useAuthStore } from '@/stores/auth'
import type { AdultMatch, AdultProfile, AdultStatus, User, UserSummary } from '@/types'
import AdultView from './AdultView.vue'

vi.mock('@/api/adult', () => ({
  getAdultStatus: vi.fn(),
  getAdultProfile: vi.fn(),
  saveAdultProfile: vi.fn(),
  likeAdult: vi.fn(),
  listAdultMatches: vi.fn(),
  unmatch: vi.fn(),
  reportAdult: vi.fn(),
}))

const RouterLinkStub = defineComponent({
  props: { to: { type: [String, Object], default: '' } },
  setup(props, { slots }) {
    return () => h('a', { href: JSON.stringify(props.to) }, slots.default?.())
  },
})

const disabledStatus: AdultStatus = {
  enabled: false,
  ageAssurance: 'development_stub',
  eligible: false,
  message: 'Adult connections are not enabled on this deployment.',
}

const allowedStatus: AdultStatus = {
  ...disabledStatus,
  enabled: true,
  eligible: true,
  message: '',
}

const otherUser: UserSummary = { id: 7, username: 'mira', display_name: 'Mira', avatar_url: null }
const me: UserSummary = { id: 1, username: 'you', display_name: 'You', avatar_url: null }

const profileFixture: AdultProfile = {
  user_id: 1,
  display_name: 'Rin',
  intent: 'friendship',
  bio: 'I like long conversations after a convention.',
  boundaries: 'No late-night messages.',
  visibility: 'opt_in',
  created_at: '2026-11-01T00:00:00Z',
}

const matchFixture: AdultMatch = {
  id: 11,
  users: [otherUser, me],
  created_at: '2026-11-20T00:00:00Z',
  conversation_id: 42,
  unmatched_at: null,
}

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
      { path: '/adult', name: 'adult', component: AdultView },
      { path: '/home', name: 'home', component: { template: '<div />' } },
    ],
  })
  await router.push('/adult')
  await router.isReady()

  const wrapper = mount(AdultView, {
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

describe('AdultView', () => {
  beforeEach(() => {
    vi.mocked(getAdultStatus).mockResolvedValue(disabledStatus)
    // A 404 here means "no opt-in profile yet", not a broken area.
    vi.mocked(getAdultProfile).mockRejectedValue(new ApiError(404, 'not_found', 'No profile yet.'))
    vi.mocked(listAdultMatches).mockResolvedValue([])
    vi.mocked(saveAdultProfile).mockImplementation(async (input) => ({
      ...profileFixture,
      display_name: input.displayName,
      intent: input.intent,
      bio: input.bio,
      boundaries: input.boundaries,
    }))
    vi.mocked(unmatch).mockResolvedValue(undefined)
    vi.mocked(reportAdult).mockResolvedValue(undefined)
  })

  it('shows only the eligibility requirements when the feature is disabled', async () => {
    const { wrapper } = await setup()

    expect(wrapper.text()).toContain('This area is not available')
    for (const heading of ['Age assurance', 'Legal review', 'Separate opt-in', 'Safety controls']) {
      expect(wrapper.text()).toContain(heading)
    }
    expect(wrapper.text()).toContain(disabledStatus.message)
    expect(wrapper.findAll('a').some((link) => link.text() === 'Back to home')).toBe(true)
    expect(wrapper.findAll('form')).toHaveLength(0)
    expect(wrapper.text()).not.toContain('Opt-in connections')
    expect(wrapper.text()).not.toContain('Your matches')
    expect(getAdultProfile).not.toHaveBeenCalled()
    expect(listAdultMatches).not.toHaveBeenCalled()
  })

  it('shows the same gated panel when the account is not eligible', async () => {
    vi.mocked(getAdultStatus).mockResolvedValue({
      ...allowedStatus,
      eligible: false,
      message: 'Age assurance has not been completed.',
    })

    const { wrapper } = await setup()

    expect(wrapper.text()).toContain('This area is not available')
    expect(wrapper.text()).toContain('Age assurance has not been completed.')
    expect(wrapper.text()).toContain('Safety controls')
    expect(wrapper.findAll('form')).toHaveLength(0)
    expect(wrapper.text()).not.toContain('Your matches')
    expect(getAdultProfile).not.toHaveBeenCalled()
    expect(listAdultMatches).not.toHaveBeenCalled()
  })

  it('validates the opt-in form and saves the camelCase payload', async () => {
    vi.mocked(getAdultStatus).mockResolvedValue(allowedStatus)

    const { wrapper } = await setup()

    expect(wrapper.get('h1').text()).toBe('Opt-in connections')
    expect(wrapper.text()).toContain('Safety in this area')

    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(saveAdultProfile).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('Enter a display name for this area.')
    expect(wrapper.text()).toContain('Choose what you are here for.')
    expect(wrapper.text()).toContain('Write a short introduction.')
    expect(wrapper.text()).toContain('Add at least one boundary.')

    await wrapper.get('#adult-display-name').setValue('Mira')
    await wrapper.get('#adult-intent').setValue('both')
    await wrapper.get('#adult-bio').setValue('Conventions and long anime talks.')
    await wrapper.get('#adult-boundaries').setValue('No direct messages before a match.')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(saveAdultProfile).toHaveBeenCalledWith({
      displayName: 'Mira',
      intent: 'both',
      bio: 'Conventions and long anime talks.',
      boundaries: 'No direct messages before a match.',
    })
    expect(wrapper.text()).toContain('Your adult profile was saved.')
    expect(wrapper.text()).toContain('not shown on your main profile, posts, or communities')
  })

  it('treats a 404 from the status endpoint as a closed gate', async () => {
    vi.mocked(getAdultStatus).mockRejectedValue(new ApiError(404, 'not_found', 'Feature disabled.'))

    const { wrapper } = await setup()

    expect(wrapper.text()).toContain('This area is not available')
    expect(wrapper.findAll('form')).toHaveLength(0)
    expect(getAdultProfile).not.toHaveBeenCalled()
  })

  it('offers a retry when the availability check fails', async () => {
    vi.mocked(getAdultStatus)
      .mockRejectedValueOnce(new Error('Network down'))
      .mockResolvedValue(disabledStatus)

    const { wrapper } = await setup()

    expect(wrapper.text()).toContain('We could not check availability')
    await button(wrapper, 'Try again').trigger('click')
    await flushPromises()

    expect(getAdultStatus).toHaveBeenCalledTimes(2)
    expect(wrapper.text()).toContain('This area is not available')
  })

  it('unmatches from the matches list', async () => {
    vi.mocked(getAdultStatus).mockResolvedValue(allowedStatus)
    vi.mocked(getAdultProfile).mockResolvedValue(profileFixture)
    vi.mocked(listAdultMatches).mockResolvedValue([matchFixture])

    const { wrapper } = await setup()

    expect(wrapper.text()).toContain('You both chose to connect.')
    expect(wrapper.text()).not.toContain('No matches yet')
    await button(wrapper, 'Unmatch').trigger('click')
    await flushPromises()

    expect(unmatch).toHaveBeenCalledWith(11)
    expect(wrapper.text()).toContain('No matches yet')
    expect(wrapper.text()).toContain('Match removed.')
  })

  it('ignores a second unmatch click while the first request is in flight', async () => {
    vi.mocked(getAdultStatus).mockResolvedValue(allowedStatus)
    vi.mocked(getAdultProfile).mockResolvedValue(profileFixture)
    vi.mocked(listAdultMatches).mockResolvedValue([matchFixture])
    let finishUnmatch: (() => void) | undefined
    vi.mocked(unmatch).mockImplementation(
      () =>
        new Promise<void>((resolve) => {
          finishUnmatch = resolve
        }),
    )

    const { wrapper } = await setup()
    const unmatchButton = button(wrapper, 'Unmatch')
    await unmatchButton.trigger('click')
    await unmatchButton.trigger('click')

    expect(unmatch).toHaveBeenCalledTimes(1)
    finishUnmatch?.()
    await flushPromises()
  })

  it('routes a report through the adult safety workflow', async () => {
    vi.mocked(getAdultStatus).mockResolvedValue(allowedStatus)
    vi.mocked(getAdultProfile).mockResolvedValue(profileFixture)
    vi.mocked(listAdultMatches).mockResolvedValue([matchFixture])

    const { wrapper } = await setup()

    await button(wrapper, 'Report').trigger('click')
    await flushPromises()
    expect(wrapper.find('[role="dialog"]').exists()).toBe(true)

    await wrapper.get('#report-reason').setValue('harassment')
    await wrapper.get('#report-details').setValue('Unwanted contact.')
    await wrapper.get('[role="dialog"] form').trigger('submit')
    await flushPromises()

    expect(reportAdult).toHaveBeenCalledWith({
      targetType: 'adult_profile',
      targetId: 7,
      reason: 'harassment',
      details: 'Unwanted contact.',
    })
    expect(wrapper.text()).toContain('Your report was sent for priority review.')
  })

  it('shows a permission-denied panel instead of the form when data access is forbidden', async () => {
    vi.mocked(getAdultStatus).mockResolvedValue(allowedStatus)
    vi.mocked(getAdultProfile).mockRejectedValue(new ApiError(403, 'forbidden', 'No access.'))

    const { wrapper } = await setup()

    expect(wrapper.text()).toContain('Access to this area is not available')
    expect(wrapper.findAll('form')).toHaveLength(0)
  })
})
