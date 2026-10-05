import { describe, it, expect } from 'vitest'
import type { ConsumerInfo } from '@/types/nats'
import { messageFates } from './messageFate'

const CREATED = Date.UTC(2026, 9, 5, 10, 0, 0)

function consumer(name: string, overrides: Partial<ConsumerInfo> = {}): ConsumerInfo {
  return {
    name,
    stream_name: 'ORDERS',
    created: CREATED,
    num_pending: 0,
    num_ack_pending: 0,
    config: { ack_policy: 'explicit', deliver_policy: 'all' },
    delivered: { consumer_seq: 8, stream_seq: 20 },
    ack_floor: { consumer_seq: 5, stream_seq: 12 },
    ...overrides,
  }
}

const message = (sequence: number, subject = 'orders.created', timestamp = CREATED + 60_000) => ({ sequence, subject, timestamp })

const stateOf = (c: ConsumerInfo, m = message(15)) => messageFates([c], m).fates[0]?.state

describe('messageFates', () => {
  it('tells acknowledged, delivered and not yet delivered apart by the consumer position', () => {
    const c = consumer('billing')
    expect(stateOf(c, message(12))).toBe('done')
    expect(stateOf(c, message(13))).toBe('awaiting_ack')
    expect(stateOf(c, message(20))).toBe('awaiting_ack')
    expect(stateOf(c, message(21))).toBe('not_delivered')
  })

  it('leaves out consumers whose filter does not take the subject', () => {
    const result = messageFates(
      [
        consumer('billing', { config: { filter_subject: 'orders.created' } }),
        consumer('shipping', { config: { filter_subjects: ['orders.shipped', 'orders.*.eu'] } }),
        consumer('audit', { config: { filter_subject: 'orders.>' } }),
        consumer('everything', { config: {} }),
      ],
      message(15),
    )
    expect(result.fates.map((f) => f.consumer.name)).toEqual(['billing', 'audit', 'everything'])
    expect(result.unrelated).toBe(1)
  })

  it('reads delivery alone for a consumer without acks', () => {
    const fireAndForget = consumer('tail', { config: { ack_policy: 'none' }, ack_floor: { consumer_seq: 8, stream_seq: 20 } })
    expect(stateOf(fireAndForget, message(15))).toBe('delivered')
    expect(stateOf(fireAndForget, message(21))).toBe('not_delivered')
  })

  it('knows a cumulative ack leaves nothing acknowledged out of order', () => {
    const cumulative = consumer('batch', { config: { ack_policy: 'all' } })
    const fate = messageFates([cumulative], message(15)).fates[0]
    expect(fate.state).toBe('awaiting_ack')
    expect(fate.detail).not.toContain('out of order')

    const explicit = messageFates([consumer('billing')], message(15)).fates[0]
    expect(explicit.detail).toContain('out of order')
  })

  it('names messages from before the consumer start as skipped', () => {
    expect(stateOf(consumer('late', { config: { deliver_policy: 'by_start_sequence', opt_start_seq: 13 } }), message(12))).toBe('skipped')
    expect(stateOf(consumer('late', { config: { deliver_policy: 'by_start_sequence', opt_start_seq: 13 } }), message(13, 'orders.created'))).not.toBe('skipped')
    expect(
      stateOf(consumer('late', { config: { deliver_policy: 'by_start_time', opt_start_time: '2026-10-05T10:30:00Z' } }), message(10, 'orders.created', CREATED)),
    ).toBe('skipped')
    expect(stateOf(consumer('fresh', { config: { deliver_policy: 'new' } }), message(10, 'orders.created', CREATED - 1_000))).toBe('skipped')
  })

  it('cannot tell acknowledged from skipped for a consumer that started at the last message', () => {
    const fromLast = consumer('tail', { config: { deliver_policy: 'last' } })
    expect(stateOf(fromLast, message(10, 'orders.created', CREATED - 1_000))).toBe('done_or_skipped')
    expect(stateOf(fromLast, message(15, 'orders.created', CREATED - 1_000))).toBe('awaiting_or_skipped')
    expect(stateOf(fromLast, message(10, 'orders.created', CREATED + 1_000))).toBe('done')
  })

  it('mentions a pause on messages still to deliver', () => {
    const paused = consumer('billing', { paused: true })
    expect(messageFates([paused], message(30)).fates[0].detail).toContain('paused')
  })
})

describe('messageFates done', () => {
  it('admits a message past the ack floor may have been given up on', () => {
    const limited = consumer('billing', { config: { ack_policy: 'explicit', deliver_policy: 'all', max_deliver: 5 } })
    const fate = messageFates([limited], message(10)).fates[0]
    expect(fate.label).toBe('Done')
    expect(fate.detail).toBe('Acknowledged, terminated by a client, or dropped after 5 delivery attempts.')

    const unlimited = messageFates([consumer('audit')], message(10)).fates[0]
    expect(unlimited.detail).toBe('Acknowledged or terminated by a client.')
  })
})
