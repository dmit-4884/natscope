import { getPublishHistory, type PublishHistoryEntry } from '@/api/publishHistory'
import { useConnectionQuery } from '@/hooks/useConnectionQuery'

export type { PublishHistoryEntry }

interface PublishHistoryOptions {
  stream?: string
  pageSize?: number
}

export function usePublishHistory(
  connectionId: string | null | undefined,
  connectionUrl?: string,
  { stream, pageSize }: PublishHistoryOptions = {},
) {
  return useConnectionQuery({
    key: ['publishHistory', connectionId ?? null, connectionUrl ?? null, stream ?? null],
    connectionId: connectionId ?? null,
    fetcher: (signal) => getPublishHistory(connectionId ?? undefined, connectionUrl, stream, signal, pageSize),
  })
}
