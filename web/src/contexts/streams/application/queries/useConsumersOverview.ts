import { useRef } from 'react'
import { getAccessDenial } from '@/api/errors'
import { getConsumersOverview } from '@/api/stats'
import { useConnectionQuery } from '@/hooks/useConnectionQuery'

export const CONSUMERS_REFRESH_MS = 5000

const CONSUMERS_REFRESH_PER_FETCH = 10

export function useConsumersOverview(connectionId: string | null, { autoRefresh }: { autoRefresh: boolean }) {
  const lastFetchMs = useRef(0)
  return useConnectionQuery({
    key: ['consumers-overview'],
    connectionId,
    fetcher: async (signal) => {
      const started = Date.now()
      try {
        return await getConsumersOverview(connectionId!, signal)
      } finally {
        lastFetchMs.current = Date.now() - started
      }
    },
    refetchInterval: autoRefresh
      ? (_data, error) =>
          getAccessDenial(error) ? false : Math.max(CONSUMERS_REFRESH_MS, lastFetchMs.current * CONSUMERS_REFRESH_PER_FETCH)
      : false,
    staleTime: 0,
  })
}
