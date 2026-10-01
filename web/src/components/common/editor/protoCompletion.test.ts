import { describe, it, expect } from 'vitest'
import { EditorState } from '@codemirror/state'
import { json } from '@codemirror/lang-json'
import { ensureSyntaxTree } from '@codemirror/language'
import { CompletionContext, type CompletionResult } from '@codemirror/autocomplete'
import type { SchemaField, TypeDescription } from '@/api/proto'
import { protoCompletionSource } from './protoCompletion'

const field = (f: Partial<SchemaField> & { name: string; kind: string }): SchemaField => ({
  jsonName: f.name,
  number: 1,
  typeName: '',
  repeated: false,
  mapKey: '',
  optional: false,
  required: false,
  oneof: '',
  deprecated: false,
  comment: '',
  ...f,
})

const description: TypeDescription = {
  messages: [
    {
      fullName: 'shop.Order',
      file: 'shop.proto',
      comment: '',
      deprecated: false,
      fields: [
        field({ name: 'order_id', jsonName: 'orderId', kind: 'string', comment: 'Order id.' }),
        field({ name: 'status', kind: 'enum', typeName: 'shop.Status' }),
        field({ name: 'items', kind: 'message', typeName: 'shop.Item', repeated: true }),
        field({ name: 'by_sku', kind: 'message', typeName: 'shop.Item', mapKey: 'string' }),
        field({ name: 'created_at', kind: 'message', typeName: 'google.protobuf.Timestamp' }),
        field({ name: 'paid', kind: 'bool' }),
        field({ name: 'legacy', kind: 'string', deprecated: true }),
      ],
    },
    {
      fullName: 'shop.Item',
      file: 'shop.proto',
      comment: '',
      deprecated: false,
      fields: [field({ name: 'sku', kind: 'string' }), field({ name: 'state', kind: 'enum', typeName: 'shop.Status' })],
    },
    {
      fullName: 'google.protobuf.Timestamp',
      file: 'google/protobuf/timestamp.proto',
      comment: '',
      deprecated: false,
      fields: [field({ name: 'seconds', kind: 'int64' })],
    },
  ],
  enums: [
    {
      fullName: 'shop.Status',
      file: 'shop.proto',
      comment: '',
      deprecated: false,
      values: [
        { name: 'STATUS_UNSPECIFIED', number: 0, comment: '', deprecated: false },
        { name: 'STATUS_PAID', number: 1, comment: 'Paid in full.', deprecated: false },
      ],
    },
  ],
  services: [],
}

const source = protoCompletionSource(description, 'shop.Order')

function complete(doc: string): string[] {
  const pos = doc.indexOf('|')
  const text = doc.replace('|', '')
  const state = EditorState.create({ doc: text, extensions: [json()] })
  ensureSyntaxTree(state, text.length, 1e9)
  const result = source(new CompletionContext(state, pos, true)) as CompletionResult | null
  return result ? result.options.map((o) => o.label) : []
}

describe('protoCompletionSource', () => {
  it('suggests the root fields in an empty object', () => {
    expect(complete('{|}')).toEqual(['order_id', 'status', 'items', 'by_sku', 'created_at', 'paid', 'legacy'])
  })

  it('skips keys the object already has, by proto or JSON name', () => {
    expect(complete('{"orderId": "1", "paid": true, "|"}')).toEqual([
      'status',
      'items',
      'by_sku',
      'created_at',
      'legacy',
    ])
  })

  it('follows nested objects through repeated fields', () => {
    expect(complete('{"items": [{"sku": "a"}, {|}]}')).toEqual(['sku', 'state'])
  })

  it('follows map values, not map keys', () => {
    expect(complete('{"by_sku": {"a1": {|}}}')).toEqual(['sku', 'state'])
    expect(complete('{"by_sku": {|}}')).toEqual([])
  })

  it('leaves well-known types alone', () => {
    expect(complete('{"created_at": {|}}')).toEqual([])
  })

  it('suggests enum values for an enum field, also inside arrays and nested messages', () => {
    expect(complete('{"status": |}')).toEqual(['STATUS_UNSPECIFIED', 'STATUS_PAID'])
    expect(complete('{"status": "STATUS_|"}')).toEqual(['STATUS_UNSPECIFIED', 'STATUS_PAID'])
    expect(complete('{"items": [{"state": "|"}]}')).toEqual(['STATUS_UNSPECIFIED', 'STATUS_PAID'])
  })

  it('suggests true and false for a bool field', () => {
    expect(complete('{"paid": |}')).toEqual(['true', 'false'])
  })

  it('stays quiet in plain values and unknown fields', () => {
    expect(complete('{"order_id": |}')).toEqual([])
    expect(complete('{"nope": {|}}')).toEqual([])
  })
})
