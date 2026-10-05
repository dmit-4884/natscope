import { describe, it, expect } from 'vitest'
import type { FilterValues } from './AdvancedFilters'
import { EMPTY_FILTERS, isSearchFilter, toSearchQuery } from './searchQuery'

const filters = (over: Partial<FilterValues>): FilterValues => ({ ...EMPTY_FILTERS, ...over })
const at = (local: string) => new Date(local).getTime()

describe('isSearchFilter', () => {
  it('turns on for text, a header or a stop point, not for the list filters', () => {
    expect(isSearchFilter(EMPTY_FILTERS)).toBe(false)
    expect(isSearchFilter(filters({ subject: 'orders.>', startSequence: 10, startDate: '2026-10-05T10:00' }))).toBe(false)
    expect(isSearchFilter(filters({ contentFilter: 'needle' }))).toBe(true)
    expect(isSearchFilter(filters({ contentFilter: '   ' }))).toBe(false)
    expect(isSearchFilter(filters({ header: 'X-Trace' }))).toBe(true)
    expect(isSearchFilter(filters({ stopSequence: 5 }))).toBe(true)
    expect(isSearchFilter(filters({ stopDate: '2026-10-05T10:00' }))).toBe(true)
  })
})

describe('toSearchQuery', () => {
  it('is null for list filters', () => {
    expect(toSearchQuery(filters({ subject: 'orders.>' }), 'backward')).toBeNull()
  })

  it('carries the text, the regex switch and the subject', () => {
    expect(toSearchQuery(filters({ contentFilter: ' needle ', contentRegex: true, subject: 'orders.>' }), 'backward')).toEqual({
      direction: 'backward',
      subject_filter: 'orders.>',
      text: 'needle',
      regex: true,
    })
  })

  it('splits a header into name and value', () => {
    expect(toSearchQuery(filters({ header: ' X-Trace = abc=1 ' }), 'backward')).toMatchObject({ header_name: 'X-Trace', header_value: 'abc=1' })
    expect(toSearchQuery(filters({ header: 'X-Trace' }), 'backward')).toMatchObject({ header_name: 'X-Trace', header_value: undefined })
  })

  it('reads newest first from the start sequence down to the stop sequence', () => {
    expect(toSearchQuery(filters({ startSequence: 900, stopSequence: 100 }), 'backward')).toMatchObject({
      direction: 'backward',
      to_seq: 900,
      from_seq: 100,
    })
  })

  it('reads oldest first from the start sequence up to the stop sequence', () => {
    expect(toSearchQuery(filters({ startSequence: 100, stopSequence: 900 }), 'forward')).toMatchObject({
      direction: 'forward',
      from_seq: 100,
      to_seq: 900,
    })
  })

  it('reads forward from a start time, like the jump, and stops at the end of the stop minute', () => {
    expect(toSearchQuery(filters({ startDate: '2026-10-05T10:00', stopDate: '2026-10-05T11:30' }), 'backward')).toMatchObject({
      direction: 'forward',
      from_time: at('2026-10-05T10:00'),
      to_time: at('2026-10-05T11:30') + 59_999,
    })
  })

  it('stops a newest-first search at the start of the stop minute', () => {
    expect(toSearchQuery(filters({ stopDate: '2026-10-05T10:00' }), 'backward')).toMatchObject({
      direction: 'backward',
      from_time: at('2026-10-05T10:00'),
    })
  })
})
