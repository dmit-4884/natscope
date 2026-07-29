import { describe, it, expect } from 'vitest'
import { SubjectPattern } from './SubjectPattern'

describe('SubjectPattern', () => {
  describe('create', () => {
    it('creates valid pattern', () => {
      const result = SubjectPattern.create('orders.*.created')
      expect(result.isOk()).toBe(true)
      expect(result.value.value).toBe('orders.*.created')
    })

    it('detects wildcards', () => {
      const result = SubjectPattern.create('orders.*.>')
      expect(result.isOk()).toBe(true)
      expect(result.value.hasWildcards).toBe(true)
    })

    it('detects no wildcards', () => {
      const result = SubjectPattern.create('orders.created')
      expect(result.isOk()).toBe(true)
      expect(result.value.hasWildcards).toBe(false)
    })

    it('trims whitespace', () => {
      const result = SubjectPattern.create('  orders.*  ')
      expect(result.isOk()).toBe(true)
      expect(result.value.value).toBe('orders.*')
    })

    it('rejects empty string', () => {
      const result = SubjectPattern.create('')
      expect(result.isErr()).toBe(true)
      expect(result.error).toBe('EMPTY_PATTERN')
    })

    it('rejects invalid characters', () => {
      const result = SubjectPattern.create('orders/created')
      expect(result.isErr()).toBe(true)
      expect(result.error).toBe('INVALID_CHARACTERS')
    })

    it('rejects consecutive dots', () => {
      const result = SubjectPattern.create('orders..created')
      expect(result.isErr()).toBe(true)
      expect(result.error).toBe('CONSECUTIVE_DOTS')
    })

    it('rejects trailing dot', () => {
      const result = SubjectPattern.create('orders.created.')
      expect(result.isErr()).toBe(true)
      expect(result.error).toBe('TRAILING_DOT')
    })
  })

  describe('fromTrusted', () => {
    it('creates without validation', () => {
      const pattern = SubjectPattern.fromTrusted('orders.*')
      expect(pattern.value).toBe('orders.*')
      expect(pattern.hasWildcards).toBe(true)
    })
  })

  describe('matches', () => {
    it('matches exact subject', () => {
      const pattern = SubjectPattern.fromTrusted('orders.created')
      expect(pattern.matches('orders.created')).toBe(true)
    })

    it('matches single wildcard', () => {
      const pattern = SubjectPattern.fromTrusted('orders.*')
      expect(pattern.matches('orders.created')).toBe(true)
      expect(pattern.matches('orders.updated')).toBe(true)
    })

    it('does not match wrong depth with *', () => {
      const pattern = SubjectPattern.fromTrusted('orders.*')
      expect(pattern.matches('orders.created.123')).toBe(false)
    })

    it('matches multi-token wildcard >', () => {
      const pattern = SubjectPattern.fromTrusted('orders.>')
      expect(pattern.matches('orders.created')).toBe(true)
      expect(pattern.matches('orders.created.123')).toBe(true)
    })

    it('does not match different prefix', () => {
      const pattern = SubjectPattern.fromTrusted('orders.*')
      expect(pattern.matches('users.created')).toBe(false)
    })

    it('matches with multiple wildcards', () => {
      const pattern = SubjectPattern.fromTrusted('*.created.*')
      expect(pattern.matches('orders.created.123')).toBe(true)
      expect(pattern.matches('users.created.456')).toBe(true)
    })

    it('does not match shorter subject', () => {
      const pattern = SubjectPattern.fromTrusted('orders.created.*')
      expect(pattern.matches('orders.created')).toBe(false)
    })
  })

  describe('segmentCount', () => {
    it('counts segments', () => {
      expect(SubjectPattern.fromTrusted('orders.*.created').segmentCount()).toBe(3)
      expect(SubjectPattern.fromTrusted('orders').segmentCount()).toBe(1)
    })
  })

  describe('wildcardCount', () => {
    it('counts wildcards', () => {
      expect(SubjectPattern.fromTrusted('orders.*.>').wildcardCount()).toBe(2)
      expect(SubjectPattern.fromTrusted('orders.created').wildcardCount()).toBe(0)
    })
  })

  describe('specificity', () => {
    it('literal pattern is more specific than wildcard', () => {
      const literal = SubjectPattern.fromTrusted('orders.created')
      const wildcard = SubjectPattern.fromTrusted('orders.*')
      expect(literal.specificity()).toBeGreaterThan(wildcard.specificity())
    })

    it('* is more specific than >', () => {
      const star = SubjectPattern.fromTrusted('orders.*')
      const chevron = SubjectPattern.fromTrusted('orders.>')
      expect(star.specificity()).toBeGreaterThan(chevron.specificity())
    })
  })

  describe('isMoreSpecificThan', () => {
    it('compares specificity', () => {
      const specific = SubjectPattern.fromTrusted('orders.created.done')
      const general = SubjectPattern.fromTrusted('orders.>')
      expect(specific.isMoreSpecificThan(general)).toBe(true)
      expect(general.isMoreSpecificThan(specific)).toBe(false)
    })
  })

  describe('basePrefix', () => {
    it('returns non-wildcard prefix', () => {
      expect(SubjectPattern.fromTrusted('orders.created.>').basePrefix()).toBe('orders.created')
    })

    it('returns empty for leading wildcard', () => {
      expect(SubjectPattern.fromTrusted('*.created').basePrefix()).toBe('')
    })

    it('returns full pattern if no wildcards', () => {
      expect(SubjectPattern.fromTrusted('orders.created').basePrefix()).toBe('orders.created')
    })
  })

  describe('looksLikeSpecificSubject', () => {
    it('detects UUID in pattern', () => {
      const pattern = SubjectPattern.fromTrusted('orders.550e8400-e29b-41d4-a716-446655440000')
      expect(pattern.looksLikeSpecificSubject()).toBe(true)
    })

    it('returns false for pattern without UUID', () => {
      const pattern = SubjectPattern.fromTrusted('orders.*.created')
      expect(pattern.looksLikeSpecificSubject()).toBe(false)
    })
  })

  describe('toString', () => {
    it('returns the value', () => {
      expect(SubjectPattern.fromTrusted('orders.*').toString()).toBe('orders.*')
    })
  })
})
