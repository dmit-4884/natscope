import { describe, expect, it } from 'vitest'
import { bytesToHex, hexToBytes } from './hex'

describe('hex', () => {
  it('round-trips bytes', () => {
    expect(bytesToHex(new Uint8Array([0xca, 0xfe, 0x00, 0x0a]))).toBe('cafe000a')
    expect(hexToBytes('cafe000a')).toEqual(new Uint8Array([0xca, 0xfe, 0x00, 0x0a]))
  })

  it('accepts spacing, case and a 0x prefix', () => {
    expect(hexToBytes(' 0xCA fe ')).toEqual(new Uint8Array([0xca, 0xfe]))
    expect(hexToBytes('')).toEqual(new Uint8Array())
  })

  it('rejects odd lengths and non-hex characters', () => {
    expect(hexToBytes('abc')).toBeNull()
    expect(hexToBytes('zz')).toBeNull()
  })
})
