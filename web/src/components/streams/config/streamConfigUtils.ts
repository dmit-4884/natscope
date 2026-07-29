import type { StreamCreateRequest } from '@/types/management'
import type { StreamDetail } from '@/types/nats'

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
    mirror_direct: stream.config?.mirror_direct,
    discard_new_per_subject: stream.config?.discard_new_per_subject,
    metadata: stream.config?.metadata,
  }
}
