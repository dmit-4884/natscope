import { describe, expect, it } from 'vitest'
import { keyTtlRaisedTo, parseKeyTtl } from './keyTtl'

describe('parseKeyTtl', () => {
  it.each([
    ['', undefined],
    ['30s', 30_000_000_000],
    ['5m', 300_000_000_000],
    ['1h30m', 5_400_000_000_000],
    ['1.5h', 5_400_000_000_000],
    ['90', 90_000_000_000],
    [' 2m ', 120_000_000_000],
  ])('reads %j', (text, ns) => {
    expect(parseKeyTtl(text)).toEqual(ns === undefined ? {} : { ns })
  })

  it.each(['soon', '5 m', '1d', 'm5', '-5s'])('refuses %j', (text) => {
    expect(parseKeyTtl(text).error).toMatch(/duration like 30s/)
  })

  it('refuses less than a second', () => {
    expect(parseKeyTtl('500ms').error).toMatch(/at least 1s/)
    expect(parseKeyTtl('0').error).toMatch(/at least 1s/)
  })
})

describe('keyTtlRaisedTo', () => {
  it('returns the marker when the TTL is below it', () => {
    expect(keyTtlRaisedTo(5_000_000_000, 60_000_000_000)).toBe(60_000_000_000)
  })

  it.each([
    [60_000_000_000, 60_000_000_000],
    [120_000_000_000, 60_000_000_000],
    [undefined, 60_000_000_000],
    [5_000_000_000, undefined],
  ])('returns nothing for %j against marker %j', (ttl, marker) => {
    expect(keyTtlRaisedTo(ttl, marker)).toBeUndefined()
  })
})
