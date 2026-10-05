import { getAccessDenial } from '@/api/errors'
import { getConsumersOverview } from '@/api/stats'
import { useConnectionQuery } from '@/hooks/useConnectionQuery'

export const CONSUMERS_REFRESH_MS = 5000

export function useConsumersOverview(connectionId: string | null, { autoRefresh }: { autoRefresh: boolean }) {
  return useConnectionQuery({
    key: ['consumers-overview'],
    connectionId,
    fetcher: (signal) => getConsumersOverview(connectionId!, signal),
    refetchInterval: autoRefresh ? (_data, error) => (getAccessDenial(error) ? false : CONSUMERS_REFRESH_MS) : false,
    refetchOnWindowFocus: false,
    retry: false,
  })
}
