import { describe, expect, it } from 'vitest'
import { parseDurationToNs } from './duration'

const S = 1_000_000_000

describe('parseDurationToNs', () => {
  it.each([
    ['1h', 3600 * S],
    ['30m', 1800 * S],
    ['1h30m', 5400 * S],
    ['90s', 90 * S],
    ['500ms', 500_000_000],
    ['2d', 172_800 * S],
    ['1.5h', 5400 * S],
    [' 5m ', 300 * S],
    ['0', 0],
    ['0s', 0],
    ['', 0],
  ])('reads %j', (text, ns) => {
    expect(parseDurationToNs(text)).toEqual({ ns })
  })

  it.each(['soon', '5 m', '1w', 'm5', '-5s', '3600000000000'])('refuses %j and shows an example', (text) => {
    expect(parseDurationToNs(text).error).toMatch(/like 30s, 5m, 12h or 7d/)
  })
})
