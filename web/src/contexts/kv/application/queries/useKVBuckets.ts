import * as api from '@/api/management'
import { useConnectionQuery } from '@/hooks/useConnectionQuery'

export function useKVBuckets(connectionId: string | undefined) {
  return useConnectionQuery({
    key: ['kv', 'buckets'],
    connectionId: connectionId ?? null,
    fetcher: (signal) => api.listKVBuckets(connectionId!, signal),
  })
}

export function useKVKeys(connectionId: string | undefined, bucket: string | undefined, filter = '', enabled = true) {
  return useConnectionQuery({
    key: ['kv', 'keys', bucket, filter],
    connectionId: connectionId ?? null,
    enabled: !!bucket && enabled,
    fetcher: (signal) => api.listKVKeys(connectionId!, bucket!, filter, signal),
    keepPreviousWhen: (previous) => previous[2] === bucket,
  })
}

export function useKVKey(
  connectionId: string | undefined,
  bucket: string | undefined,
  key: string | undefined,
) {
  return useConnectionQuery({
    key: ['kv', 'key', bucket, key],
    connectionId: connectionId ?? null,
    enabled: !!bucket && !!key,
    fetcher: (signal) => api.getKVKey(connectionId!, bucket!, key!, signal),
  })
}

// Pass key only while the history view is open — the fetch spins up an
// ephemeral consumer on the server, so it should stay on-demand.
export function useKVKeyHistory(
  connectionId: string | undefined,
  bucket: string | undefined,
  key: string | undefined,
) {
  return useConnectionQuery({
    key: ['kv', 'history', bucket, key],
    connectionId: connectionId ?? null,
    enabled: !!bucket && !!key,
    fetcher: (signal) => api.getKVKeyHistory(connectionId!, bucket!, key!, signal),
  })
}
