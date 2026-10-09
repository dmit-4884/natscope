import { useQuery, type Query, type QueryKey, type UseQueryResult } from '@tanstack/react-query'

/**
 * Shared key prefix for every connection-scoped query. Switching connection
 * cancels everything under this prefix in one call, aborting in-flight queries.
 */
export const CONNECTION_QUERY_PREFIX = 'conn' as const

export interface UseConnectionQueryOptions<T> {
  /** Connection-scoped key suffix, e.g. ['streams'] or ['messages', streamName, params]. */
  key: readonly unknown[]
  /** Active connection id. Query is disabled while null/empty. */
  connectionId: string | null | undefined
  /** Fetcher that MUST forward the AbortSignal to the gRPC client. */
  fetcher: (signal: AbortSignal) => Promise<T>
  /** AND-combined with a non-empty connectionId; null check not needed. */
  enabled?: boolean
  staleTime?: number
  gcTime?: number
  refetchOnMount?: boolean | 'always'
  refetchOnWindowFocus?: boolean | 'always'
  refetchOnReconnect?: boolean | 'always'
  refetchInterval?: number | false | ((data: T | undefined, error: unknown) => number | false)
  retry?: boolean | number
  /** Keeps the last answer of the same connection on screen while this key loads, when the previous key suffix passes. */
  keepPreviousWhen?: (previousKey: readonly unknown[]) => boolean
}

/**
 * useQuery for connection-scoped data: prefixes the key with
 * CONNECTION_QUERY_PREFIX + connectionId (so a switch cancels all in-flight),
 * forwards the AbortSignal through `fetcher` (cancel closes the stream), and
 * disables while connectionId is empty. Don't put connectionId in `key` — the
 * wrapper injects it.
 */
export function useConnectionQuery<T>(opts: UseConnectionQueryOptions<T>): UseQueryResult<T> {
  const connectionId = opts.connectionId ?? null
  const queryKey: QueryKey = [CONNECTION_QUERY_PREFIX, connectionId, ...opts.key]
  const interval = opts.refetchInterval
  const keepPreviousWhen = opts.keepPreviousWhen
  const tuned = {
    staleTime: opts.staleTime,
    gcTime: opts.gcTime,
    refetchOnMount: opts.refetchOnMount,
    refetchOnWindowFocus: opts.refetchOnWindowFocus,
    refetchOnReconnect: opts.refetchOnReconnect,
    refetchInterval: typeof interval === 'function' ? (query: Query<T>) => interval(query.state.data, query.state.error) : interval,
    retry: opts.retry,
    placeholderData: keepPreviousWhen
      ? (previous: T | undefined, previousQuery: Query<T> | undefined) => {
          const key = previousQuery?.queryKey
          return key && key[1] === connectionId && keepPreviousWhen(key.slice(2)) ? previous : undefined
        }
      : undefined,
  }

  return useQuery<T>({
    queryKey,
    queryFn: ({ signal }) => opts.fetcher(signal),
    enabled: !!connectionId && (opts.enabled ?? true),
    ...Object.fromEntries(Object.entries(tuned).filter(([, value]) => value !== undefined)),
  })
}
