import { describe, it, expect } from 'vitest'
import { stableJson } from './stableJson'

describe('stableJson', () => {
  it('serializes reordered objects identically', () => {
    expect(stableJson({ a: 1, b: 2 })).toBe(stableJson({ b: 2, a: 1 }))
  })

  it('sorts nested object keys but preserves array order', () => {
    expect(stableJson({ outer: { z: 1, a: [{ y: 1, x: 2 }] } })).toBe('{"outer":{"a":[{"x":2,"y":1}],"z":1}}')
    expect(stableJson(['b', 'a'])).toBe('["b","a"]')
  })

  it('drops undefined values so an absent key equals an unset one', () => {
    expect(stableJson({ a: 1, b: undefined })).toBe(stableJson({ a: 1 }))
  })

  it('indents when a space is given', () => {
    expect(stableJson({ b: 1, a: 2 }, 2)).toBe('{\n  "a": 2,\n  "b": 1\n}')
  })

  it('passes primitives through', () => {
    expect(stableJson('x')).toBe('"x"')
    expect(stableJson(null)).toBe('null')
    expect(stableJson(undefined)).toBeUndefined()
  })
})
