import { describe, it, expect } from 'vitest'
import { ProtoMessage } from './ProtoMessage'

describe('ProtoMessage', () => {
  const apiData = {
    full_name: 'my.package.OrderEvent',
    proto_file: 'order/order_event.proto',
    package: 'my.package',
    source_id: 'src-A',
    source_tag: 'v1.0.0',
    fields: [
      { name: 'order_id', type: 'string', number: 1, label: 'optional', is_message: false },
      { name: 'amount', type: 'double', number: 2 },
      { name: 'items', type: 'OrderItem', number: 3, label: 'repeated', is_message: true },
    ],
  }

  describe('fromApi', () => {
    it('maps all fields including source identity', () => {
      const msg = ProtoMessage.fromApi(apiData)
      // Identity is now (sourceId, fullName) — same FQN in two sources is two entities.
      expect(msg.id).toBe('src-A|my.package.OrderEvent')
      expect(msg.fullName).toBe('my.package.OrderEvent')
      expect(msg.protoFile).toBe('order/order_event.proto')
      expect(msg.packageName).toBe('my.package')
      expect(msg.sourceId).toBe('src-A')
      expect(msg.sourceTag).toBe('v1.0.0')
      expect(msg.fields).toHaveLength(3)
    })

    it('converts fields correctly', () => {
      const msg = ProtoMessage.fromApi(apiData)
      expect(msg.fields[0].name).toBe('order_id')
      expect(msg.fields[2].isMessage).toBe(true)
      expect(msg.fields[2].label).toBe('repeated')
    })
  })

  describe('fieldCount', () => {
    it('returns number of fields', () => {
      expect(ProtoMessage.fromApi(apiData).fieldCount).toBe(3)
    })

    it('returns 0 for empty fields', () => {
      expect(ProtoMessage.fromApi({ ...apiData, fields: [] }).fieldCount).toBe(0)
    })
  })

  describe('shortName', () => {
    it('returns just the message name without package', () => {
      expect(ProtoMessage.fromApi(apiData).shortName()).toBe('OrderEvent')
    })

    it('handles simple name without dots', () => {
      const msg = ProtoMessage.fromApi({ ...apiData, full_name: 'Simple' })
      expect(msg.shortName()).toBe('Simple')
    })
  })

  describe('equality', () => {
    it('two messages with same (sourceId, fullName) are equal', () => {
      const msg1 = ProtoMessage.fromApi(apiData)
      const msg2 = ProtoMessage.fromApi(apiData)
      expect(msg1.equals(msg2)).toBe(true)
    })

    it('two messages with different fullName are not equal', () => {
      const msg1 = ProtoMessage.fromApi(apiData)
      const msg2 = ProtoMessage.fromApi({ ...apiData, full_name: 'other.Type' })
      expect(msg1.equals(msg2)).toBe(false)
    })

    it('two messages with same fullName but different source are NOT equal', () => {
      const msg1 = ProtoMessage.fromApi(apiData)
      const msg2 = ProtoMessage.fromApi({ ...apiData, source_id: 'src-B' })
      expect(msg1.equals(msg2)).toBe(false)
    })
  })
})
