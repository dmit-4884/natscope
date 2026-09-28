import { describe, it, expect } from 'vitest'
import { parseIntOr, clampInt32, INT32_MIN, INT32_MAX } from './numbers'

describe('parseIntOr', () => {
  it('parses a valid integer string', () => {
    expect(parseIntOr('42', 0)).toBe(42)
  })

  it('falls back on empty or non-numeric input', () => {
    expect(parseIntOr('', 7)).toBe(7)
    expect(parseIntOr('abc', 7)).toBe(7)
  })

  it('does not eat a literal zero', () => {
    expect(parseIntOr('0', 5)).toBe(0)
  })
})

describe('clampInt32', () => {
  it('leaves in-range values untouched', () => {
    expect(clampInt32(0)).toBe(0)
    expect(clampInt32(-1)).toBe(-1)
    expect(clampInt32(1000)).toBe(1000)
  })

  it('clamps a value above the int32 max', () => {
    expect(clampInt32(3000000000)).toBe(INT32_MAX)
  })

  it('clamps a value below the int32 min', () => {
    expect(clampInt32(-3000000000)).toBe(INT32_MIN)
  })

  it('leaves the boundary values untouched', () => {
    expect(clampInt32(INT32_MAX)).toBe(INT32_MAX)
    expect(clampInt32(INT32_MIN)).toBe(INT32_MIN)
  })
})
