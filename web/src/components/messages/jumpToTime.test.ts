import { describe, it, expect } from 'vitest'
import { parseStartDate, toDatetimeLocal, minutesAgoLocal } from './jumpToTime'

describe('parseStartDate', () => {
  it('returns null for empty input', () => {
    expect(parseStartDate(null)).toBeNull()
    expect(parseStartDate(undefined)).toBeNull()
    expect(parseStartDate('')).toBeNull()
  })

  it('returns null for unparseable input', () => {
    expect(parseStartDate('not-a-date')).toBeNull()
  })

  it('parses a local datetime-local string to epoch ms', () => {
    // Interpreted as local time → compare against a local Date built the same way.
    const ms = parseStartDate('2026-06-12T14:32')
    expect(ms).toBe(new Date(2026, 5, 12, 14, 32).getTime())
  })

  it('is the inverse of toDatetimeLocal (to the minute)', () => {
    const d = new Date(2026, 0, 2, 9, 5)
    const str = toDatetimeLocal(d)
    expect(str).toBe('2026-01-02T09:05')
    expect(parseStartDate(str)).toBe(d.getTime())
  })
})

describe('toDatetimeLocal', () => {
  it('zero-pads month, day, hour and minute', () => {
    expect(toDatetimeLocal(new Date(2026, 2, 4, 7, 9))).toBe('2026-03-04T07:09')
  })
})

describe('minutesAgoLocal', () => {
  it('subtracts the given minutes from the reference time', () => {
    const now = new Date(2026, 5, 12, 15, 0)
    expect(minutesAgoLocal(60, now)).toBe('2026-06-12T14:00')
    expect(minutesAgoLocal(24 * 60, now)).toBe('2026-06-11T15:00')
  })
})
