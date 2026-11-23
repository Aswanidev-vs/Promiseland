import { ref } from 'vue'
import { defineStore } from 'pinia'
import { errorMessage } from '@/api/client'
import * as api from '@/api/events'
import type { Event, ListEventsParams, RsvpStatus } from '@/types'

/** List cache for the events directory plus an optimistic RSVP layer. */
export const useEventsStore = defineStore('events', () => {
  const events = ref<Event[]>([])
  const byId = ref<Record<number, Event>>({})
  const loading = ref(false)
  const error = ref<string | null>(null)
  const nextCursor = ref('')

  function remember(event: Event): Event {
    byId.value = { ...byId.value, [event.id]: event }
    const index = events.value.findIndex((item) => item.id === event.id)
    if (index !== -1) events.value.splice(index, 1, event)
    return event
  }

  /** Only entering/leaving an RSVP changes the count; going ↔ maybe does not. */
  function applyRsvp(event: Event, from: string | null, to: string | null): void {
    if (from === null && to !== null) event.rsvp_count += 1
    else if (from !== null && to === null) {
      event.rsvp_count = Math.max(0, event.rsvp_count - 1)
    }
    event.rsvp_status = to
  }

  async function loadEvents(p: ListEventsParams = {}, append = false): Promise<void> {
    loading.value = true
    error.value = null
    try {
      const page = await api.listEvents(p)
      events.value = append ? [...events.value, ...page.items] : page.items
      page.items.forEach(remember)
      nextCursor.value = page.next_cursor
    } catch (cause) {
      error.value = errorMessage(cause)
    } finally {
      loading.value = false
    }
  }

  async function loadEvent(id: number): Promise<Event | null> {
    try {
      return remember(await api.getEvent(id))
    } catch (cause) {
      error.value = errorMessage(cause)
      return null
    }
  }

  /** Updates the cached event immediately and rolls back if the API rejects. */
  async function rsvp(id: number, status: RsvpStatus): Promise<Event | null> {
    const event = byId.value[id]
    const previous = event?.rsvp_status ?? null
    if (event) applyRsvp(event, previous, status)
    try {
      return remember(await api.rsvpEvent(id, status))
    } catch (cause) {
      if (event) applyRsvp(event, status, previous)
      error.value = errorMessage(cause)
      return null
    }
  }

  async function cancelRsvp(id: number): Promise<Event | null> {
    const event = byId.value[id]
    const previous = event?.rsvp_status ?? null
    if (event) applyRsvp(event, previous, null)
    try {
      return remember(await api.cancelRsvp(id))
    } catch (cause) {
      if (event) applyRsvp(event, null, previous)
      error.value = errorMessage(cause)
      return null
    }
  }

  return {
    events,
    byId,
    loading,
    error,
    nextCursor,
    loadEvents,
    loadEvent,
    rsvp,
    cancelRsvp,
  }
})
