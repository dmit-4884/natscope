import { describe, it, expect } from 'vitest'
import type { ConsumersOverview } from '@/api/stats'
import type { ConsumerInfo, StreamInfo } from '@/types/nats'
import { buildRows, filterRows, firstDirection, formatAgo, rowsToCsv, rowsToJson, sortRows } from './consumersOverview'

const NOW = Date.UTC(2026, 9, 5, 12, 0, 0)

function consumer(name: string, stream: string, overrides: Partial<ConsumerInfo> = {}): ConsumerInfo {
  return {
    name,
    stream_name: stream,
    num_pending: 0,
    num_ack_pending: 0,
    config: { durable_name: name, ack_policy: 'explicit', max_ack_pending: 100 },
    delivered: { consumer_seq: 1, stream_seq: 1, last_active: NOW - 2_000 },
    ack_floor: { consumer_seq: 1, stream_seq: 1 },
    ...overrides,
  }
}

function stream(name: string, maxMsgs = -1, messages = 10): StreamInfo {
  return {
    name,
    subjects: [`${name.toLowerCase()}.>`],
    messages,
    bytes: 100,
    consumer_count: 1,
    created: NOW,
    config: { retention: 'limits', max_msgs: maxMsgs, max_bytes: -1, max_age: 0, discard: 'old' },
    state: { messages, bytes: 100, first_seq: 2, last_seq: messages + 1, first_ts: NOW, last_ts: NOW },
  }
}

const overview: ConsumersOverview = {
  consumers: [
    consumer('archiver', 'ORDERS'),
    consumer('billing', 'ORDERS', { num_pending: 40, num_ack_pending: 100, config: { durable_name: 'billing', max_ack_pending: 100, filter_subject: 'orders.created' } }),
    consumer('mailer', 'EVENTS', { num_pending: 7, num_waiting: 2 }),
    consumer('ghost', 'EVENTS', { num_pending: 3, num_waiting: 1, config: { ack_policy: 'explicit' } }),
  ],
  streams: [stream('ORDERS'), stream('EVENTS', 10, 10)],
  unreadable: [],
}

describe('buildRows', () => {
  it('joins each consumer with its stream and sorts the troubled ones first', () => {
    const rows = buildRows(overview, NOW)
    expect(rows.map((r) => r.consumer.name)).toEqual(['billing', 'ghost', 'mailer', 'archiver'])
    expect(rows[0].state).toBe('stuck')
    expect(rows[1].issues.map((i) => i.kind)).toEqual(['stream_full'])
    expect(rows[0].stream?.name).toBe('ORDERS')
  })
})

describe('filterRows', () => {
  const rows = buildRows(overview, NOW)

  it('matches the consumer, the stream or a filter subject, ignoring case', () => {
    expect(filterRows(rows, 'BILL', false).map((r) => r.consumer.name)).toEqual(['billing'])
    expect(filterRows(rows, 'events', false).map((r) => r.consumer.name)).toEqual(['ghost', 'mailer'])
    expect(filterRows(rows, 'orders.created', false).map((r) => r.consumer.name)).toEqual(['billing'])
  })

  it('keeps only consumers with problems on request', () => {
    expect(filterRows(rows, '', true).map((r) => r.consumer.name)).toEqual(['billing', 'ghost', 'mailer'])
  })
})

describe('exports', () => {
  const rows = buildRows(overview, NOW).slice(0, 1)

  it('writes a CSV row per consumer with its problems', () => {
    const csv = rowsToCsv(rows).split('\n')
    expect(csv[0]).toBe('stream,consumer,type,durable,filter_subjects,pending,ack_pending,max_ack_pending,redelivered,waiting,last_delivery,status,problems')
    expect(csv[1]).toBe(`ORDERS,billing,pull,yes,orders.created,40,100,100,0,0,${new Date(NOW - 2_000).toISOString()},stuck,Ack limit reached`)
  })

  it('defuses text a spreadsheet would run as a formula', () => {
    const names = ['=HYPERLINK("http://x")', '+1', '-2', '@cmd', '\tx']
    const risky = buildRows({ ...overview, consumers: names.map((n) => consumer(n, 'ORDERS')) }, NOW)
    const cells = rowsToCsv(risky).split('\n').slice(1).map((line) => line.split(',')[1])
    expect(cells.sort()).toEqual([`"'=HYPERLINK(""http://x"")"`, `'+1`, `'-2`, `'@cmd`, `'\tx`].sort())
  })

  it('quotes CSV fields that need it', () => {
    const odd = buildRows({ ...overview, consumers: [consumer('a,"b"', 'ORDERS')] }, NOW)
    expect(rowsToCsv(odd).split('\n')[1].startsWith('ORDERS,"a,""b""",')).toBe(true)
  })

  it('writes JSON with the explained problems', () => {
    const json = JSON.parse(rowsToJson(rows))
    expect(json[0]).toMatchObject({ stream: 'ORDERS', consumer: 'billing', status: 'stuck', pending: 40 })
    expect(json[0].problems[0]).toMatchObject({ kind: 'ack_limit', severity: 'error' })
    expect(json[0].problems[0].detail).toContain('max ack pending')
  })
})

describe('formatAgo', () => {
  it('says how long ago in the largest whole unit', () => {
    expect(formatAgo(NOW - 2_000, NOW)).toBe('just now')
    expect(formatAgo(NOW - 42_000, NOW)).toBe('42s ago')
    expect(formatAgo(NOW - 5 * 60_000, NOW)).toBe('5m ago')
    expect(formatAgo(NOW - 3 * 3_600_000, NOW)).toBe('3h ago')
    expect(formatAgo(NOW - 2 * 86_400_000, NOW)).toBe('2d ago')
    expect(formatAgo(undefined, NOW)).toBe('never')
  })
})

describe('sortRows', () => {
  const rows = buildRows(overview, NOW)
  const names = (sorted: typeof rows) => sorted.map((r) => r.consumer.name)

  it('keeps the health order when sorting by status, and reverses it', () => {
    expect(names(sortRows(rows, { key: 'status', direction: 'asc' }))).toEqual(['billing', 'ghost', 'mailer', 'archiver'])
    expect(names(sortRows(rows, { key: 'status', direction: 'desc' }))).toEqual(['archiver', 'mailer', 'ghost', 'billing'])
  })

  it('sorts by pending messages either way', () => {
    expect(names(sortRows(rows, { key: 'pending', direction: 'desc' }))).toEqual(['billing', 'mailer', 'ghost', 'archiver'])
    expect(names(sortRows(rows, { key: 'pending', direction: 'asc' }))).toEqual(['archiver', 'ghost', 'mailer', 'billing'])
  })

  it('sorts by consumer name', () => {
    expect(names(sortRows(rows, { key: 'consumer', direction: 'asc' }))).toEqual(['archiver', 'billing', 'ghost', 'mailer'])
  })

  it('breaks ties in the health order whatever the direction', () => {
    expect(names(sortRows(rows, { key: 'ack', direction: 'desc' }))).toEqual(['billing', 'ghost', 'mailer', 'archiver'])
    expect(names(sortRows(rows, { key: 'ack', direction: 'asc' }))).toEqual(['ghost', 'mailer', 'archiver', 'billing'])
  })

  it('sorts by redeliveries', () => {
    const redelivered = buildRows(
      { ...overview, consumers: [consumer('once', 'ORDERS', { num_redelivered: 1 }), consumer('often', 'ORDERS', { num_redelivered: 9 }), consumer('never', 'ORDERS')] },
      NOW,
    )
    expect(names(sortRows(redelivered, { key: 'redelivered', direction: 'desc' }))).toEqual(['often', 'once', 'never'])
  })

  it('puts a consumer that never delivered after the latest deliveries', () => {
    const delivered = buildRows(
      {
        ...overview,
        consumers: [
          consumer('idle', 'ORDERS', { delivered: { consumer_seq: 0, stream_seq: 0 } }),
          consumer('recent', 'ORDERS', { delivered: { consumer_seq: 1, stream_seq: 1, last_active: NOW - 1_000 } }),
          consumer('stale', 'ORDERS', { delivered: { consumer_seq: 1, stream_seq: 1, last_active: NOW - 60_000 } }),
        ],
      },
      NOW,
    )
    expect(names(sortRows(delivered, { key: 'last', direction: 'desc' }))).toEqual(['recent', 'stale', 'idle'])
    expect(names(sortRows(delivered, { key: 'last', direction: 'asc' }))).toEqual(['idle', 'stale', 'recent'])
  })

  it('starts counts and times with the largest and names with A', () => {
    expect(firstDirection('pending')).toBe('desc')
    expect(firstDirection('last')).toBe('desc')
    expect(firstDirection('consumer')).toBe('asc')
    expect(firstDirection('status')).toBe('asc')
  })
})
