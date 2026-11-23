import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { Event } from '@/types'

vi.mock('@/api/events', () => ({
  listEvents: vi.fn(),
  getEvent: vi.fn(),
  rsvpEvent: vi.fn(),
  cancelRsvp: vi.fn(),
}))

import * as api from '@/api/events'
import { useEventsStore } from './events'

function makeEvent(): Event {
  return {
    id: 7,
    organizer: { id: 1, username: 'kai', display_name: 'Kai', avatar_url: null },
    title: 'Convention meetup',
    description: 'Meet after the panels',
    event_type: 'meetup',
    starts_at: '2026-12-20T10:00:00Z',
    ends_at: '2026-12-20T16:00:00Z',
    timezone: 'Asia/Tokyo',
    venue_text: 'Harbour cafe',
    city: 'Tokyo',
    country: 'JP',
    is_online: false,
    capacity: 40,
    rsvp_count: 4,
    rsvp_status: null,
    cover_url: null,
    status: 'scheduled',
    created_at: '2026-11-01T00:00:00Z',
  }
}

beforeEach(() => {
  setActivePinia(createPinia())
  vi.clearAllMocks()
})

describe('events store', () => {
  it('applies an RSVP optimistically and rolls it back on failure', async () => {
    const event = makeEvent()
    const store = useEventsStore()
    store.byId[event.id] = event
    vi.mocked(api.rsvpEvent).mockRejectedValueOnce(new Error('offline'))

    await expect(store.rsvp(event.id, 'going')).resolves.toBeNull()

    expect(event.rsvp_status).toBeNull()
    expect(event.rsvp_count).toBe(4)
    expect(store.error).toContain('offline')
  })

  it('keeps the server result on a successful RSVP', async () => {
    const event = makeEvent()
    const store = useEventsStore()
    store.byId[event.id] = event
    const confirmed = { ...event, rsvp_status: 'going', rsvp_count: 5 }
    vi.mocked(api.rsvpEvent).mockResolvedValueOnce(confirmed)

    await expect(store.rsvp(event.id, 'going')).resolves.toEqual(confirmed)
    expect(store.byId[event.id]?.rsvp_status).toBe('going')
    expect(store.byId[event.id]?.rsvp_count).toBe(5)
  })

  it('does not change the count when switching between going and maybe', async () => {
    const event = { ...makeEvent(), rsvp_status: 'going', rsvp_count: 5 }
    const store = useEventsStore()
    store.byId[event.id] = event
    vi.mocked(api.rsvpEvent).mockResolvedValueOnce({ ...event, rsvp_status: 'maybe' })

    await store.rsvp(event.id, 'maybe')
    expect(store.byId[event.id]?.rsvp_count).toBe(5)
  })
})
