import { describe, it, expect } from 'vitest'
import { Mapping } from './Mapping'

describe('Mapping', () => {
  const apiData = {
    id: 'mapping-1',
    pattern: 'orders.*.created',
    message_type: 'my.package.OrderCreated',
    source_id: 'src-A',
    created_at: 1700000000000,
    updated_at: 1700000001000,
  }

  describe('fromApi', () => {
    it('maps all fields including sourceId', () => {
      const mapping = Mapping.fromApi(apiData)
      expect(mapping.id).toBe('mapping-1')
      expect(mapping.pattern.value).toBe('orders.*.created')
      expect(mapping.messageType).toBe('my.package.OrderCreated')
      expect(mapping.sourceId).toBe('src-A')
      expect(mapping.createdAt).toEqual(new Date(1700000000000))
      expect(mapping.updatedAt).toEqual(new Date(1700000001000))
    })

    it('handles missing updated_at', () => {
      const mapping = Mapping.fromApi({ ...apiData, updated_at: undefined })
      expect(mapping.updatedAt).toBeNull()
    })
  })

  describe('createNew', () => {
    it('creates with empty id and provided sourceId', () => {
      const mapping = Mapping.createNew('orders.*', 'OrderEvent', 'src-A')
      expect(mapping.id).toBe('')
      expect(mapping.pattern.value).toBe('orders.*')
      expect(mapping.messageType).toBe('OrderEvent')
      expect(mapping.sourceId).toBe('src-A')
      expect(mapping.updatedAt).toBeNull()
    })

    it('throws when sourceId is empty', () => {
      expect(() => Mapping.createNew('orders.*', 'X', '')).toThrow(/sourceId is required/)
    })
  })

  describe('matches', () => {
    it('delegates to pattern matching', () => {
      const mapping = Mapping.fromApi(apiData)
      expect(mapping.matches('orders.123.created')).toBe(true)
      expect(mapping.matches('orders.456.updated')).toBe(false)
    })
  })

  describe('looksLikeSpecificSubject', () => {
    it('detects UUID in pattern', () => {
      const mapping = Mapping.fromApi({
        ...apiData,
        pattern: 'orders.550e8400-e29b-41d4-a716-446655440000.created',
      })
      expect(mapping.looksLikeSpecificSubject()).toBe(true)
    })

    it('returns false for wildcard pattern', () => {
      const mapping = Mapping.fromApi(apiData)
      expect(mapping.looksLikeSpecificSubject()).toBe(false)
    })
  })

  describe('equality', () => {
    it('two mappings with same id are equal', () => {
      const m1 = Mapping.fromApi(apiData)
      const m2 = Mapping.fromApi(apiData)
      expect(m1.equals(m2)).toBe(true)
    })

    it('two mappings with different ids are not equal', () => {
      const m1 = Mapping.fromApi(apiData)
      const m2 = Mapping.fromApi({ ...apiData, id: 'other' })
      expect(m1.equals(m2)).toBe(false)
    })
  })

  describe('withHealth', () => {
    it('returns a new mapping with health set', () => {
      const m = Mapping.fromApi(apiData)
      const withOk = m.withHealth('ok')
      expect(withOk.health).toBe('ok')
      expect(m.health).toBeUndefined()
    })

    it('preserves detail when supplied', () => {
      const m = Mapping.fromApi(apiData).withHealth('selection_missing', 'no version selected')
      expect(m.health).toBe('selection_missing')
      expect(m.healthDetail).toBe('no version selected')
    })
  })
})
