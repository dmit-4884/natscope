import type { StreamCreateRequest, StreamSource } from '@/types/management'
import type { ReplicaInfo, StreamDetail, StreamSourceRef } from '@/types/nats'
import { formatNsDuration, formatNumber } from '@/utils/formatters'
import { plural } from '@/utils/plural'

export const COMPRESSION_LABELS: Record<string, string> = { none: 'None', s2: 'S2' }

export interface ReplicaState {
  variant: 'success' | 'warning' | 'error'
  label: string | undefined
  detail: string
}

export function replicaState(replica: ReplicaInfo): ReplicaState {
  if (replica.offline) {
    const detail = replica.active > 0 ? `Offline, last seen ${formatNsDuration(replica.active)} ago` : 'Offline'
    return { variant: 'error', label: 'offline', detail }
  }
  if (!replica.current) {
    return replica.lag > 0
      ? { variant: 'warning', label: `${formatNumber(replica.lag)} behind`, detail: `${plural(replica.lag, 'operation')} behind the leader` }
      : { variant: 'warning', label: 'catching up', detail: 'Catching up with the leader' }
  }
  return { variant: 'success', label: undefined, detail: 'In sync with the leader' }
}

/** Format camelCase/PascalCase to readable text: "DiscardOld" → "Discard Old". */
export function formatConfigValue(value: string): string {
  if (!value) return '-'
  return value
    .replace(/([A-Z])/g, ' $1')
    .replace(/^./, (str) => str.toUpperCase())
    .trim()
}

export function hasMetadata(streamDetail: StreamDetail): boolean {
  return !!(streamDetail.config.metadata && Object.keys(streamDetail.config.metadata).length > 0)
}

export function isMirrorConfigured(value: Pick<StreamCreateRequest, 'mirror'>): boolean {
  return !!value.mirror?.name?.trim()
}

export function normalizeSubjects(subjects: string[] | undefined): string[] {
  return (subjects ?? []).map((s) => s.trim()).filter((s) => s !== '')
}

export function canCreateStream(value: StreamCreateRequest): boolean {
  if (!value.name?.trim()) return false
  return isMirrorConfigured(value) || normalizeSubjects(value.subjects).length > 0
}

function toFormSource(source: StreamSourceRef): StreamSource {
  return {
    name: source.name,
    opt_start_seq: source.opt_start_seq,
    filter_subject: source.filter_subject,
    external: source.external
      ? { api: source.external.api_prefix, deliver: source.external.deliver_prefix }
      : undefined,
  }
}

/** Converts a server-side StreamDetail to the form-friendly StreamCreateRequest shape. */
export function streamToConfig(stream: StreamDetail): StreamCreateRequest {
  return {
    name: stream.name,
    description: stream.description || '',
    subjects: stream.subjects || [],
    retention: (stream.config?.retention?.toLowerCase() || 'limits') as 'limits' | 'interest' | 'workqueue',
    storage: (stream.config?.storage?.toLowerCase() || 'file') as 'file' | 'memory',
    discard: (stream.config?.discard?.toLowerCase() || 'old') as 'old' | 'new',
    max_msgs: stream.config?.max_msgs ?? -1,
    max_bytes: stream.config?.max_bytes ?? -1,
    max_age: stream.config?.max_age ?? 0,
    max_msgs_per_subject: stream.config?.max_msgs_per_subject,
    max_msg_size: stream.config?.max_msg_size,
    num_replicas: stream.config?.num_replicas ?? 1,
    duplicate_window: stream.config?.duplicate_window,
    deny_delete: stream.config?.deny_delete,
    deny_purge: stream.config?.deny_purge,
    allow_direct: stream.config?.allow_direct,
    allow_rollup: stream.config?.allow_rollup_hdrs,
    compression: stream.config?.compression as 'none' | 's2' | undefined,
    max_consumers: stream.config?.max_consumers,
    allow_msg_ttl: stream.config?.allow_msg_ttl,
    allow_atomic_publish: stream.config?.allow_atomic,
    allow_msg_counter: stream.config?.allow_msg_counter,
    allow_msg_schedules: stream.config?.allow_msg_schedules,
    subject_delete_marker_ttl: stream.config?.subject_delete_marker_ttl,
    persist_mode: stream.config?.persist_mode ?? 'default',
    allow_batch_publish: stream.config?.allow_batched,
    mirror_direct: stream.config?.mirror_direct,
    discard_new_per_subject: stream.config?.discard_new_per_subject,
    metadata: stream.config?.metadata,
    mirror: stream.config?.mirror ? toFormSource(stream.config.mirror) : undefined,
    sources: stream.config?.sources?.map(toFormSource),
    republish: stream.config?.republish,
    subject_transform: stream.config?.subject_transform,
    consumer_limits: stream.config?.consumer_limits,
  }
}
