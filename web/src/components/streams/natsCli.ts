import type { StreamConfig, ConsumerConfig } from '@/types/nats'

/**
 * Generate official `nats` CLI commands from a stream/consumer config (target
 * CLI v0.1.x–0.2.x).
 * Config durations are NANOSECONDS, rendered as Go durations. Exotic features
 * (mirror/sources/republish/
 * metadata/push consumers/backoff/flow_control/etc.) are NOT reproduced — copy
 * the JSON config for those.
 */

/**
 * Single-quote so the shell treats the value literally — prevents `$`/backtick
 * expansion of `$KV.`/`$O.` subjects and attacker-controllable descriptions.
 */
const sq = (s: string): string => `'${s.replace(/'/g, "'\\''")}'`

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

  let cmd = `nats stream add ${name} ${flags.join(' ')}`

  const omitted: string[] = []
  if (c.mirror) omitted.push('mirror')
  if (c.sources && c.sources.length > 0) omitted.push('sources')
  if (c.republish) omitted.push('republish')
  if (c.subject_transform) omitted.push('subject_transform')
  if (c.consumer_limits) omitted.push('consumer_limits')
  if (c.metadata && Object.keys(c.metadata).length > 0) omitted.push('metadata')
  if (omitted.length > 0) {
    cmd += `\n# NOTE: omitted (set via the JSON config): ${omitted.join(', ')}`
  }
  return cmd
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

/**
 * Build a `nats consumer add` command; consumerName is the durable name (passed
 * explicitly so callers can label ephemerals).
 */
export function consumerConfigToNatsCli(consumerName: string, streamName: string, c: ConsumerConfig): string {
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
  // Always pull: without --pull natscli prompts interactively, breaking
  // reproducibility.
  flags.push('--pull')

  let cmd = `nats consumer add ${streamName} ${consumerName} ${flags.join(' ')}`

  const omitted: string[] = []
  if (c.headers_only) omitted.push('headers_only')
  if (c.inactive_threshold != null && c.inactive_threshold > 0) omitted.push('inactive_threshold')
  if (c.mem_storage) omitted.push('mem_storage')
  if (omitted.length > 0) {
    cmd += `\n# NOTE: omitted (set via the JSON config): ${omitted.join(', ')}`
  }

  // by_start_time isn't expressible as --deliver (see cliDeliver) — falls back
  // to "all", so NOTE the user.
  if (c.deliver_policy === 'by_start_time') {
    cmd += `\n# NOTE: deliver_policy by_start_time is not reproducible as a flag; ` +
      `set the start time manually${c.opt_start_time ? ` (was ${c.opt_start_time})` : ''}.`
  }

  return cmd
}
