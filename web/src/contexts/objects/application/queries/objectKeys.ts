import { CONNECTION_QUERY_PREFIX } from '@/hooks/useConnectionQuery'

/**
 * Object store query keys. Must match the shape produced by `useConnectionQuery`:
 *   [CONNECTION_QUERY_PREFIX, connectionId, ...userKey]
 * Otherwise mutation invalidations silently miss the actual cache entries.
 */
export const objectKeys = {
  all: [CONNECTION_QUERY_PREFIX] as const,
  buckets: (connectionId: string | undefined) =>
    [CONNECTION_QUERY_PREFIX, connectionId ?? null, 'objects', 'buckets'] as const,
  bucket: (connectionId: string | undefined, bucket: string | undefined) =>
    [CONNECTION_QUERY_PREFIX, connectionId ?? null, 'objects', 'bucket', bucket] as const,
  list: (connectionId: string | undefined, bucket: string | undefined) =>
    [CONNECTION_QUERY_PREFIX, connectionId ?? null, 'objects', 'list', bucket] as const,
  object: (
    connectionId: string | undefined,
    bucket: string | undefined,
    name: string | undefined,
  ) => [CONNECTION_QUERY_PREFIX, connectionId ?? null, 'objects', 'object', bucket, name] as const,
}
