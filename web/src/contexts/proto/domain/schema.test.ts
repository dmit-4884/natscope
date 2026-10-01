import { describe, expect, it } from 'vitest'
import type { SchemaField, SchemaType } from '@/api/proto'
import { fieldTypeLabel, groupByPackage, messageCompletions, shortTypeName } from './schema'

const field = (overrides: Partial<SchemaField>): SchemaField => ({
  name: 'f',
  jsonName: 'f',
  number: 1,
  kind: 'string',
  typeName: '',
  repeated: false,
  mapKey: '',
  optional: false,
  required: false,
  oneof: '',
  deprecated: false,
  comment: '',
  ...overrides,
})

const type = (fullName: string, packageName: string): SchemaType => ({
  id: `src|${fullName}`,
  fullName,
  kind: 'message',
  file: 'a.proto',
  packageName,
  comment: '',
  memberCount: 0,
  dependency: false,
  sourceId: 'src',
  sourceRevision: 'local',
})

describe('schema helpers', () => {
  it('shortens full names', () => {
    expect(shortTypeName('shop.v1.Order')).toBe('Order')
    expect(shortTypeName('Order')).toBe('Order')
  })

  it('labels scalar, repeated, message and map fields', () => {
    expect(fieldTypeLabel(field({ kind: 'int64' }))).toBe('int64')
    expect(fieldTypeLabel(field({ repeated: true }))).toBe('repeated string')
    expect(fieldTypeLabel(field({ kind: 'message', typeName: 'shop.Item', repeated: true }))).toBe('repeated shop.Item')
    expect(fieldTypeLabel(field({ kind: 'enum', typeName: 'shop.Status', mapKey: 'string' }))).toBe('map<string, shop.Status>')
  })

  it('builds editor completions from the root message', () => {
    const completions = messageCompletions({
      fullName: 'shop.Order',
      file: 'a.proto',
      comment: '',
      deprecated: false,
      fields: [field({ name: 'items', kind: 'message', typeName: 'shop.Item', repeated: true })],
    })
    expect(completions).toEqual([{ name: 'items', type: 'repeated shop.Item', repeated: true, isMessage: true }])
  })

  it('groups types by package in name order', () => {
    const groups = groupByPackage([type('b.Z', 'b'), type('a.Y', 'a'), type('a.X', 'a'), type('Root', '')])
    expect(groups.map(([pkg, list]) => [pkg, list.map((t) => t.fullName)])).toEqual([
      ['(no package)', ['Root']],
      ['a', ['a.X', 'a.Y']],
      ['b', ['b.Z']],
    ])
  })
})
