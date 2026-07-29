/**
 * Stream-scoped state keyed by (connectionUrl, streamName). connectionUrl over
 * connectionId so a recreated connection with the same URL inherits drafts.
 */
export interface StreamScope {
  connectionUrl: string | null
  streamName: string
}

/** Stable key used both for storage and for lookup. */
export function streamKey(scope: StreamScope): string {
  return `${scope.connectionUrl ?? ''}:${scope.streamName}`
}

/** True when the scope refers to an addressable stream (UI can read/write). */
export function isScopeReady(scope: StreamScope | null | undefined): scope is StreamScope {
  return !!scope && !!scope.streamName
}
