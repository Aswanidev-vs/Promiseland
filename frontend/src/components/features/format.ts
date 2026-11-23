/** Shared formatting helpers for R1 feature views. No UI primitives live here. */
import { ApiError } from '@/api/client'

const GENERIC_ERROR = 'Something went wrong. Try again.'

export function errorMessage(error: unknown, fallback = GENERIC_ERROR): string {
  return error instanceof Error && error.message ? error.message : fallback
}

export function isForbidden(error: unknown): boolean {
  return error instanceof ApiError && error.status === 403
}

export function isNotFound(error: unknown): boolean {
  return error instanceof ApiError && error.status === 404
}

export function formatWhen(value?: string | null): string {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  const minutes = Math.floor((Date.now() - date.getTime()) / 60000)
  if (minutes < 1) return 'just now'
  if (minutes < 60) return `${minutes}m ago`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h ago`
  const days = Math.floor(hours / 24)
  if (days < 7) return `${days}d ago`
  return date.toLocaleDateString(undefined, { month: 'short', day: 'numeric' })
}

export function formatDay(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return 'Earlier'
  const today = new Date()
  const startOf = (d: Date) => new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime()
  const days = Math.round((startOf(today) - startOf(date)) / 86400000)
  if (days === 0) return 'Today'
  if (days === 1) return 'Yesterday'
  if (days > 1 && days < 7) return date.toLocaleDateString(undefined, { weekday: 'long' })
  return date.toLocaleDateString(undefined, { month: 'long', day: 'numeric', year: 'numeric' })
}

export function plural(count: number, singular: string, pluralForm?: string): string {
  return `${count} ${count === 1 ? singular : pluralForm || `${singular}s`}`
}

/** Report reasons from PRD §7.15. Values are stable API strings. */
export const REPORT_REASONS = [
  { value: 'spam', label: 'Spam' },
  { value: 'harassment', label: 'Harassment' },
  { value: 'impersonation', label: 'Impersonation' },
  { value: 'hate_abuse', label: 'Hate or abuse' },
  { value: 'inappropriate', label: 'Inappropriate content' },
  { value: 'copyright', label: 'Copyright concern' },
  { value: 'spoiler_abuse', label: 'Spoiler abuse' },
  { value: 'unsafe_event', label: 'Unsafe event' },
  { value: 'other', label: 'Something else' },
] as const

export function reportReasonLabel(value: string): string {
  return REPORT_REASONS.find((reason) => reason.value === value)?.label || value
}

/** Only allow local paths from query strings; never redirect off-site. */
export function safeRedirect(value: unknown, fallback = '/home'): string {
  if (typeof value !== 'string' || !value.startsWith('/') || value.startsWith('//')) return fallback
  return value
}
