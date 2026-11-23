/** Release 2 formatting helpers and stable label maps. No UI primitives here. */

export const EVENT_TYPES = [
  { value: 'convention', label: 'Convention' },
  { value: 'meetup', label: 'Meetup' },
  { value: 'screening', label: 'Screening' },
  { value: 'photoshoot', label: 'Photoshoot' },
  { value: 'club', label: 'Club' },
  { value: 'online', label: 'Online' },
  { value: 'competition', label: 'Competition' },
  { value: 'other', label: 'Other' },
] as const

export const CREATOR_CATEGORIES = [
  { value: 'cosplay', label: 'Cosplay' },
  { value: 'crossplay', label: 'Crossplay' },
  { value: 'fan_art', label: 'Fan art' },
  { value: 'photography', label: 'Photography' },
  { value: 'edit', label: 'Edit' },
  { value: 'collection', label: 'Collection' },
  { value: 'other', label: 'Other' },
] as const

export const TEAM_UP_STATUSES = [
  { value: 'open', label: 'Open' },
  { value: 'filled', label: 'Filled' },
  { value: 'closed', label: 'Closed' },
] as const

export function eventTypeLabel(value: string): string {
  return EVENT_TYPES.find((item) => item.value === value)?.label || 'Event'
}

export function creatorCategoryLabel(value: string): string {
  return CREATOR_CATEGORIES.find((item) => item.value === value)?.label || 'Other'
}

export function teamUpStatusLabel(value: string): string {
  return TEAM_UP_STATUSES.find((item) => item.value === value)?.label || value
}

/** Display raw API status values in sentence case without changing their wire value. */
export function sentenceCase(value: string): string {
  return value ? value.charAt(0).toUpperCase() + value.slice(1) : value
}

export interface EventDateParts {
  month: string
  day: string
  weekday: string
  time: string
  full: string
  datetime: string
}

/** Event instants are stored with an offset; display them in the viewer's local zone. */
export function eventDateParts(startsAt: string, endsAt?: string | null): EventDateParts {
  const start = new Date(startsAt)
  if (Number.isNaN(start.getTime())) {
    return { month: '—', day: '—', weekday: '', time: '', full: 'Date to be confirmed', datetime: '' }
  }
  const end = endsAt ? new Date(endsAt) : null
  const timeFormat = new Intl.DateTimeFormat(undefined, {
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
    hourCycle: 'h23',
  })
  const time = timeFormat.format(start)
  const endTime = end && !Number.isNaN(end.getTime()) ? timeFormat.format(end) : ''
  return {
    month: new Intl.DateTimeFormat(undefined, { month: 'short' }).format(start),
    day: new Intl.DateTimeFormat(undefined, { day: '2-digit' }).format(start),
    weekday: new Intl.DateTimeFormat(undefined, { weekday: 'short' }).format(start),
    time: endTime ? `${time} – ${endTime}` : time,
    full: new Intl.DateTimeFormat(undefined, {
      weekday: 'long',
      month: 'long',
      day: 'numeric',
      year: 'numeric',
    }).format(start),
    datetime: start.toISOString(),
  }
}

/** A compact, single-line date for cards and lists. */
export function formatEventWhen(startsAt: string, endsAt?: string | null): string {
  const parts = eventDateParts(startsAt, endsAt)
  return parts.datetime ? `${parts.full} · ${parts.time}` : parts.full
}

/** Capacity note never claims a count when the organizer set no limit (0). */
export function formatCapacity(count: number, capacity: number): string {
  const going = Math.max(0, count || 0)
  return capacity > 0 ? `${going} of ${capacity} going` : `${going} going`
}

/** datetime-local value → RFC3339 UTC instant. */
export function toRfc3339(localValue: string): string {
  if (!localValue) return ''
  const date = new Date(localValue)
  return Number.isNaN(date.getTime()) ? '' : date.toISOString()
}

/** RFC3339 instant → datetime-local value in the viewer's local zone. */
export function fromRfc3339(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  const pad = (part: number) => String(part).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`
}
