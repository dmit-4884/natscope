import { describe, it, expect } from 'vitest'
import { StreamName } from './StreamName'

describe('StreamName', () => {
  describe('create', () => {
    it('creates valid stream name', () => {
      const result = StreamName.create('ORDERS')
      expect(result.isOk()).toBe(true)
      expect(result.value.value).toBe('ORDERS')
    })

    it('allows alphanumeric with dashes and underscores', () => {
      expect(StreamName.create('my-stream_v2').isOk()).toBe(true)
    })

    it('trims whitespace', () => {
      const result = StreamName.create('  ORDERS  ')
      expect(result.isOk()).toBe(true)
      expect(result.value.value).toBe('ORDERS')
    })

    it('rejects empty string', () => {
      const result = StreamName.create('')
      expect(result.isErr()).toBe(true)
      expect(result.error.code).toBe('EMPTY')
    })

    it('rejects whitespace-only', () => {
      const result = StreamName.create('   ')
      expect(result.isErr()).toBe(true)
      expect(result.error.code).toBe('EMPTY')
    })

    it('rejects names exceeding 256 chars', () => {
      const result = StreamName.create('A'.repeat(257))
      expect(result.isErr()).toBe(true)
      expect(result.error.code).toBe('TOO_LONG')
    })

    it('accepts name at max length', () => {
      expect(StreamName.create('A'.repeat(256)).isOk()).toBe(true)
    })

    it('rejects invalid characters (dots)', () => {
      const result = StreamName.create('my.stream')
      expect(result.isErr()).toBe(true)
      expect(result.error.code).toBe('INVALID_CHARS')
    })

    it('rejects invalid characters (spaces)', () => {
      const result = StreamName.create('my stream')
      expect(result.isErr()).toBe(true)
      expect(result.error.code).toBe('INVALID_CHARS')
    })

    it('rejects reserved prefix _', () => {
      const result = StreamName.create('_internal')
      expect(result.isErr()).toBe(true)
      expect(result.error.code).toBe('RESERVED')
    })

    it('rejects $ character as invalid chars', () => {
      const result = StreamName.create('$system')
      expect(result.isErr()).toBe(true)
      expect(result.error.code).toBe('INVALID_CHARS')
    })
  })

  describe('fromTrusted', () => {
    it('creates without validation', () => {
      const name = StreamName.fromTrusted('ANY_NAME')
      expect(name.value).toBe('ANY_NAME')
    })
  })

  describe('toUpperCase', () => {
    it('returns uppercase version', () => {
      const name = StreamName.fromTrusted('orders')
      expect(name.toUpperCase()).toBe('ORDERS')
    })
  })

  describe('toString', () => {
    it('returns the value', () => {
      const name = StreamName.fromTrusted('ORDERS')
      expect(name.toString()).toBe('ORDERS')
    })
  })
})
