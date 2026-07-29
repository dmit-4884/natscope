import { describe, it, expect } from 'vitest'
import { ProtoField } from './ProtoField'

describe('ProtoField', () => {
  describe('fromApi', () => {
    it('creates field with all properties', () => {
      const field = ProtoField.fromApi({
        name: 'user_id',
        type: 'string',
        number: 1,
        label: 'required',
        is_message: false,
      })
      expect(field.name).toBe('user_id')
      expect(field.type).toBe('string')
      expect(field.number).toBe(1)
      expect(field.label).toBe('required')
      expect(field.isMessage).toBe(false)
    })

    it('defaults label to optional', () => {
      const field = ProtoField.fromApi({ name: 'id', type: 'int32', number: 1 })
      expect(field.label).toBe('optional')
    })

    it('defaults isMessage to false', () => {
      const field = ProtoField.fromApi({ name: 'id', type: 'int32', number: 1 })
      expect(field.isMessage).toBe(false)
    })
  })

  describe('isRepeated', () => {
    it('returns true for repeated fields', () => {
      const field = ProtoField.fromApi({ name: 'tags', type: 'string', number: 1, label: 'repeated' })
      expect(field.isRepeated()).toBe(true)
    })

    it('returns false for non-repeated fields', () => {
      const field = ProtoField.fromApi({ name: 'id', type: 'string', number: 1, label: 'optional' })
      expect(field.isRepeated()).toBe(false)
    })
  })

  describe('isRequired', () => {
    it('returns true for required fields', () => {
      const field = ProtoField.fromApi({ name: 'id', type: 'string', number: 1, label: 'required' })
      expect(field.isRequired()).toBe(true)
    })

    it('returns false for optional fields', () => {
      const field = ProtoField.fromApi({ name: 'id', type: 'string', number: 1 })
      expect(field.isRequired()).toBe(false)
    })
  })

  describe('isScalar', () => {
    it.each([
      'double', 'float', 'int32', 'int64', 'uint32', 'uint64',
      'sint32', 'sint64', 'fixed32', 'fixed64', 'sfixed32', 'sfixed64',
      'bool', 'string', 'bytes',
    ])('returns true for %s', (type) => {
      const field = ProtoField.fromApi({ name: 'f', type, number: 1 })
      expect(field.isScalar()).toBe(true)
    })

    it('returns false for message type', () => {
      const field = ProtoField.fromApi({ name: 'user', type: 'my.User', number: 1, is_message: true })
      expect(field.isScalar()).toBe(false)
    })

    it('returns false for enum type', () => {
      const field = ProtoField.fromApi({ name: 'status', type: 'Status', number: 1 })
      expect(field.isScalar()).toBe(false)
    })
  })

  describe('typeDisplayName', () => {
    it('returns type for non-repeated', () => {
      const field = ProtoField.fromApi({ name: 'id', type: 'string', number: 1 })
      expect(field.typeDisplayName()).toBe('string')
    })

    it('prefixes repeated', () => {
      const field = ProtoField.fromApi({ name: 'tags', type: 'string', number: 1, label: 'repeated' })
      expect(field.typeDisplayName()).toBe('repeated string')
    })
  })
})
