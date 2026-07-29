import * as api from '@/api/management'
import { useConnectionQuery } from '@/hooks/useConnectionQuery'

export function useConsumers(
  connectionId: string | undefined,
  streamName: string | undefined,
) {
  return useConnectionQuery({
    key: ['consumers', streamName],
    connectionId: connectionId ?? null,
    enabled: !!streamName,
    fetcher: (signal) => api.listConsumers(connectionId!, streamName!, signal),
  })
}
