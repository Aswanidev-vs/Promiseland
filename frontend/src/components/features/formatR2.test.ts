import { describe, expect, it } from 'vitest'
import { eventDateParts, formatCapacity, fromRfc3339, toRfc3339 } from './formatR2'

describe('R2 formatting helpers', () => {
  it('converts a local datetime-local value to an RFC3339 instant and back', () => {
    const local = '2026-11-21T18:30'
    const instant = toRfc3339(local)
    const parsed = new Date(instant)

    expect(Number.isNaN(parsed.getTime())).toBe(false)
    expect(parsed.toISOString()).toBe(instant)
    expect(fromRfc3339(instant)).toBe(local)
    expect(toRfc3339('')).toBe('')
  })

  it('renders event dates in the local zone with a tabular day', () => {
    const parts = eventDateParts('2026-11-21T12:00:00Z', '2026-11-21T15:00:00Z')
    const start = new Date('2026-11-21T12:00:00Z')

    expect(parts.day).toBe(String(start.getDate()).padStart(2, '0'))
    expect(parts.month).toBe(new Intl.DateTimeFormat(undefined, { month: 'short' }).format(start))
    expect(parts.datetime).toBe('2026-11-21T12:00:00.000Z')
    expect(parts.time).toContain('–')
  })

  it('never reports a capacity limit when none was set', () => {
    expect(formatCapacity(7, 0)).toBe('7 going')
    expect(formatCapacity(7, 10)).toBe('7 of 10 going')
  })
})
