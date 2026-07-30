import { CONNECTION_QUERY_PREFIX } from '@/hooks/useConnectionQuery'

/**
 * Stream query keys. Must match useConnectionQuery's shape
 * [CONNECTION_QUERY_PREFIX, connectionId, ...userKey] so invalidations hit the
 * entries written by the query hooks.
 */
export const streamKeys = {
  list: (connectionId: string | null | undefined) =>
    [CONNECTION_QUERY_PREFIX, connectionId ?? null, 'streams'] as const,
  detail: (connectionId: string | null | undefined, name: string) =>
    [CONNECTION_QUERY_PREFIX, connectionId ?? null, 'stream', name] as const,
  consumers: (connectionId: string | null | undefined, name: string) =>
    [CONNECTION_QUERY_PREFIX, connectionId ?? null, 'consumers', name] as const,
  // Prefix key matching every paged/filtered messages query for a stream.
  messages: (connectionId: string | null | undefined, name: string) =>
    [CONNECTION_QUERY_PREFIX, connectionId ?? null, 'messages', name] as const,
}
