import { getHealth } from '@/api/stats'
import { useConnectionQuery } from '@/hooks/useConnectionQuery'

export function useConnectionHealth(connectionId: string | null) {
  const { data: health, refetch } = useConnectionQuery({
    key: ['health'],
    connectionId,
    fetcher: (signal) => getHealth(connectionId!, signal),
  })

  return {
    status: health?.status ?? 'disconnected',
    rtt: health?.rtt,
    error: health?.error,
    serverVersion: health?.server_version,
    refetch,
  }
}
