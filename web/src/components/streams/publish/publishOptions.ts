import type { CapabilityKey } from '@/contexts/connection'

export type ScheduleKind = 'at' | 'every' | 'cron'

export interface PublishOptions {
  ttl: string
  increment: string
  schedule: {
    enabled: boolean
    kind: ScheduleKind
    at: string
    every: string
    cron: string
    timeZone: string
    target: string
    ttl: string
  }
}

export const EMPTY_PUBLISH_OPTIONS: PublishOptions = {
  ttl: '',
  increment: '',
  schedule: { enabled: false, kind: 'at', at: '', every: '', cron: '', timeZone: '', target: '', ttl: '' },
}

export const INCREMENT_HEADER = 'Nats-Incr'

const DURATION = /^(?:\d+|(?:\d+(?:\.\d+)?(?:ns|us|µs|ms|s|m|h))+)$/
const INCREMENT = /^[+-]?\d+$/

export function isIncrement(options: PublishOptions): boolean {
  return options.increment.trim() !== ''
}

function toRfc3339(local: string): string | undefined {
  const date = new Date(local)
  if (Number.isNaN(date.getTime())) return undefined
  return date.toISOString().replace(/\.\d{3}Z$/, 'Z')
}

function scheduleHeaders(
  schedule: PublishOptions['schedule'],
  subject: string,
): { headers: Record<string, string>; error?: string } {
  const headers: Record<string, string> = {}
  if (schedule.kind === 'at') {
    if (!schedule.at) return { headers, error: 'Pick when the scheduled message is published' }
    const at = toRfc3339(schedule.at)
    if (!at) return { headers, error: 'Pick when the scheduled message is published' }
    if (new Date(at).getTime() <= Date.now()) return { headers, error: 'The schedule time must be in the future' }
    headers['Nats-Schedule'] = `@at ${at}`
  } else if (schedule.kind === 'every') {
    const every = schedule.every.trim()
    if (!DURATION.test(every)) return { headers, error: 'Interval must be a duration like 30s, 5m or 1h' }
    headers['Nats-Schedule'] = `@every ${every}`
  } else {
    const cron = schedule.cron.trim()
    if (!cron) return { headers, error: 'Enter a cron expression' }
    headers['Nats-Schedule'] = cron
    if (schedule.timeZone.trim()) headers['Nats-Schedule-Time-Zone'] = schedule.timeZone.trim()
  }

  const target = schedule.target.trim()
  if (!target) return { headers, error: 'Enter the target subject the schedule publishes to' }
  if (target === subject) {
    return { headers, error: 'The target subject must differ from the subject holding the schedule' }
  }
  headers['Nats-Schedule-Target'] = target

  const ttl = schedule.ttl.trim()
  if (ttl) {
    if (!DURATION.test(ttl)) return { headers, error: 'Schedule TTL must be a duration like 30s, 5m or 1h' }
    headers['Nats-Schedule-TTL'] = ttl
  }
  return { headers }
}

export function buildOptionHeaders(
  options: PublishOptions,
  subject: string,
  userHeaders: Record<string, string>,
): { headers: Record<string, string>; error?: string } {
  const headers: Record<string, string> = {}

  const ttl = options.ttl.trim()
  if (ttl) {
    if (!DURATION.test(ttl)) return { headers, error: 'TTL must be a duration like 30s, 5m or 1h' }
    headers['Nats-TTL'] = ttl
  }

  const increment = options.increment.trim()
  if (increment) {
    if (!INCREMENT.test(increment)) return { headers, error: 'Increment must be a whole number, e.g. 1 or -5' }
    headers[INCREMENT_HEADER] = /^[+-]/.test(increment) ? increment : `+${increment}`
  }

  if (options.schedule.enabled) {
    const schedule = scheduleHeaders(options.schedule, subject)
    if (schedule.error) return { headers, error: schedule.error }
    Object.assign(headers, schedule.headers)
  }

  const typed = new Set(Object.keys(userHeaders).map((key) => key.toLowerCase()))
  const clash = Object.keys(headers).find((key) => typed.has(key.toLowerCase()))
  if (clash) return { headers, error: `${clash} is set both here and in Headers — remove one` }

  return { headers }
}

export interface StreamPublishFeatures {
  msgTtl: boolean
  msgSchedules: boolean
  msgCounter: boolean
}

export interface OptionAvailability {
  ttlUnavailable?: string
  incrementUnavailable?: string
  scheduleUnavailable?: string
  cronUnavailable?: string
}

export function optionAvailability(
  unsupportedReason: (feature: CapabilityKey) => string | undefined,
  stream: StreamPublishFeatures | undefined,
): OptionAvailability {
  const out: OptionAvailability = {}
  const streamTtlReason = stream?.msgCounter
    ? 'Counter streams take no per-message TTL'
    : 'Enable "Allow Per-Message TTL" on this stream first'
  const streamScheduleReason = stream?.msgCounter
    ? 'Counter streams take no schedules'
    : 'Enable "Allow Message Schedules" on this stream first'
  const ttl = unsupportedReason('messageTtl') ?? (stream && !stream.msgTtl ? streamTtlReason : undefined)
  const increment =
    unsupportedReason('msgCounters') ?? (stream && !stream.msgCounter ? 'Only counter streams accept increments' : undefined)
  const schedule =
    unsupportedReason('msgSchedules') ?? (stream && !stream.msgSchedules ? streamScheduleReason : undefined)
  const cron = unsupportedReason('cronSchedules')
  if (ttl) out.ttlUnavailable = ttl
  if (increment) out.incrementUnavailable = increment
  if (schedule) out.scheduleUnavailable = schedule
  if (cron) out.cronUnavailable = cron
  return out
}
