import type { ConsumerInfo, StreamInfo } from '@/types/nats'
import { formatDateTime, formatDuration, formatNsDuration } from '@/utils/formatters'
import { plural } from '@/utils/plural'

const IDLE_PULL_MS = 60_000
const STREAM_FULL_RATIO = 0.9

export type ConsumerKind = 'pull' | 'push'

type IssueKind = 'paused' | 'ack_limit' | 'no_subscriber' | 'nobody_pulling' | 'redelivering' | 'stream_full'

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

function messages(count: number): string {
  return plural(count, 'message')
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

function ackLimitReached(consumer: ConsumerInfo): boolean {
  const max = consumer.config?.max_ack_pending ?? 0
  return max > 0 && consumer.num_ack_pending >= max
}

function idleFor(consumer: ConsumerInfo, now: number): number | null {
  const last = consumer.delivered?.last_active
  return last == null ? null : now - last
}

function streamFill(stream: StreamInfo): number {
  const { max_msgs: maxMsgs, max_bytes: maxBytes } = stream.config
  const byMsgs = maxMsgs > 0 ? stream.messages / maxMsgs : 0
  const byBytes = maxBytes > 0 ? stream.bytes / maxBytes : 0
  return Math.max(byMsgs, byBytes)
}

export function consumerIssues(consumer: ConsumerInfo, stream: StreamInfo | undefined, now: number): ConsumerIssue[] {
  const issues: ConsumerIssue[] = []
  const pending = consumer.num_pending
  const paused = consumer.paused === true
  const blocked = ackLimitReached(consumer)

  if (paused) issues.push(pausedIssue(consumer))

  if (blocked) {
    issues.push({
      kind: 'ack_limit',
      severity: 'error',
      label: 'Ack limit reached',
      detail:
        `${messages(consumer.num_ack_pending)} wait for an ack, the most this consumer allows (max ack pending ` +
        `${consumer.config?.max_ack_pending}). Nothing new is delivered until clients ack them or the ack wait runs out.`,
    })
  }

  if (!paused && pending > 0 && consumerKind(consumer) === 'push' && !consumer.push_bound) {
    issues.push({
      kind: 'no_subscriber',
      severity: 'error',
      label: 'No subscriber',
      detail: `${messages(pending)} wait, and nobody is subscribed to ${consumer.config?.deliver_subject}, where this push consumer delivers.`,
    })
  }

  if (!paused && !blocked && pending > 0 && consumerKind(consumer) === 'pull' && !(consumer.num_waiting ?? 0)) {
    const idle = idleFor(consumer, now)
    if (idle == null || idle > IDLE_PULL_MS) {
      issues.push({
        kind: 'nobody_pulling',
        severity: 'warning',
        label: 'Nobody pulling',
        detail:
          idle == null
            ? `${messages(pending)} wait, and no client has pulled yet.`
            : `${messages(pending)} wait, and no client has pulled for ${formatDuration(Math.floor(idle / 1000))}.`,
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

  if (stream && pending > 0 && (stream.config.discard ?? 'old') === 'old') {
    const fill = streamFill(stream)
    if (fill >= STREAM_FULL_RATIO) {
      issues.push({
        kind: 'stream_full',
        severity: 'warning',
        label: 'May lose messages',
        detail:
          `Stream ${stream.name} is ${Math.min(100, Math.floor(fill * 100))}% full and drops its oldest messages when full, ` +
          `so some of the ${messages(pending)} this consumer has not received yet may be gone before it gets them.`,
      })
    }
  }

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
