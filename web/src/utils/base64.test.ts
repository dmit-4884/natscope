import { describe, it, expect } from 'vitest'
import { decodeBase64ToUtf8, decodeBase64ToBytes, encodeBytesToBase64 } from './base64'

describe('base64 utils', () => {
  describe('decodeBase64ToUtf8', () => {
    it('decodes ASCII text', () => {
      const encoded = btoa('Hello World')
      expect(decodeBase64ToUtf8(encoded)).toBe('Hello World')
    })

    it('decodes JSON', () => {
      const encoded = btoa('{"key": "value"}')
      expect(decodeBase64ToUtf8(encoded)).toBe('{"key": "value"}')
    })

    it('decodes empty string', () => {
      const encoded = btoa('')
      expect(decodeBase64ToUtf8(encoded)).toBe('')
    })

    it('decodes special characters', () => {
      const encoded = btoa('test & <value>')
      expect(decodeBase64ToUtf8(encoded)).toBe('test & <value>')
    })
  })

  describe('decodeBase64ToBytes', () => {
    it('returns Uint8Array', () => {
      const encoded = btoa('hello')
      const bytes = decodeBase64ToBytes(encoded)
      expect(bytes).toBeInstanceOf(Uint8Array)
      expect(bytes.length).toBe(5)
    })

    it('returns correct byte values', () => {
      const encoded = btoa('AB')
      const bytes = decodeBase64ToBytes(encoded)
      expect(bytes[0]).toBe(65) // 'A'
      expect(bytes[1]).toBe(66) // 'B'
    })

    it('handles empty input', () => {
      const bytes = decodeBase64ToBytes(btoa(''))
      expect(bytes.length).toBe(0)
    })
  })

  describe('encodeBytesToBase64', () => {
    it('encodes a large array without throwing and round-trips', () => {
      const bytes = new Uint8Array(200_000)
      for (let i = 0; i < bytes.length; i++) {
        bytes[i] = i % 256
      }

      expect(() => encodeBytesToBase64(bytes)).not.toThrow()

      const encoded = encodeBytesToBase64(bytes)
      const decoded = decodeBase64ToBytes(encoded)
      expect(decoded).toEqual(bytes)
    })
  })
})
