import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { EMPTY_PUBLISH_OPTIONS, buildOptionHeaders, optionAvailability, type PublishOptions } from './publishOptions'

function opts(over: Partial<PublishOptions> = {}, schedule: Partial<PublishOptions['schedule']> = {}): PublishOptions {
  return { ...EMPTY_PUBLISH_OPTIONS, ...over, schedule: { ...EMPTY_PUBLISH_OPTIONS.schedule, ...schedule } }
}

describe('buildOptionHeaders', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2030-01-01T00:00:00Z'))
  })
  afterEach(() => {
    vi.useRealTimers()
  })

  it('adds nothing when no option is set', () => {
    expect(buildOptionHeaders(EMPTY_PUBLISH_OPTIONS, 'orders.new', {})).toEqual({ headers: {} })
  })

  it.each(['30s', '5m', '1h30m', '90', '1.5h'])('accepts the ttl %s', (ttl) => {
    expect(buildOptionHeaders(opts({ ttl }), 'orders.new', {})).toEqual({ headers: { 'Nats-TTL': ttl } })
  })

  it.each(['soon', '5 m', '-1s', 's'])('rejects the ttl %s', (ttl) => {
    expect(buildOptionHeaders(opts({ ttl }), 'orders.new', {}).error).toBe('TTL must be a duration like 30s, 5m or 1h')
  })

  it.each([
    ['1', '+1'],
    ['+5', '+5'],
    ['-3', '-3'],
    ['0', '+0'],
  ])('signs the increment %s as %s', (increment, header) => {
    expect(buildOptionHeaders(opts({ increment }), 'hits.page', {})).toEqual({ headers: { 'Nats-Incr': header } })
  })

  it.each(['one', '1.5', '++1', ''.padEnd(1, ' ')])('rejects the increment %j', (increment) => {
    const result = buildOptionHeaders(opts({ increment }), 'hits.page', {})
    if (increment.trim() === '') {
      expect(result).toEqual({ headers: {} })
    } else {
      expect(result.error).toBe('Increment must be a whole number, e.g. 1 or -5')
    }
  })

  it('schedules a delayed message at a local time in UTC', () => {
    const result = buildOptionHeaders(
      opts({}, { enabled: true, kind: 'at', at: '2030-01-02T10:30', target: 'orders.run' }),
      'orders.schedule.a',
      {},
    )
    expect(result).toEqual({
      headers: {
        'Nats-Schedule': `@at ${new Date('2030-01-02T10:30').toISOString().replace(/\.\d{3}Z$/, 'Z')}`,
        'Nats-Schedule-Target': 'orders.run',
      },
    })
  })

  it('schedules an interval with a ttl for the produced message', () => {
    const result = buildOptionHeaders(
      opts({}, { enabled: true, kind: 'every', every: '5m', target: 'sensors.sampled', ttl: '1h' }),
      'sensors.schedule',
      {},
    )
    expect(result).toEqual({
      headers: {
        'Nats-Schedule': '@every 5m',
        'Nats-Schedule-Target': 'sensors.sampled',
        'Nats-Schedule-TTL': '1h',
      },
    })
  })

  it('schedules a cron with a time zone', () => {
    const result = buildOptionHeaders(
      opts({}, { enabled: true, kind: 'cron', cron: '0 0 9 * * *', timeZone: 'Europe/Amsterdam', target: 'report.daily' }),
      'report.schedule',
      {},
    )
    expect(result.headers).toEqual({
      'Nats-Schedule': '0 0 9 * * *',
      'Nats-Schedule-Target': 'report.daily',
      'Nats-Schedule-Time-Zone': 'Europe/Amsterdam',
    })
  })

  it('ignores the time zone outside cron schedules', () => {
    const result = buildOptionHeaders(
      opts({}, { enabled: true, kind: 'every', every: '1m', timeZone: 'Europe/Amsterdam', target: 't.x' }),
      's.x',
      {},
    )
    expect(result.headers).not.toHaveProperty('Nats-Schedule-Time-Zone')
  })

  it.each([
    [{ kind: 'at' as const, at: '', target: 't.x' }, 'Pick when the scheduled message is published'],
    [{ kind: 'at' as const, at: '2020-01-01T00:00', target: 't.x' }, 'The schedule time must be in the future'],
    [{ kind: 'every' as const, every: 'often', target: 't.x' }, 'Interval must be a duration like 30s, 5m or 1h'],
    [{ kind: 'cron' as const, cron: '  ', target: 't.x' }, 'Enter a cron expression'],
    [{ kind: 'every' as const, every: '1m', target: '' }, 'Enter the target subject the schedule publishes to'],
    [{ kind: 'every' as const, every: '1m', target: 's.x' }, 'The target subject must differ from the subject holding the schedule'],
    [{ kind: 'every' as const, every: '1m', target: 't.x', ttl: 'later' }, 'Schedule TTL must be a duration like 30s, 5m or 1h'],
  ])('rejects the schedule %j', (schedule, error) => {
    expect(buildOptionHeaders(opts({}, { enabled: true, ...schedule }), 's.x', {}).error).toBe(error)
  })

  it('ignores a disabled schedule', () => {
    expect(buildOptionHeaders(opts({}, { enabled: false, kind: 'every', every: 'bad' }), 's.x', {})).toEqual({ headers: {} })
  })

  it('refuses a header that is also typed by hand', () => {
    expect(buildOptionHeaders(opts({ ttl: '5m' }), 'orders.new', { 'Nats-TTL': '1h' }).error).toBe(
      'Nats-TTL is set both here and in Headers — remove one',
    )
  })

  it('treats a differently cased hand-typed header as the same header', () => {
    expect(buildOptionHeaders(opts({ increment: '1' }), 'hits.page', { 'nats-incr': '+2' }).error).toBe(
      'Nats-Incr is set both here and in Headers — remove one',
    )
  })
})

describe('optionAvailability', () => {
  const noServerLimits = () => undefined
  const onlyTo212 = (key: string) =>
    ['consumerReset', 'cronSchedules', 'batchPublish'].includes(key) ? 'Requires NATS 2.14+ (connected server is v2.12.3)' : undefined
  const allStreamFeatures = { msgTtl: true, msgSchedules: true, msgCounter: true }

  it('allows everything on a capable server and stream', () => {
    expect(optionAvailability(noServerLimits, allStreamFeatures)).toEqual({})
  })

  it('fails open while the stream config is unknown', () => {
    expect(optionAvailability(noServerLimits, undefined)).toEqual({})
  })

  it('names the stream setting that is off', () => {
    expect(optionAvailability(noServerLimits, { msgTtl: false, msgSchedules: false, msgCounter: false })).toEqual({
      ttlUnavailable: 'Enable "Allow Per-Message TTL" on this stream first',
      scheduleUnavailable: 'Enable "Allow Message Schedules" on this stream first',
      incrementUnavailable: 'Only counter streams accept increments',
    })
  })

  it('prefers the server version reason over the stream setting', () => {
    const tooOld = (key: string) => (key === 'messageTtl' ? 'Requires NATS 2.11+ (connected server is v2.10.24)' : undefined)
    expect(optionAvailability(tooOld, { ...allStreamFeatures, msgTtl: false }).ttlUnavailable).toBe(
      'Requires NATS 2.11+ (connected server is v2.10.24)',
    )
  })

  it('explains that counter streams take no ttl or schedules', () => {
    expect(optionAvailability(noServerLimits, { msgTtl: false, msgSchedules: false, msgCounter: true })).toEqual({
      ttlUnavailable: 'Counter streams take no per-message TTL',
      scheduleUnavailable: 'Counter streams take no schedules',
    })
  })

  it('limits repeating schedules on 2.12 servers', () => {
    expect(optionAvailability(onlyTo212, allStreamFeatures)).toEqual({
      cronUnavailable: 'Requires NATS 2.14+ (connected server is v2.12.3)',
    })
  })
})
