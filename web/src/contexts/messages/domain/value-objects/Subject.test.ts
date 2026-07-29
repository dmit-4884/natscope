import { describe, it, expect } from 'vitest'
import { Subject } from './Subject'

describe('Subject', () => {
  describe('create', () => {
    it('creates a valid subject', () => {
      const result = Subject.create('orders.created.123')
      expect(result.isOk()).toBe(true)
      expect(result.value.value).toBe('orders.created.123')
    })

    it('trims whitespace', () => {
      const result = Subject.create('  orders.created  ')
      expect(result.isOk()).toBe(true)
      expect(result.value.value).toBe('orders.created')
    })

    it('rejects empty string', () => {
      const result = Subject.create('')
      expect(result.isErr()).toBe(true)
      expect(result.error.code).toBe('EMPTY')
    })

    it('rejects whitespace-only', () => {
      const result = Subject.create('   ')
      expect(result.isErr()).toBe(true)
      expect(result.error.code).toBe('EMPTY')
    })

    it('rejects string exceeding max length', () => {
      const result = Subject.create('a'.repeat(257))
      expect(result.isErr()).toBe(true)
      expect(result.error.code).toBe('TOO_LONG')
    })

    it('accepts string at max length', () => {
      const result = Subject.create('a'.repeat(256))
      expect(result.isOk()).toBe(true)
    })

    it('rejects invalid characters', () => {
      const result = Subject.create('orders/created')
      expect(result.isErr()).toBe(true)
      expect(result.error.code).toBe('INVALID_CHARS')
    })

    it('allows wildcards', () => {
      expect(Subject.create('orders.*').isOk()).toBe(true)
      expect(Subject.create('orders.>').isOk()).toBe(true)
    })

    it('allows dashes', () => {
      expect(Subject.create('my-orders.created').isOk()).toBe(true)
    })
  })

  describe('fromTrusted', () => {
    it('creates without validation', () => {
      const subject = Subject.fromTrusted('any.subject.here')
      expect(subject.value).toBe('any.subject.here')
    })
  })

  describe('toPattern', () => {
    it('replaces UUIDs with wildcard', () => {
      const subject = Subject.fromTrusted('orders.550e8400-e29b-41d4-a716-446655440000.created')
      expect(subject.toPattern()).toBe('orders.*.created')
    })

    it('replaces trailing numeric IDs with wildcard', () => {
      const subject = Subject.fromTrusted('users.profile.12345')
      expect(subject.toPattern()).toBe('users.profile.*')
    })

    it('returns unchanged pattern for plain subjects', () => {
      const subject = Subject.fromTrusted('orders.created')
      expect(subject.toPattern()).toBe('orders.created')
    })
  })

  describe('matchesPattern', () => {
    it('matches exact subject', () => {
      const subject = Subject.fromTrusted('orders.created')
      expect(subject.matchesPattern('orders.created')).toBe(true)
    })

    it('matches single wildcard', () => {
      const subject = Subject.fromTrusted('orders.created')
      expect(subject.matchesPattern('orders.*')).toBe(true)
    })

    it('does not match wrong wildcard depth', () => {
      const subject = Subject.fromTrusted('orders.created.123')
      expect(subject.matchesPattern('orders.*')).toBe(false)
    })

    it('matches multi-level wildcard', () => {
      const subject = Subject.fromTrusted('orders.created.123')
      expect(subject.matchesPattern('orders.>')).toBe(true)
    })

    it('matches multi-level wildcard single token', () => {
      const subject = Subject.fromTrusted('orders.created')
      expect(subject.matchesPattern('orders.>')).toBe(true)
    })

    it('does not match different prefix', () => {
      const subject = Subject.fromTrusted('users.login')
      expect(subject.matchesPattern('orders.*')).toBe(false)
    })

    it('matches multiple wildcards', () => {
      const subject = Subject.fromTrusted('orders.created.123')
      expect(subject.matchesPattern('orders.*.*')).toBe(true)
    })

    it('does not match shorter subject', () => {
      const subject = Subject.fromTrusted('orders')
      expect(subject.matchesPattern('orders.created')).toBe(false)
    })
  })

  describe('parts', () => {
    it('splits subject into tokens', () => {
      expect(Subject.fromTrusted('orders.created.123').parts()).toEqual(['orders', 'created', '123'])
    })

    it('returns single part for simple subject', () => {
      expect(Subject.fromTrusted('orders').parts()).toEqual(['orders'])
    })
  })

  describe('parent', () => {
    it('returns parent subject', () => {
      const parent = Subject.fromTrusted('orders.created.123').parent()
      expect(parent).not.toBeNull()
      expect(parent!.value).toBe('orders.created')
    })

    it('returns null for single-token subject', () => {
      expect(Subject.fromTrusted('orders').parent()).toBeNull()
    })
  })

  describe('lastToken', () => {
    it('returns last token', () => {
      expect(Subject.fromTrusted('orders.created.123').lastToken()).toBe('123')
    })

    it('returns only token for single subject', () => {
      expect(Subject.fromTrusted('orders').lastToken()).toBe('orders')
    })
  })

  describe('depth', () => {
    it('returns number of tokens', () => {
      expect(Subject.fromTrusted('orders.created.123').depth()).toBe(3)
      expect(Subject.fromTrusted('orders').depth()).toBe(1)
    })
  })

  describe('toString', () => {
    it('returns the value', () => {
      expect(Subject.fromTrusted('orders.created').toString()).toBe('orders.created')
    })
  })
})
