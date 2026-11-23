import { createPinia } from 'pinia'
import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { cancelRsvp, rsvpEvent } from '@/api/events'
import type { Event } from '@/types'
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

function mountControl(event: Event) {
  return mount(RsvpControl, {
    props: { event },
    global: { plugins: [createPinia()] },
  })
}

describe('RsvpControl', () => {
  beforeEach(() => {
    vi.mocked(rsvpEvent).mockResolvedValue(makeEvent({ rsvp_status: 'going', rsvp_count: 6 }))
    vi.mocked(cancelRsvp).mockResolvedValue(makeEvent())
  })

  it('emits the chosen status for Going and Maybe', async () => {
    const wrapper = mountControl(makeEvent())

    expect(wrapper.get('.rsvp__capacity').text()).toBe('5 of 20 going')

    await wrapper.get('[data-status="going"]').trigger('click')
    expect(rsvpEvent).toHaveBeenCalledWith(12, 'going')
    expect(wrapper.emitted('change')?.[0]).toEqual(['going'])

    await wrapper.get('[data-status="maybe"]').trigger('click')
    expect(rsvpEvent).toHaveBeenCalledWith(12, 'maybe')
    expect(wrapper.emitted('change')?.[1]).toEqual(['maybe'])
  })

  it('emits null when an existing response is cancelled', async () => {
    const wrapper = mountControl(makeEvent({ rsvp_status: 'going' }))

    await wrapper.get('[data-status="cancel"]').trigger('click')

    expect(cancelRsvp).toHaveBeenCalledWith(12)
    expect(wrapper.emitted('change')?.[0]).toEqual([null])
  })

  it('explains when an event has reached capacity', () => {
    const wrapper = mountControl(makeEvent({ capacity: 10, rsvp_count: 10 }))

    expect(wrapper.get('.rsvp__capacity').text()).toBe('10 of 10 going')
    expect(wrapper.get('.rsvp__note').text()).toBe('This event is at capacity.')
  })
})
