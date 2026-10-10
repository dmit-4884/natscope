import type { StreamConfig, ConsumerConfig } from '@/types/nats'
import { shellQuote as sq } from '@/utils/shell'

/**
 * Generate official `nats` CLI commands from a stream/consumer config (flags of
 * natscli v0.4). Config durations are NANOSECONDS, rendered as Go durations.
 * Every non-default config field that has no flag is listed in a `# NOTE` line,
 * found by scanning the whole config object, so a new field is never dropped
 * silently.
 */

/**
 * Render a nanosecond duration as a compact Go duration string (non-positive →
 * "0s").
 */
export function nanosToGoDuration(ns: number): string {
  if (!Number.isFinite(ns) || ns <= 0) return '0s'
  let rem = Math.round(ns)

  if (rem < 1_000_000_000) {
    if (rem % 1_000_000 === 0) return `${rem / 1_000_000}ms`
    if (rem % 1_000 === 0) return `${rem / 1_000}us`
    return `${rem}ns`
  }

  const h = Math.floor(rem / 3_600_000_000_000)
  rem -= h * 3_600_000_000_000
  const m = Math.floor(rem / 60_000_000_000)
  rem -= m * 60_000_000_000
  const s = rem / 1_000_000_000 // may be fractional

  let out = ''
  if (h) out += `${h}h`
  if (m) out += `${m}m`
  if (s) out += `${s}s`
  return out || '0s'
}

// nats CLI uses "work" for the WorkQueue retention policy.
function cliRetention(retention: string): string {
  return retention === 'workqueue' ? 'work' : retention
}

function isDefaultValue(value: unknown): boolean {
  if (value === undefined || value === null || value === false || value === 0 || value === '') return true
  if (Array.isArray(value)) return value.length === 0
  if (typeof value === 'object') return Object.values(value).every(isDefaultValue)
  return false
}

function unhandledKeys(config: object, handled: ReadonlySet<string>): string[] {
  return Object.entries(config)
    .filter(([key, value]) => !handled.has(key) && !isDefaultValue(value))
    .map(([key]) => key)
}

function omissionNote(keys: string[]): string {
  return keys.length > 0 ? `\n# NOTE: omitted (no CLI flag; set via the JSON config): ${keys.join(', ')}` : ''
}

function onlyName(ref: { name: string }): boolean {
  return Object.entries(ref).every(([key, value]) => key === 'name' || isDefaultValue(value))
}

const STREAM_HANDLED: ReadonlySet<string> = new Set([
  'retention',
  'storage',
  'num_replicas',
  'max_msgs',
  'max_msgs_per_subject',
  'max_bytes',
  'max_msg_size',
  'max_age',
  'max_consumers',
  'discard',
  'discard_new_per_subject',
  'duplicate_window',
  'compression',
  'deny_delete',
  'deny_purge',
  'allow_rollup_hdrs',
  'allow_direct',
  'mirror_direct',
  'allow_msg_ttl',
  'allow_msg_counter',
  'allow_msg_schedules',
  'allow_atomic',
  'allow_batched',
  'subject_delete_marker_ttl',
  'persist_mode',
  'metadata',
  'mirror',
  'sources',
  'republish',
  'subject_transform',
  'consumer_limits',
  'name',
  'subjects',
])

export interface StreamCliInput {
  name: string
  subjects: string[]
  config: StreamConfig
}

/**
 * Build a `nats stream add` command from a config; appends a `# NOTE` line
 * listing exotic features to set by hand.
 */
export function streamConfigToNatsCli({ name, subjects, config: c }: StreamCliInput): string {
  const flags: string[] = []

  if (subjects.length > 0) flags.push(`--subjects=${sq(subjects.join(','))}`)
  if (c.retention) flags.push(`--retention=${cliRetention(c.retention)}`)
  if (c.storage) flags.push(`--storage=${c.storage}`)
  flags.push(`--replicas=${c.num_replicas ?? 1}`)
  flags.push(`--max-msgs=${c.max_msgs ?? -1}`)
  if (c.max_msgs_per_subject != null && c.max_msgs_per_subject !== -1) {
    flags.push(`--max-msgs-per-subject=${c.max_msgs_per_subject}`)
  }
  flags.push(`--max-bytes=${c.max_bytes ?? -1}`)
  flags.push(`--max-msg-size=${c.max_msg_size ?? -1}`)
  flags.push(`--max-age=${c.max_age && c.max_age > 0 ? nanosToGoDuration(c.max_age) : '-1'}`)
  if (c.max_consumers != null && c.max_consumers !== -1) flags.push(`--max-consumers=${c.max_consumers}`)
  if (c.discard) flags.push(`--discard=${c.discard}`)
  // Emit --dupe-window only when positive; 0s would override the server default
  // (2m).
  if (c.duplicate_window != null && c.duplicate_window > 0) {
    flags.push(`--dupe-window=${nanosToGoDuration(c.duplicate_window)}`)
  }
  if (c.compression && c.compression !== 'none') flags.push(`--compression=${c.compression}`)
  if (c.deny_delete) flags.push('--deny-delete')
  if (c.deny_purge) flags.push('--deny-purge')
  if (c.allow_rollup_hdrs) flags.push('--allow-rollup')
  // natscli defaults allow_direct=true, so the negative case needs an explicit
  // flag.
  if (c.allow_direct) flags.push('--allow-direct')
  else if (c.allow_direct === false) flags.push('--no-allow-direct')

  if (c.discard_new_per_subject) flags.push('--discard-per-subject')
  if (c.mirror_direct) flags.push('--allow-mirror-direct')
  if (c.allow_msg_ttl) flags.push('--allow-msg-ttl')
  if (c.allow_msg_counter) flags.push('--allow-counter')
  if (c.allow_msg_schedules) flags.push('--allow-schedules')
  if (c.allow_atomic) flags.push('--allow-batch')
  if (c.allow_batched) flags.push('--allow-fast')
  if (c.subject_delete_marker_ttl && c.subject_delete_marker_ttl > 0) {
    flags.push(`--subject-del-markers-ttl=${nanosToGoDuration(c.subject_delete_marker_ttl)}`)
  }
  if (c.persist_mode === 'async') flags.push('--persist-mode=async')
  if (c.republish) {
    flags.push(`--republish-source=${sq(c.republish.src)}`, `--republish-destination=${sq(c.republish.dest)}`)
    if (c.republish.headers_only) flags.push('--republish-headers')
  }
  if (c.subject_transform) {
    flags.push(`--transform-source=${sq(c.subject_transform.src)}`, `--transform-destination=${sq(c.subject_transform.dest)}`)
  }
  if (c.consumer_limits?.inactive_threshold) {
    flags.push(`--limit-consumer-inactive=${nanosToGoDuration(c.consumer_limits.inactive_threshold)}`)
  }
  if (c.consumer_limits?.max_ack_pending) flags.push(`--limit-consumer-max-pending=${c.consumer_limits.max_ack_pending}`)
  for (const [key, value] of Object.entries(c.metadata ?? {})) flags.push(`--metadata=${sq(`${key}=${value}`)}`)

  const omitted = unhandledKeys(c, STREAM_HANDLED)
  if (c.mirror) {
    if (onlyName(c.mirror)) flags.push(`--mirror=${sq(c.mirror.name)}`)
    else omitted.push('mirror')
  }
  if (c.sources && c.sources.length > 0) {
    if (c.sources.every(onlyName)) for (const source of c.sources) flags.push(`--source=${sq(source.name)}`)
    else omitted.push('sources')
  }

  return `nats stream add ${name} ${flags.join(' ')}${omissionNote(omitted)}`
}

// Map deliver_policy to the CLI's --deliver value; by_start_time isn't
// expressible as a flag, so it falls back to "all" (caller emits a NOTE).
function cliDeliver(c: ConsumerConfig): string {
  switch (c.deliver_policy) {
    case 'new':
      return 'new'
    case 'last':
      return 'last'
    case 'last_per_subject':
      return 'subject'
    case 'by_start_sequence':
      return String(c.opt_start_seq ?? 1)
    case 'by_start_time':
    case 'all':
    default:
      return 'all'
  }
}

const CONSUMER_HANDLED: ReadonlySet<string> = new Set([
  'name',
  'durable_name',
  'description',
  'deliver_subject',
  'deliver_group',
  'deliver_policy',
  'opt_start_seq',
  'opt_start_time',
  'ack_policy',
  'ack_wait',
  'max_deliver',
  'backoff',
  'filter_subject',
  'filter_subjects',
  'replay_policy',
  'rate_limit_bps',
  'sample_freq',
  'max_waiting',
  'max_ack_pending',
  'flow_control',
  'idle_heartbeat',
  'headers_only',
  'max_batch',
  'max_expires',
  'max_bytes',
  'inactive_threshold',
  'num_replicas',
  'mem_storage',
  'metadata',
  'priority_policy',
  'priority_groups',
  'priority_timeout',
])

const PRIORITY_GROUP_FLAG: Record<string, string> = {
  pinned_client: '--pinned-groups',
  overflow: '--overflow-groups',
  prioritized: '--prioritized-groups',
}

/**
 * Build a `nats consumer add` command; consumerName is the durable name (passed
 * explicitly so callers can label ephemerals). A consumer with a deliver subject
 * stays a push consumer (`--target`); every other one is pull.
 */
export function consumerConfigToNatsCli(consumerName: string, streamName: string, c: ConsumerConfig, pauseUntil?: string): string {
  const flags: string[] = []

  const filters = c.filter_subjects && c.filter_subjects.length > 0
    ? c.filter_subjects
    : c.filter_subject
      ? [c.filter_subject]
      : []
  for (const f of filters) flags.push(`--filter=${sq(f)}`)

  flags.push(`--deliver=${cliDeliver(c)}`)
  if (c.ack_policy) flags.push(`--ack=${c.ack_policy}`)
  if (c.ack_wait && c.ack_wait > 0) flags.push(`--wait=${nanosToGoDuration(c.ack_wait)}`)
  flags.push(`--max-deliver=${c.max_deliver ?? -1}`)
  if (c.replay_policy) flags.push(`--replay=${c.replay_policy}`)
  if (c.max_ack_pending != null) flags.push(`--max-pending=${c.max_ack_pending}`)
  if (c.max_waiting != null && c.max_waiting > 0) flags.push(`--max-waiting=${c.max_waiting}`)
  if (c.num_replicas != null && c.num_replicas > 0) flags.push(`--replicas=${c.num_replicas}`)
  // natscli expects an integer percentage, not a "%" suffix.
  if (c.sample_freq) flags.push(`--sample=${String(c.sample_freq).replace('%', '')}`)
  if (c.description) flags.push(`--description=${sq(c.description)}`)
  if (c.rate_limit_bps) flags.push(`--bps=${c.rate_limit_bps}`)
  if (c.headers_only) flags.push('--headers-only')
  if (c.mem_storage) flags.push('--memory')
  if (c.inactive_threshold && c.inactive_threshold > 0) flags.push(`--inactive-threshold=${nanosToGoDuration(c.inactive_threshold)}`)
  for (const [key, value] of Object.entries(c.metadata ?? {})) flags.push(`--metadata=${sq(`${key}=${value}`)}`)

  const omitted = unhandledKeys(c, CONSUMER_HANDLED)

  if (c.deliver_subject) {
    flags.push(`--target=${sq(c.deliver_subject)}`)
    if (c.deliver_group) flags.push(`--deliver-group=${sq(c.deliver_group)}`)
    if (c.flow_control) flags.push('--flow-control')
    if (c.idle_heartbeat && c.idle_heartbeat > 0) flags.push(`--heartbeat=${nanosToGoDuration(c.idle_heartbeat)}`)
    flags.push('--defaults')
  } else {
    if (c.max_batch) flags.push(`--max-pull-batch=${c.max_batch}`)
    if (c.max_expires && c.max_expires > 0) flags.push(`--max-pull-expire=${nanosToGoDuration(c.max_expires)}`)
    if (c.max_bytes) flags.push(`--max-pull-bytes=${c.max_bytes}`)
    const groupFlag = PRIORITY_GROUP_FLAG[c.priority_policy ?? '']
    if (groupFlag && c.priority_groups) for (const group of c.priority_groups) flags.push(`${groupFlag}=${sq(group)}`)
    else if (c.priority_groups && c.priority_groups.length > 0) omitted.push('priority_groups')
    if (c.priority_timeout && c.priority_timeout > 0) flags.push(`--pinned-ttl=${nanosToGoDuration(c.priority_timeout)}`)
    // Without --pull natscli prompts interactively, breaking reproducibility.
    flags.push('--pull')
  }

  if (c.backoff && c.backoff.length > 0) omitted.push('backoff')
  if (pauseUntil) omitted.push(`pause_until (${pauseUntil}; use --pause=<duration or timestamp>)`)

  let cmd = `nats consumer add ${streamName} ${consumerName} ${flags.join(' ')}${omissionNote(omitted)}`

  // by_start_time isn't expressible as --deliver (see cliDeliver) — falls back
  // to "all", so NOTE the user.
  if (c.deliver_policy === 'by_start_time') {
    cmd += `\n# NOTE: deliver_policy by_start_time is not reproducible as a flag; ` +
      `set the start time manually${c.opt_start_time ? ` (was ${c.opt_start_time})` : ''}.`
  }

  return cmd
}
