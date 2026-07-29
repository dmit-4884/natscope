/** React Query key factory; uniform invalidation semantics across contexts. */

type QueryKeyFilter = Record<string, unknown> | undefined

export interface QueryKeys<N extends string> {
  /** Root key — invalidates every query for the namespace. */
  all: readonly [N]
  /** Parent key for all list queries in the namespace. */
  lists: () => readonly [N, 'list']
  /** Specific list keyed by filter. */
  list: (filter?: QueryKeyFilter) => readonly [N, 'list', QueryKeyFilter]
  /** Parent key for all detail queries. */
  details: () => readonly [N, 'detail']
  /** Specific detail keyed by entity id. */
  detail: (id: string) => readonly [N, 'detail', string]
  /** Custom subtree — for rare cases (stats, health, etc.). */
  custom: <T extends readonly unknown[]>(...parts: T) => readonly [N, ...T]
}

export function createQueryKeys<N extends string>(namespace: N): QueryKeys<N> {
  return {
    all: [namespace] as const,
    lists: () => [namespace, 'list'] as const,
    list: (filter?: QueryKeyFilter) => [namespace, 'list', filter] as const,
    details: () => [namespace, 'detail'] as const,
    detail: (id: string) => [namespace, 'detail', id] as const,
    custom: <T extends readonly unknown[]>(...parts: T) => [namespace, ...parts] as const,
  }
}
