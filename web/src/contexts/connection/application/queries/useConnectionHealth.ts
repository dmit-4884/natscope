import { getHealth } from '@/api/stats'
import { useConnectionQuery } from '@/hooks/useConnectionQuery'
import { resolveConnectionStatus, type ConnectionStatus } from '../../domain/value-objects/ConnectionStatus'

const HEALTH_POLL_INTERVAL_MS = 5_000

export function useConnectionHealth(connectionId: string | null) {
  const { data: health, error, isPending, refetch } = useConnectionQuery({
    key: ['health'],
    connectionId,
    fetcher: (signal) => getHealth(connectionId!, signal),
    retry: false,
    refetchOnWindowFocus: 'always',
    refetchInterval: HEALTH_POLL_INTERVAL_MS,
    staleTime: 0,
  })

  const status: ConnectionStatus = resolveConnectionStatus({
    reported: health?.status,
    isPending: !!connectionId && isPending,
    hasError: !!error,
  })

  return {
    status,
    rtt: health?.rtt,
    error: health?.error,
    queryError: error,
    serverVersion: health?.server_version,
    refetch,
  }
}
