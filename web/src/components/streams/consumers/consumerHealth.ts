import type { ConsumerInfo, StreamInfo } from '@/types/nats'
import { formatDateTime, formatDuration, formatNsDuration } from '@/utils/formatters'
import { plural } from '@/utils/plural'

const IDLE_PULL_MS = 60_000
const MIN_ACK_SILENCE_MS = 30_000
const STREAM_FULL_RATIO = 0.9
const OLDEST_SHARE = 0.1

export type ConsumerKind = 'pull' | 'push'

type IssueKind =
  | 'paused'
  | 'ack_limit'
  | 'at_ack_limit'
  | 'no_subscriber'
  | 'nobody_pulling'
  | 'redelivering'
  | 'stream_full'

export interface ConsumerIssue {
  kind: IssueKind
  severity: 'error' | 'warning'
  label: string
  detail: string
}

export type ConsumerState = 'stuck' | 'warning' | 'working' | 'caught_up'

export function consumerKind(consumer: ConsumerInfo): ConsumerKind {
  return consumer.config?.deliver_subject ? 'push' : 'pull'
}

function waiting(count: number): string {
  return `${plural(count, 'message is', 'messages are')} waiting`
}

function ago(ms: number): string {
  return formatDuration(Math.max(0, Math.floor(ms / 1000)))
}

function pausedIssue(consumer: ConsumerInfo): ConsumerIssue {
  const until = consumer.pause_until ? new Date(consumer.pause_until) : null
  return {
    kind: 'paused',
    severity: 'warning',
    label: 'Paused',
    detail:
      until && !Number.isNaN(until.getTime())
        ? `Delivery is paused until ${formatDateTime(until)}.`
        : 'Delivery is paused.',
  }
}

function ackLimitIssue(consumer: ConsumerInfo, now: number): ConsumerIssue {
  const max = consumer.config?.max_ack_pending ?? 0
  const lastAck = consumer.ack_floor?.last_active
  const silence = Math.max(MIN_ACK_SILENCE_MS, (consumer.config?.ack_wait ?? 0) / 1_000_000)
  const atLimit = `${waiting(consumer.num_ack_pending)} for an ack, the most this consumer allows (max ack pending ${max}).`
  if (lastAck != null && now - lastAck <= silence) {
    return {
      kind: 'at_ack_limit',
      severity: 'warning',
      label: 'At its ack limit',
      detail: `${atLimit} Clients are still acking, so it is busy rather than stuck; new messages go out as acks free a slot.`,
    }
  }
  return {
    kind: 'ack_limit',
    severity: 'error',
    label: 'Ack limit reached',
    detail:
      `${atLimit} ${lastAck != null ? `No ack came in for ${ago(now - lastAck)}. ` : ''}` +
      'Nothing new is delivered until clients ack or the ack wait runs out.',
  }
}

function streamFillOf(stream: StreamInfo): number {
  const { max_msgs: maxMsgs, max_bytes: maxBytes } = stream.config
  const byMsgs = maxMsgs > 0 ? stream.messages / maxMsgs : 0
  const byBytes = maxBytes > 0 ? stream.bytes / maxBytes : 0
  return Math.max(byMsgs, byBytes)
}

function streamFullIssue(consumer: ConsumerInfo, stream: StreamInfo | undefined): ConsumerIssue | null {
  if (!stream?.state || consumer.num_pending === 0 || (stream.config.discard ?? 'old') !== 'old') return null
  const fill = streamFillOf(stream)
  const next = (consumer.delivered?.stream_seq ?? 0) + 1
  if (fill < STREAM_FULL_RATIO || next - stream.state.first_seq >= stream.messages * OLDEST_SHARE) return null
  const percent = Math.min(100, Math.floor(fill * 100))
  if (next < stream.state.first_seq && (consumer.delivered?.consumer_seq ?? 0) > 0) {
    return {
      kind: 'stream_full',
      severity: 'warning',
      label: 'Losing messages',
      detail:
        `Stream ${stream.name} is ${percent}% full and already dropped messages #${next}–#${stream.state.first_seq - 1} ` +
        'before this consumer reached them. More of its messages may go the same way.',
    }
  }
  return {
    kind: 'stream_full',
    severity: 'warning',
    label: 'May lose messages',
    detail:
      `Stream ${stream.name} is ${percent}% full and drops its oldest messages when full. ` +
      'The next message for this consumer is among the oldest, so it may be gone before the consumer gets it.',
  }
}

export function consumerIssues(consumer: ConsumerInfo, stream: StreamInfo | undefined, now: number): ConsumerIssue[] {
  const issues: ConsumerIssue[] = []
  const at = consumer.time_stamp ?? now
  const pending = consumer.num_pending
  const paused = consumer.paused === true
  const max = consumer.config?.max_ack_pending ?? 0
  const blocked = max > 0 && consumer.num_ack_pending >= max

  if (paused) issues.push(pausedIssue(consumer))
  if (blocked) issues.push(ackLimitIssue(consumer, at))

  if (!paused && pending > 0 && consumerKind(consumer) === 'push' && !consumer.push_bound) {
    issues.push({
      kind: 'no_subscriber',
      severity: 'error',
      label: 'No subscriber',
      detail: `${waiting(pending)}, and nobody is subscribed to ${consumer.config?.deliver_subject}, where this push consumer delivers.`,
    })
  }

  if (!paused && !blocked && pending > 0 && consumerKind(consumer) === 'pull' && !(consumer.num_waiting ?? 0)) {
    const neverDelivered = (consumer.delivered?.consumer_seq ?? 0) === 0
    const last = consumer.delivered?.last_active
    if (neverDelivered) {
      issues.push({
        kind: 'nobody_pulling',
        severity: 'warning',
        label: 'Nobody pulling',
        detail: `${waiting(pending)}, and no client has pulled yet.`,
      })
    } else if (last != null && at - last > IDLE_PULL_MS) {
      issues.push({
        kind: 'nobody_pulling',
        severity: 'warning',
        label: 'Nobody pulling',
        detail: `${waiting(pending)}, and no client has pulled for ${ago(at - last)}.`,
      })
    }
  }

  const redelivered = consumer.num_redelivered ?? 0
  if (redelivered > 0) {
    const ackWait = consumer.config?.ack_wait
    issues.push({
      kind: 'redelivering',
      severity: 'warning',
      label: 'Redelivering',
      detail:
        `${plural(redelivered, 'unacknowledged message was', 'unacknowledged messages were')} delivered again: a client ` +
        `rejected ${redelivered === 1 ? 'it' : 'them'} or did not ack within the ack wait${ackWait ? ` (${formatNsDuration(ackWait)})` : ''}.`,
    })
  }

  const full = streamFullIssue(consumer, stream)
  if (full) issues.push(full)

  return issues.sort((a, b) => (a.severity === b.severity ? 0 : a.severity === 'error' ? -1 : 1))
}

export function consumerState(consumer: ConsumerInfo, issues: ConsumerIssue[]): ConsumerState {
  if (issues.some((i) => i.severity === 'error')) return 'stuck'
  if (issues.length > 0) return 'warning'
  if (consumer.num_pending > 0 || consumer.num_ack_pending > 0) return 'working'
  return 'caught_up'
}

const STATE_RANK: Record<ConsumerState, number> = { stuck: 0, warning: 1, working: 2, caught_up: 3 }

export function compareByHealth(
  a: { consumer: ConsumerInfo; state: ConsumerState },
  b: { consumer: ConsumerInfo; state: ConsumerState },
): number {
  return (
    STATE_RANK[a.state] - STATE_RANK[b.state] ||
    b.consumer.num_pending - a.consumer.num_pending ||
    a.consumer.name.localeCompare(b.consumer.name)
  )
}

export function statusText(issues: ConsumerIssue[], state: ConsumerState): string {
  if (issues.length > 0) return issues.map((i) => i.label).join(', ')
  return state === 'working' ? 'Catching up' : 'Caught up'
}
