import { describe, it, expect } from 'vitest'
import type { ConsumerInfo, StreamInfo } from '@/types/nats'
import { compareByHealth, consumerIssues, consumerKind, consumerState } from './consumerHealth'

const NOW = Date.UTC(2026, 9, 5, 12, 0, 0)

function consumer(overrides: Partial<ConsumerInfo> = {}): ConsumerInfo {
  return {
    name: 'worker',
    stream_name: 'ORDERS',
    num_pending: 0,
    num_ack_pending: 0,
    config: { ack_policy: 'explicit', max_ack_pending: 1000, ack_wait: 30_000_000_000 },
    delivered: { consumer_seq: 10, stream_seq: 10, last_active: NOW - 1_000 },
    ack_floor: { consumer_seq: 10, stream_seq: 10, last_active: NOW - 1_000 },
    ...overrides,
  }
}

function stream(overrides: Partial<StreamInfo['config']> = {}, messages = 10, bytes = 1_000): StreamInfo {
  return {
    name: 'ORDERS',
    subjects: ['orders.>'],
    messages,
    bytes,
    consumer_count: 1,
    created: NOW,
    config: { retention: 'limits', max_msgs: -1, max_bytes: -1, max_age: 0, discard: 'old', ...overrides },
  }
}

const kinds = (c: ConsumerInfo, s?: StreamInfo) => consumerIssues(c, s, NOW).map((i) => i.kind)

describe('consumerKind', () => {
  it('is push when the consumer delivers to a subject, pull otherwise', () => {
    expect(consumerKind(consumer({ config: { deliver_subject: 'deliver.orders' } }))).toBe('push')
    expect(consumerKind(consumer())).toBe('pull')
  })
})

describe('consumerIssues', () => {
  it('finds nothing wrong with a caught-up consumer', () => {
    expect(consumerIssues(consumer(), stream(), NOW)).toEqual([])
  })

  it('flags a consumer that hit its ack limit as stuck', () => {
    const issues = consumerIssues(consumer({ num_pending: 50, num_ack_pending: 1000, num_waiting: 1 }), stream(), NOW)
    expect(issues.map((i) => i.kind)).toEqual(['ack_limit'])
    expect(issues[0].severity).toBe('error')
    expect(issues[0].detail).toContain('1000')
  })

  it('ignores the ack limit when the consumer has none', () => {
    expect(kinds(consumer({ num_ack_pending: 5, config: { ack_policy: 'explicit' } }))).toEqual([])
  })

  it('flags a push consumer with pending messages and no subscriber', () => {
    const lonely = consumer({ num_pending: 12, config: { deliver_subject: 'deliver.orders' } })
    const issues = consumerIssues(lonely, stream(), NOW)
    expect(issues.map((i) => i.kind)).toEqual(['no_subscriber'])
    expect(issues[0].severity).toBe('error')
    expect(issues[0].detail).toContain('deliver.orders')

    expect(kinds({ ...lonely, push_bound: true })).toEqual([])
    expect(kinds({ ...lonely, num_pending: 0 })).toEqual([])
  })

  it('flags a pull consumer nobody pulled from for a while', () => {
    const idle = consumer({ num_pending: 5, delivered: { consumer_seq: 3, stream_seq: 3, last_active: NOW - 5 * 60_000 } })
    const issues = consumerIssues(idle, stream(), NOW)
    expect(issues.map((i) => i.kind)).toEqual(['nobody_pulling'])
    expect(issues[0].severity).toBe('warning')
    expect(issues[0].detail).toContain('5m')
  })

  it('flags a pull consumer that never delivered anything', () => {
    const fresh = consumer({ num_pending: 5, delivered: { consumer_seq: 0, stream_seq: 0 } })
    expect(consumerIssues(fresh, stream(), NOW)[0].detail).toContain('no client has pulled yet')
  })

  it('trusts a waiting pull request or a recent delivery', () => {
    expect(kinds(consumer({ num_pending: 5, num_waiting: 1, delivered: { consumer_seq: 0, stream_seq: 0 } }))).toEqual([])
    expect(kinds(consumer({ num_pending: 5, delivered: { consumer_seq: 3, stream_seq: 3, last_active: NOW - 10_000 } }))).toEqual([])
  })

  it('flags redeliveries with the ack wait', () => {
    const issues = consumerIssues(consumer({ num_ack_pending: 3, num_redelivered: 2 }), stream(), NOW)
    expect(issues.map((i) => i.kind)).toEqual(['redelivering'])
    expect(issues[0].detail).toContain('30s')
  })

  it('reports a paused consumer instead of blaming its clients', () => {
    const paused = consumer({ paused: true, pause_until: '2026-10-05T13:00:00Z', num_pending: 9, delivered: { consumer_seq: 0, stream_seq: 0 } })
    expect(kinds(paused)).toEqual(['paused'])
    expect(kinds({ ...paused, config: { deliver_subject: 'deliver.orders' } })).toEqual(['paused'])
  })

  it('warns when a nearly full stream that drops old messages holds undelivered ones', () => {
    const behind = consumer({ num_pending: 40, num_waiting: 1 })
    const issues = consumerIssues(behind, stream({ max_msgs: 100 }, 95), NOW)
    expect(issues.map((i) => i.kind)).toEqual(['stream_full'])
    expect(issues[0].detail).toContain('95%')

    expect(kinds(behind, stream({ max_bytes: 10_000 }, 10, 9_500))).toEqual(['stream_full'])
    expect(kinds(behind, stream({ max_msgs: 100, discard: 'new' }, 95))).toEqual([])
    expect(kinds(behind, stream({ max_msgs: 100 }, 50))).toEqual([])
    expect(kinds({ ...behind, num_pending: 0 }, stream({ max_msgs: 100 }, 95))).toEqual([])
  })

  it('lists the stuck issues first', () => {
    const many = consumer({ num_pending: 5, num_ack_pending: 1000, num_redelivered: 4, config: { deliver_subject: 'deliver.orders', max_ack_pending: 1000 } })
    expect(kinds(many)).toEqual(['ack_limit', 'no_subscriber', 'redelivering'])
  })

  it('does not blame idle pull clients while the ack limit blocks delivery', () => {
    const blocked = consumer({ num_pending: 5, num_ack_pending: 1000, delivered: { consumer_seq: 0, stream_seq: 0 } })
    expect(kinds(blocked)).toEqual(['ack_limit'])
  })
})

describe('consumerState', () => {
  it('names the worst issue, then whether work is left', () => {
    expect(consumerState(consumer({ num_pending: 50, num_ack_pending: 1000 }), [{ kind: 'ack_limit', severity: 'error', label: '', detail: '' }])).toBe('stuck')
    expect(consumerState(consumer(), [{ kind: 'redelivering', severity: 'warning', label: '', detail: '' }])).toBe('warning')
    expect(consumerState(consumer({ num_pending: 3 }), [])).toBe('working')
    expect(consumerState(consumer({ num_ack_pending: 1 }), [])).toBe('working')
    expect(consumerState(consumer(), [])).toBe('caught_up')
  })
})

describe('compareByHealth', () => {
  it('puts stuck consumers first, then the most pending, then by name', () => {
    const row = (name: string, state: ReturnType<typeof consumerState>, pending: number) => ({
      consumer: consumer({ name, num_pending: pending }),
      state,
    })
    const rows = [row('b', 'caught_up', 0), row('a', 'working', 5), row('c', 'stuck', 1), row('d', 'working', 9), row('e', 'warning', 0)]
    expect(rows.sort(compareByHealth).map((r) => r.consumer.name)).toEqual(['c', 'e', 'd', 'a', 'b'])
  })
})
