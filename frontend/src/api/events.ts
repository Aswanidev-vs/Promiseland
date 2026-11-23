import { buildQuery, request } from './client'
import type {
  CreateEventInput,
  Event,
  EventComment,
  EventUpdate,
  ListEventsParams,
  Page,
  PaginationParams,
  RsvpStatus,
} from '@/types'

/** Filter order follows the R2 endpoint contract: q,type,from,to,city,country,online,limit,cursor. */
export async function listEvents(p: ListEventsParams = {}): Promise<Page<Event>> {
  return request<Page<Event>>(
    `/events${buildQuery({
      q: p.q,
      type: p.type,
      from: p.from,
      to: p.to,
      city: p.city,
      country: p.country,
      online: p.online,
      limit: p.limit,
      cursor: p.cursor,
    })}`,
  )
}

export async function getEvent(id: number): Promise<Event> {
  return request<Event>(`/events/${id}`)
}

export async function createEvent(input: CreateEventInput): Promise<Event> {
  return request<Event>('/events', {
    method: 'POST',
    body: JSON.stringify({
      title: input.title,
      description: input.description,
      event_type: input.eventType,
      starts_at: input.startsAt,
      ends_at: input.endsAt,
      timezone: input.timezone,
      venue_text: input.venueText,
      city: input.city,
      country: input.country,
      is_online: input.isOnline,
      capacity: input.capacity,
      cover_media_id: input.coverMediaId,
    }),
  })
}

export async function updateEvent(id: number, patch: Partial<Event>): Promise<Event> {
  return request<Event>(`/events/${id}`, { method: 'PATCH', body: JSON.stringify(patch) })
}

export async function deleteEvent(id: number): Promise<void> {
  await request<void>(`/events/${id}`, { method: 'DELETE' })
}

/** Join or change an RSVP. The body is `{status}`, not a bare string. */
export async function rsvpEvent(id: number, status: RsvpStatus): Promise<Event> {
  return request<Event>(`/events/${id}/rsvp`, {
    method: 'POST',
    body: JSON.stringify({ status }),
  })
}

export async function cancelRsvp(id: number): Promise<Event> {
  return request<Event>(`/events/${id}/rsvp`, { method: 'DELETE' })
}

export async function listEventUpdates(id: number): Promise<EventUpdate[]> {
  return request<EventUpdate[]>(`/events/${id}/updates`)
}

export async function addEventUpdate(id: number, body: string): Promise<EventUpdate> {
  return request<EventUpdate>(`/events/${id}/updates`, {
    method: 'POST',
    body: JSON.stringify({ body }),
  })
}

export async function listEventComments(
  id: number,
  p: PaginationParams = {},
): Promise<Page<EventComment>> {
  return request<Page<EventComment>>(
    `/events/${id}/comments${buildQuery({ limit: p.limit, cursor: p.cursor })}`,
  )
}

export async function addEventComment(
  id: number,
  body: string,
  parentId?: number,
): Promise<EventComment> {
  return request<EventComment>(`/events/${id}/comments`, {
    method: 'POST',
    body: JSON.stringify({ body, parent_id: parentId }),
  })
}
