import { describe, it, expect } from 'vitest'
import { MessageFetchPolicy, MESSAGE_FETCH_POLICY_DEFAULTS } from './MessageFetchPolicy'

describe('MessageFetchPolicy', () => {
  it('default() returns the defaults', () => {
    const p = MessageFetchPolicy.default()
    expect(p.toObject()).toEqual(MESSAGE_FETCH_POLICY_DEFAULTS)
  })

  it('fromPartial fills in defaults for missing fields', () => {
    const p = MessageFetchPolicy.fromPartial({ defaultPageSize: 100 })
    expect(p.defaultPageSize).toBe(100)
    expect(p.fetchMethod).toBe('consumer')
  })

  it('fromPartial falls back to default for unknown enum values', () => {
    const p = MessageFetchPolicy.fromPartial({ fetchMethod: 'garbage' })
    expect(p.fetchMethod).toBe('consumer')
  })

  it('create rejects an invalid fetchMethod', () => {
    const r = MessageFetchPolicy.create({ fetchMethod: 'garbage' as 'direct' })
    expect(r.isErr()).toBe(true)
    expect(r.error.field).toBe('fetchMethod')
  })

  it('create rejects out-of-range defaultPageSize', () => {
    const r = MessageFetchPolicy.create({ defaultPageSize: 0 })
    expect(r.isErr()).toBe(true)
    expect(r.error.field).toBe('defaultPageSize')
  })

  it('merge validates and returns Ok on good input', () => {
    const p = MessageFetchPolicy.default()
    const r = p.merge({ fetchMethod: 'consumer' })
    expect(r.isOk()).toBe(true)
    if (r.isOk()) expect(r.value.fetchMethod).toBe('consumer')
  })

  it('merge rejects bad enum', () => {
    const p = MessageFetchPolicy.default()
    const r = p.merge({ fetchMethod: 'nope' })
    expect(r.isErr()).toBe(true)
  })

  describe('export fields', () => {
    it('default export format is json with the standard range limit', () => {
      const p = MessageFetchPolicy.default()
      expect(p.defaultExportFormat).toBe('json')
      expect(p.exportRangeLimit).toBe(50000)
    })

    it('fromPartial accepts a valid export format', () => {
      const p = MessageFetchPolicy.fromPartial({ defaultExportFormat: 'csv' })
      expect(p.defaultExportFormat).toBe('csv')
    })

    it('fromPartial falls back to json for an unknown export format', () => {
      const p = MessageFetchPolicy.fromPartial({ defaultExportFormat: 'xml' })
      expect(p.defaultExportFormat).toBe('json')
    })

    it('fromPartial coerces a zero/negative range limit to the default', () => {
      expect(MessageFetchPolicy.fromPartial({ exportRangeLimit: 0 }).exportRangeLimit).toBe(50000)
      expect(MessageFetchPolicy.fromPartial({ exportRangeLimit: -5 }).exportRangeLimit).toBe(50000)
    })

    it('create rejects an invalid export format', () => {
      const r = MessageFetchPolicy.create({ defaultExportFormat: 'xml' as 'json' })
      expect(r.isErr()).toBe(true)
      expect(r.error.field).toBe('defaultExportFormat')
    })

    it('create rejects a non-positive export range limit', () => {
      const r = MessageFetchPolicy.create({ exportRangeLimit: 0 })
      expect(r.isErr()).toBe(true)
      expect(r.error.field).toBe('exportRangeLimit')
    })
  })
})
