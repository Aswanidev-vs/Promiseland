import { createPinia, setActivePinia } from 'pinia'
import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { cancelRsvp, rsvpEvent } from '@/api/events'
import { useAuthStore } from '@/stores/auth'
import type { Event, User } from '@/types'
import RsvpControl from './RsvpControl.vue'

vi.mock('@/api/events', () => ({
  rsvpEvent: vi.fn(),
  cancelRsvp: vi.fn(),
}))

function makeEvent(overrides: Partial<Event> = {}): Event {
  return {
    id: 12,
    organizer: { id: 2, username: 'mira', display_name: 'Mira', avatar_url: null },
    title: 'Meetup',
    description: 'A gathering',
    event_type: 'meetup',
    starts_at: '2026-11-21T18:00:00Z',
    ends_at: '2026-11-21T21:00:00Z',
    timezone: 'UTC',
    venue_text: 'Harbour hall',
    city: 'London',
    country: 'UK',
    is_online: false,
    capacity: 20,
    rsvp_count: 5,
    rsvp_status: null,
    cover_url: null,
    status: 'scheduled',
    created_at: '2026-11-01T10:00:00Z',
    ...overrides,
  } as Event
}

async function mountControl(event: Event, { signedIn = true } = {}) {
  const pinia = createPinia()
  setActivePinia(pinia)
  if (signedIn) {
    useAuthStore(pinia).user = { id: 2, username: 'mira' } as unknown as User
  }

  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/events/:id', name: 'event', component: { template: '<div />' } },
      { path: '/login', name: 'login', component: { template: '<div />' } },
    ],
  })
  await router.push('/events/12')
  await router.isReady()

  return {
    wrapper: mount(RsvpControl, {
      props: { event },
      global: { plugins: [pinia, router] },
    }),
    router,
  }
}

describe('RsvpControl', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(rsvpEvent).mockResolvedValue(makeEvent({ rsvp_status: 'going', rsvp_count: 6 }))
    vi.mocked(cancelRsvp).mockResolvedValue(makeEvent())
  })

  it('emits the chosen status for Going and Maybe', async () => {
    const { wrapper } = await mountControl(makeEvent())

    expect(wrapper.get('.rsvp__capacity').text()).toBe('5 of 20 going')

    await wrapper.get('[data-status="going"]').trigger('click')
    expect(rsvpEvent).toHaveBeenCalledWith(12, 'going')
    expect(wrapper.emitted('change')?.[0]).toEqual(['going'])

    await wrapper.get('[data-status="maybe"]').trigger('click')
    expect(rsvpEvent).toHaveBeenCalledWith(12, 'maybe')
    expect(wrapper.emitted('change')?.[1]).toEqual(['maybe'])
  })

  it('emits null when an existing response is cancelled', async () => {
    const { wrapper } = await mountControl(makeEvent({ rsvp_status: 'going' }))

    await wrapper.get('[data-status="cancel"]').trigger('click')

    expect(cancelRsvp).toHaveBeenCalledWith(12)
    expect(wrapper.emitted('change')?.[0]).toEqual([null])
  })

  it('explains when an event has reached capacity', async () => {
    const { wrapper } = await mountControl(makeEvent({ capacity: 10, rsvp_count: 10 }))

    expect(wrapper.get('.rsvp__capacity').text()).toBe('10 of 10 going')
    expect(wrapper.get('.rsvp__note').text()).toBe('This event is at capacity.')
  })

  it('sends guests to sign in instead of calling the API', async () => {
    const { wrapper, router } = await mountControl(makeEvent(), { signedIn: false })

    await wrapper.get('[data-status="going"]').trigger('click')
    await flushPromises()

    expect(rsvpEvent).not.toHaveBeenCalled()
    expect(router.currentRoute.value.path).toBe('/login')
    expect(router.currentRoute.value.query.redirect).toBe('/events/12')
  })
})
