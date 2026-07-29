import { describe, it, expect } from 'vitest'
import {
  buildSubject,
  countWildcards,
  extractWildcardValues,
  subjectMatchesStream,
} from './subjectPatternUtils'

describe('extractWildcardValues', () => {
  it('recovers single-star values', () => {
    expect(extractWildcardValues('orders.*.created', 'orders.eu.created')).toEqual(['eu'])
  })

  it('recovers multiple star values', () => {
    expect(extractWildcardValues('MARKETPLACE.BIDS.*.*', 'MARKETPLACE.BIDS.lot1.user2')).toEqual([
      'lot1',
      'user2',
    ])
  })

  it('captures the rest of the subject for >', () => {
    expect(extractWildcardValues('orders.>', 'orders.eu.west.created')).toEqual(['eu.west.created'])
  })

  it('mixes * and >', () => {
    expect(extractWildcardValues('orders.*.events.>', 'orders.eu.events.user.created')).toEqual([
      'eu',
      'user.created',
    ])
  })

  it('is the inverse of buildSubject', () => {
    const pattern = 'a.*.b.*'
    const values = ['x', 'y']
    expect(extractWildcardValues(pattern, buildSubject(pattern, values))).toEqual(values)
  })

  it('returns empty slots on static token mismatch', () => {
    const result = extractWildcardValues('orders.*.created', 'invoices.eu.created')
    expect(result).toHaveLength(countWildcards('orders.*.created'))
    expect(result).toEqual([''])
  })

  it('blanks remaining slots after a mid-pattern mismatch', () => {
    expect(extractWildcardValues('a.*.b.*', 'a.x.WRONG.y')).toEqual(['x', ''])
  })

  it('handles subject shorter than pattern', () => {
    expect(extractWildcardValues('a.*.*', 'a.x')).toEqual(['x', ''])
  })

  it('returns empty array for pattern without wildcards', () => {
    expect(extractWildcardValues('orders.created', 'orders.created')).toEqual([])
  })
})

describe('subjectMatchesStream', () => {
  it('matches the exact configured subject', () => {
    expect(subjectMatchesStream('orders.>', ['orders.>'])).toBe(true)
    expect(subjectMatchesStream('orders.created', ['orders.created', 'orders.paid'])).toBe(true)
  })

  it('keeps a custom subject covered by a > filter', () => {
    expect(subjectMatchesStream('orders.custom.test', ['orders.>'])).toBe(true)
  })

  it('keeps a custom subject covered by a * filter', () => {
    expect(subjectMatchesStream('orders.custom', ['orders.*'])).toBe(true)
  })

  it('rejects a foreign subject not covered by any filter', () => {
    expect(subjectMatchesStream('events.created', ['orders.>'])).toBe(false)
    expect(subjectMatchesStream('orders.shipped', ['orders.created', 'orders.paid'])).toBe(false)
  })

  it('rejects a subject too short for a > filter', () => {
    expect(subjectMatchesStream('orders', ['orders.>'])).toBe(false)
  })

  it('rejects a subject longer than a * filter', () => {
    expect(subjectMatchesStream('orders.a.b', ['orders.*'])).toBe(false)
  })

  it('returns false for an empty candidate', () => {
    expect(subjectMatchesStream('', ['orders.>'])).toBe(false)
  })

  it('matches when any of several filters covers the subject', () => {
    expect(subjectMatchesStream('metrics.cpu', ['orders.>', 'metrics.*'])).toBe(true)
  })
})
