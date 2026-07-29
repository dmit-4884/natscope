import { describe, it, expect } from 'vitest'
import { resolveNavTarget } from './navTarget'

describe('resolveNavTarget', () => {
  it('backward list: down goes to older via backward', () => {
    expect(resolveNavTarget('down', 'backward', 10)).toEqual({ startSeq: 9, apiDirection: 'backward' })
  })

  it('backward list: up goes to newer via forward', () => {
    expect(resolveNavTarget('up', 'backward', 10)).toEqual({ startSeq: 11, apiDirection: 'forward' })
  })

  it('forward list: down goes to newer via forward', () => {
    expect(resolveNavTarget('down', 'forward', 10)).toEqual({ startSeq: 11, apiDirection: 'forward' })
  })

  it('forward list: up goes to older via backward', () => {
    expect(resolveNavTarget('up', 'forward', 10)).toEqual({ startSeq: 9, apiDirection: 'backward' })
  })

  it('returns edge instead of start_seq <= 0 (backward list, down at seq 1)', () => {
    expect(resolveNavTarget('down', 'backward', 1)).toBe('edge')
  })

  it('returns edge at seq 1 for forward list going up', () => {
    expect(resolveNavTarget('up', 'forward', 1)).toBe('edge')
  })
})
