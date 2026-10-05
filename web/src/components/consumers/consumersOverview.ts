import type { ConsumersOverview } from '@/api/stats'
import type { ConsumerInfo, StreamInfo } from '@/types/nats'
import {
  compareByHealth,
  consumerIssues,
  consumerKind,
  consumerState,
  type ConsumerIssue,
  type ConsumerKind,
  type ConsumerState,
} from '../streams/consumers/consumerHealth'
import { getFilterSubjectsArray } from '../streams/consumers/consumerUtils'

export interface ConsumerRow {
  key: string
  consumer: ConsumerInfo
  stream: StreamInfo | undefined
  kind: ConsumerKind
  issues: ConsumerIssue[]
  state: ConsumerState
}

export function buildRows(overview: ConsumersOverview, now: number): ConsumerRow[] {
  const streams = new Map(overview.streams.map((s) => [s.name, s]))
  return overview.consumers
    .map((consumer) => {
      const stream = streams.get(consumer.stream_name ?? '')
      const issues = consumerIssues(consumer, stream, now)
      return {
        key: `${consumer.stream_name}/${consumer.name}`,
        consumer,
        stream,
        kind: consumerKind(consumer),
        issues,
        state: consumerState(consumer, issues),
      }
    })
    .sort(compareByHealth)
}

export function filterRows(rows: ConsumerRow[], query: string, problemsOnly: boolean): ConsumerRow[] {
  const needle = query.trim().toLowerCase()
  return rows.filter((row) => {
    if (problemsOnly && row.issues.length === 0) return false
    if (!needle) return true
    return [row.consumer.name, row.consumer.stream_name ?? '', ...getFilterSubjectsArray(row.consumer)].some((text) =>
      text.toLowerCase().includes(needle),
    )
  })
}

export function formatAgo(at: number | undefined, now: number): string {
  if (at == null) return 'never'
  const seconds = Math.max(0, Math.floor((now - at) / 1000))
  if (seconds < 5) return 'just now'
  if (seconds < 60) return `${seconds}s ago`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`
  if (seconds < 86_400) return `${Math.floor(seconds / 3600)}h ago`
  return `${Math.floor(seconds / 86_400)}d ago`
}

function exportRecord(row: ConsumerRow) {
  const { consumer } = row
  const lastDelivery = consumer.delivered?.last_active
  return {
    stream: consumer.stream_name ?? '',
    consumer: consumer.name,
    type: row.kind,
    durable: !!consumer.config?.durable_name,
    filter_subjects: getFilterSubjectsArray(consumer),
    pending: consumer.num_pending,
    ack_pending: consumer.num_ack_pending,
    max_ack_pending: consumer.config?.max_ack_pending ?? 0,
    redelivered: consumer.num_redelivered ?? 0,
    waiting: consumer.num_waiting ?? 0,
    last_delivery: lastDelivery != null ? new Date(lastDelivery).toISOString() : '',
    status: row.state,
  }
}

function csvField(value: string | number): string {
  const text = typeof value === 'string' && /^[=+\-@\t\r]/.test(value) ? `'${value}` : String(value)
  return /[",\n\r]/.test(text) ? `"${text.replace(/"/g, '""')}"` : text
}

export function rowsToCsv(rows: ConsumerRow[]): string {
  const header = [
    'stream',
    'consumer',
    'type',
    'durable',
    'filter_subjects',
    'pending',
    'ack_pending',
    'max_ack_pending',
    'redelivered',
    'waiting',
    'last_delivery',
    'status',
    'problems',
  ]
  const lines = rows.map((row) => {
    const r = exportRecord(row)
    return [
      r.stream,
      r.consumer,
      r.type,
      r.durable ? 'yes' : 'no',
      r.filter_subjects.join(' '),
      r.pending,
      r.ack_pending,
      r.max_ack_pending,
      r.redelivered,
      r.waiting,
      r.last_delivery,
      r.status,
      row.issues.map((i) => i.label).join('; '),
    ]
      .map(csvField)
      .join(',')
  })
  return [header.join(','), ...lines].join('\n')
}

export function rowsToJson(rows: ConsumerRow[]): string {
  return JSON.stringify(
    rows.map((row) => ({
      ...exportRecord(row),
      problems: row.issues.map(({ kind, severity, label, detail }) => ({ kind, severity, label, detail })),
    })),
    null,
    2,
  )
}
