import { CONNECTION_QUERY_PREFIX } from '@/hooks/useConnectionQuery'

/**
 * KV query keys. Must match useConnectionQuery's shape
 * [CONNECTION_QUERY_PREFIX, connectionId, ...userKey] so mutation
 * invalidations hit the entries written by the query hooks.
 */
export const kvKeys = {
  all: [CONNECTION_QUERY_PREFIX] as const,
  buckets: (connectionId: string | undefined) =>
    [CONNECTION_QUERY_PREFIX, connectionId ?? null, 'kv', 'buckets'] as const,
  bucket: (connectionId: string | undefined, bucket: string | undefined) =>
    [CONNECTION_QUERY_PREFIX, connectionId ?? null, 'kv', 'bucket', bucket] as const,
  keys: (connectionId: string | undefined, bucket: string | undefined) =>
    [CONNECTION_QUERY_PREFIX, connectionId ?? null, 'kv', 'keys', bucket] as const,
  key: (
    connectionId: string | undefined,
    bucket: string | undefined,
    key: string | undefined,
  ) => [CONNECTION_QUERY_PREFIX, connectionId ?? null, 'kv', 'key', bucket, key] as const,
  // Prefix without the key — invalidates every key's history in the bucket.
  historyAll: (connectionId: string | undefined, bucket: string | undefined) =>
    [CONNECTION_QUERY_PREFIX, connectionId ?? null, 'kv', 'history', bucket] as const,
  history: (
    connectionId: string | undefined,
    bucket: string | undefined,
    key: string | undefined,
  ) => [CONNECTION_QUERY_PREFIX, connectionId ?? null, 'kv', 'history', bucket, key] as const,
}
