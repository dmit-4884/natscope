import { describe, expect, it } from 'vitest'
import type { Message } from '@/types/nats'
import { createExportSerializer, type ExportSerializeOptions } from './exportFormat'

function msg(seq: number, data: unknown, extra: Partial<Message> = {}): Message {
  return {
    sequence: seq,
    subject: `s.${seq}`,
    timestamp: 1000 + seq,
    data_base64: btoa(JSON.stringify(data)),
    data_size: 1,
    content_type: 'json',
    ...extra,
  }
}

const base: ExportSerializeOptions = { format: 'json', includeMetadata: true, includeHeaders: false, includeDecoded: true }

function run(options: ExportSerializeOptions, pages: Message[][]): string {
  const s = createExportSerializer(options)
  return s.begin() + pages.map((p) => s.chunk(p)).join('') + s.end()
}

describe('createExportSerializer', () => {
  const pages = [[msg(1, { a: 1 }), msg(2, { b: [1, 2] })], [msg(3, 'x')]]

  it('builds the same pretty JSON array whatever the paging', () => {
    const all = pages.flat()
    const expected = JSON.stringify(
      all.map((m) => ({ sequence: m.sequence, subject: m.subject, timestamp: m.timestamp, data: JSON.parse(atob(m.data_base64)) })),
      null,
      2,
    )
    expect(run(base, pages)).toBe(expected)
    expect(run(base, [all])).toBe(expected)
  })

  it('writes an empty JSON array as []', () => {
    expect(run(base, [])).toBe('[]')
  })

  it('writes one JSON object per line for ndjson, ending with a newline', () => {
    const out = run({ ...base, format: 'ndjson' }, pages)
    const lines = out.split('\n')
    expect(lines).toHaveLength(4)
    expect(lines[3]).toBe('')
    expect(JSON.parse(lines[0])).toMatchObject({ sequence: 1, data: { a: 1 } })
  })

  it('writes a csv header once, then a row per message, formula-safe', () => {
    const out = run({ ...base, format: 'csv' }, [[msg(1, 'x', { subject: '=cmd' })], [msg(2, 'y')]])
    const lines = out.split('\n').filter(Boolean)
    expect(lines[0]).toBe('sequence,subject,timestamp,data')
    expect(lines).toHaveLength(3)
    expect(lines[1]).toContain(`'`)
  })

  it('includes headers only when asked', () => {
    const withHeaders = run({ ...base, format: 'ndjson', includeHeaders: true }, [[msg(1, 1, { headers: { k: 'v' } })]])
    expect(JSON.parse(withHeaders.trim())).toMatchObject({ headers: { k: 'v' } })
    const without = run({ ...base, format: 'ndjson' }, [[msg(1, 1, { headers: { k: 'v' } })]])
    expect(JSON.parse(without.trim())).not.toHaveProperty('headers')
  })

  it('reports the mime type and extension of the format', () => {
    expect(createExportSerializer({ ...base, format: 'ndjson' })).toMatchObject({ mime: 'application/x-ndjson', extension: 'ndjson' })
    expect(createExportSerializer({ ...base, format: 'csv' })).toMatchObject({ mime: 'text/csv', extension: 'csv' })
    expect(createExportSerializer(base)).toMatchObject({ mime: 'application/json', extension: 'json' })
  })
})
