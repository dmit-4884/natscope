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

function stream(overrides: Partial<StreamInfo['config']> = {}, messages = 10, bytes = 1_000, firstSeq = 1): StreamInfo {
  return {
    name: 'ORDERS',
    subjects: ['orders.>'],
    messages,
    bytes,
    consumer_count: 1,
    created: NOW,
    config: { retention: 'limits', max_msgs: -1, max_bytes: -1, max_age: 0, discard: 'old', ...overrides },
    state: { messages, bytes, first_seq: firstSeq, last_seq: firstSeq + messages - 1, first_ts: NOW, last_ts: NOW },
  }
}

const kinds = (c: ConsumerInfo, s?: StreamInfo) => consumerIssues(c, s, NOW).map((i) => i.kind)
const ackedLongAgo = { consumer_seq: 10, stream_seq: 10, last_active: NOW - 5 * 60_000 }

describe('consumerKind', () => {
  it('is push when the consumer delivers to a subject, pull otherwise', () => {
    expect(consumerKind(consumer({ config: { deliver_subject: 'deliver.orders' } }))).toBe('push')
    expect(consumerKind(consumer())).toBe('pull')
  })
})

describe('consumerIssues', () => {
  it('groups digits in the diagnosis texts', () => {
    const issues = consumerIssues(consumer({ num_pending: 5_415_090, num_waiting: 0, delivered: { consumer_seq: 0, stream_seq: 0 } }), stream(), NOW)
    expect(issues[0].detail).toContain('5,415,090 messages are waiting')
  })

  it('finds nothing wrong with a caught-up consumer', () => {
    expect(consumerIssues(consumer(), stream(), NOW)).toEqual([])
  })

  it('flags a consumer at its ack limit whose acks stopped as stuck', () => {
    const issues = consumerIssues(consumer({ num_pending: 50, num_ack_pending: 1000, num_waiting: 1, ack_floor: ackedLongAgo }), stream(), NOW)
    expect(issues.map((i) => i.kind)).toEqual(['ack_limit'])
    expect(issues[0].severity).toBe('error')
    expect(issues[0].detail).toContain('1,000 messages are waiting for an ack')
    expect(issues[0].detail).toContain('5m')
  })

  it('calls a consumer at its ack limit busy while acks keep coming', () => {
    const issues = consumerIssues(consumer({ num_pending: 50, num_ack_pending: 1000, num_waiting: 1 }), stream(), NOW)
    expect(issues.map((i) => i.kind)).toEqual(['at_ack_limit'])
    expect(issues[0].severity).toBe('warning')
  })

  it('treats an ack limit without any known ack as stuck', () => {
    expect(kinds(consumer({ num_ack_pending: 1000, ack_floor: { consumer_seq: 0, stream_seq: 0 } }))).toEqual(['ack_limit'])
  })

  it('uses the singular for one message', () => {
    const one = consumer({ num_ack_pending: 1, config: { max_ack_pending: 1 }, ack_floor: ackedLongAgo })
    expect(consumerIssues(one, stream(), NOW)[0].detail).toContain('1 message is waiting for an ack')
  })

  it('ignores the ack limit when the consumer has none', () => {
    expect(kinds(consumer({ num_ack_pending: 5, config: { ack_policy: 'explicit' } }))).toEqual([])
    expect(kinds(consumer({ num_ack_pending: 5, config: { ack_policy: 'explicit', max_ack_pending: -1 } }))).toEqual([])
  })

  it('flags a push consumer with pending messages and no subscriber', () => {
    const lonely = consumer({ num_pending: 12, config: { deliver_subject: 'deliver.orders' } })
    const issues = consumerIssues(lonely, stream(), NOW)
    expect(issues.map((i) => i.kind)).toEqual(['no_subscriber'])
    expect(issues[0].severity).toBe('error')
    expect(issues[0].detail).toContain('12 messages are waiting, and nobody is subscribed to deliver.orders')

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

  it('counts an idle start policy as never delivered', () => {
    const startsNew = consumer({ num_pending: 5, delivered: { consumer_seq: 0, stream_seq: 40 } })
    expect(consumerIssues(startsNew, stream(), NOW)[0].detail).toContain('no client has pulled yet')
  })

  it('does not guess when the server no longer knows the last delivery', () => {
    expect(kinds(consumer({ num_pending: 5, delivered: { consumer_seq: 3, stream_seq: 3 } }))).toEqual([])
  })

  it('measures idleness on the server clock when it has one', () => {
    const skewed = consumer({
      num_pending: 5,
      time_stamp: NOW - 10 * 60_000,
      delivered: { consumer_seq: 3, stream_seq: 3, last_active: NOW - 10 * 60_000 - 5_000 },
    })
    expect(kinds(skewed)).toEqual([])
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

  it('warns when the next message for a consumer is among the oldest of a nearly full stream that drops old ones', () => {
    const atTheTail = consumer({ num_pending: 90, num_waiting: 1, delivered: { consumer_seq: 5, stream_seq: 905, last_active: NOW } })
    const full = stream({ max_msgs: 100 }, 95, 1_000, 906)
    const issues = consumerIssues(atTheTail, full, NOW)
    expect(issues.map((i) => i.kind)).toEqual(['stream_full'])
    expect(issues[0].detail).toContain('95%')

    expect(kinds(atTheTail, stream({ max_bytes: 10_000 }, 95, 9_500, 906))).toEqual(['stream_full'])
    expect(kinds(atTheTail, stream({ max_msgs: 100, discard: 'new' }, 95, 1_000, 906))).toEqual([])
    expect(kinds(atTheTail, stream({ max_msgs: 100 }, 50, 1_000, 906))).toEqual([])
    expect(kinds({ ...atTheTail, num_pending: 0 }, full)).toEqual([])
  })

  it('leaves a consumer near the head of a full stream alone', () => {
    const nearHead = consumer({ num_pending: 3, num_waiting: 1, delivered: { consumer_seq: 90, stream_seq: 997, last_active: NOW } })
    expect(kinds(nearHead, stream({ max_msgs: 100 }, 95, 1_000, 906))).toEqual([])
  })

  it('lists the stuck issues first', () => {
    const many = consumer({
      num_pending: 5,
      num_ack_pending: 1000,
      num_redelivered: 4,
      config: { deliver_subject: 'deliver.orders', max_ack_pending: 1000 },
      ack_floor: ackedLongAgo,
    })
    expect(kinds(many)).toEqual(['ack_limit', 'no_subscriber', 'redelivering'])
  })

  it('does not blame idle pull clients while the ack limit blocks delivery', () => {
    const blocked = consumer({ num_pending: 5, num_ack_pending: 1000, delivered: { consumer_seq: 0, stream_seq: 0 }, ack_floor: ackedLongAgo })
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
  it('puts stuck consumers first, then by name, so a refresh does not reshuffle them', () => {
    const row = (name: string, state: ReturnType<typeof consumerState>, pending: number) => ({
      consumer: consumer({ name, num_pending: pending }),
      state,
    })
    const rows = [row('b', 'caught_up', 0), row('a', 'working', 5), row('c', 'stuck', 1), row('d', 'working', 9), row('e', 'warning', 0)]
    expect(rows.sort(compareByHealth).map((r) => r.consumer.name)).toEqual(['c', 'e', 'a', 'd', 'b'])
  })
})

describe('consumerIssues lost messages', () => {
  it('says when the stream already dropped messages the consumer had not reached', () => {
    const behind = consumer({ num_pending: 90, num_waiting: 1, delivered: { consumer_seq: 5, stream_seq: 800, last_active: NOW } })
    const issues = consumerIssues(behind, stream({ max_msgs: 100 }, 95, 1_000, 906), NOW)
    expect(issues.map((i) => i.kind)).toEqual(['stream_full'])
    expect(issues[0].label).toBe('Losing messages')
    expect(issues[0].detail).toContain('already dropped messages #801–#905')
  })
})

describe('consumerIssues lost messages for a new consumer', () => {
  it('does not claim losses for a consumer that has not delivered anything', () => {
    const fresh = consumer({ num_pending: 90, num_waiting: 1, delivered: { consumer_seq: 0, stream_seq: 0 } })
    const issues = consumerIssues(fresh, stream({ max_msgs: 100 }, 95, 1_000, 906), NOW)
    expect(issues[0].label).toBe('May lose messages')
  })
})

describe('consumerIssues lost messages for a filtered consumer', () => {
  const full = stream({ max_msgs: 100 }, 95, 1_000, 906)
  const behind = (filter: string) =>
    consumer({ num_pending: 1, num_waiting: 1, config: { filter_subject: filter }, delivered: { consumer_seq: 5, stream_seq: 5, last_active: NOW } })

  it('warns that messages matching its filter may be lost once the stream dropped what lies before its position', () => {
    const issues = consumerIssues(behind('orders.eu'), full, NOW)
    expect(issues.map((i) => i.kind)).toEqual(['stream_full'])
    expect(issues[0].label).toBe('Losing messages')
    expect(issues[0].detail).toContain('#6–#905')
    expect(issues[0].detail).toContain('matching its filter')
  })

  it('warns a filtered consumer that never delivered, while its filtered backlog may sit among the oldest', () => {
    const fresh = consumer({ num_pending: 1_500_000, num_waiting: 1, config: { filter_subject: 'orders.eu' }, delivered: { consumer_seq: 0, stream_seq: 0 } })
    const issues = consumerIssues(fresh, full, NOW)
    expect(issues.map((i) => i.kind)).toEqual(['stream_full'])
    expect(issues[0].label).toBe('May lose messages')
    expect(issues[0].detail).toContain('1,500,000')
    expect(issues[0].detail).not.toContain('The next message for this consumer is among the oldest')
  })

  it('stays quiet for a filtered consumer well ahead of the head of the stream', () => {
    const ahead = consumer({ num_pending: 1, num_waiting: 1, config: { filter_subject: 'orders.eu' }, delivered: { consumer_seq: 5, stream_seq: 990, last_active: NOW } })
    expect(kinds(ahead, full)).toEqual([])
  })

  it('still warns when its filter takes every subject of the stream', () => {
    expect(kinds(behind('orders.>'), full)).toEqual(['stream_full'])
  })
})

describe('consumerIssues redelivering', () => {
  it('stops once nothing waits for an ack, as a message that ran out of attempts keeps the count', () => {
    expect(kinds(consumer({ num_ack_pending: 0, num_redelivered: 1 }))).toEqual([])
  })
})
