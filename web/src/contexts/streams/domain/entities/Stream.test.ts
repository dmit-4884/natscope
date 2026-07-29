import { describe, it, expect } from 'vitest'
import { Stream } from './Stream'

describe('Stream', () => {
  const apiData = {
    name: 'ORDERS',
    description: 'Order events',
    subjects: ['orders.>'],
    messages: 1000,
    bytes: 1048576,
    consumer_count: 3,
    created: 1700000000000,
    config: {
      retention: 'limits',
      max_msgs: 10000,
      max_bytes: 104857600,
      max_age: 86400000000000,
      storage: 'file',
      num_replicas: 1,
    },
  }

  describe('fromApi', () => {
    it('creates stream with basic fields', () => {
      const stream = Stream.fromApi(apiData)
      expect(stream.id).toBe('ORDERS')
      expect(stream.name.value).toBe('ORDERS')
      expect(stream.description).toBe('Order events')
      expect(stream.subjects).toEqual(['orders.>'])
      expect(stream.created).toEqual(new Date(1700000000000))
    })

    it('creates config correctly', () => {
      const stream = Stream.fromApi(apiData)
      expect(stream.config.retention).toBe('limits')
      expect(stream.config.maxMessages).toBe(10000)
      expect(stream.config.storage).toBe('file')
    })

    it('uses state fields when state object is provided', () => {
      const stream = Stream.fromApi({
        ...apiData,
        state: {
          messages: 500,
          bytes: 524288,
          first_seq: 1,
          last_seq: 500,
          first_ts: 1700000000000,
          last_ts: 1700000001000,
          consumer_count: 5,
        },
      })
      expect(stream.messageCount).toBe(500)
      expect(stream.bytes).toBe(524288)
      expect(stream.firstSequence).toBe(1)
      expect(stream.lastSequence).toBe(500)
      expect(stream.consumerCount).toBe(5)
    })

    it('falls back to top-level fields when no state', () => {
      const stream = Stream.fromApi(apiData)
      expect(stream.messageCount).toBe(1000)
      expect(stream.bytes).toBe(1048576)
      expect(stream.consumerCount).toBe(3)
      expect(stream.firstSequence).toBe(0)
      expect(stream.lastSequence).toBe(0)
    })

    it('defaults description to empty string', () => {
      const stream = Stream.fromApi({ ...apiData, description: undefined })
      expect(stream.description).toBe('')
    })
  })

  describe('equality', () => {
    it('streams with same name are equal', () => {
      const s1 = Stream.fromApi(apiData)
      const s2 = Stream.fromApi(apiData)
      expect(s1.equals(s2)).toBe(true)
    })

    it('streams with different names are not equal', () => {
      const s1 = Stream.fromApi(apiData)
      const s2 = Stream.fromApi({ ...apiData, name: 'USERS' })
      expect(s1.equals(s2)).toBe(false)
    })
  })
})
