import { useCallback, useRef } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { listServices, type MicroDiscovery } from '@/api/discovery'
import { useConnectionQuery } from '@/hooks/useConnectionQuery'
import { isDenied } from '@/shared/domain/access'
import { discoveryKeys } from './discoveryKeys'

export const MICRO_REFRESH_MS = 5000

export function useMicroServices(connectionId: string | null, { autoRefresh }: { autoRefresh: boolean }) {
  const queryClient = useQueryClient()
  const recheckStats = useRef(true)

  const query = useConnectionQuery({
    key: ['discovery', 'services'],
    connectionId,
    fetcher: async (signal) => {
      const previous = queryClient.getQueryData<MicroDiscovery>(discoveryKeys.services(connectionId))
      const skipStats = !recheckStats.current && isDenied(previous?.stats_access)
      recheckStats.current = false
      const discovery = await listServices(connectionId!, { skipStats }, signal)
      return skipStats ? { ...discovery, stats_access: previous?.stats_access } : discovery
    },
    refetchInterval: autoRefresh ? (data) => (data && isDenied(data.info_access) ? false : MICRO_REFRESH_MS) : false,
    refetchOnWindowFocus: false,
    retry: false,
  })

  const { refetch } = query
  const recheck = useCallback(() => {
    recheckStats.current = true
    return refetch()
  }, [refetch])

  return { query, recheck }
}
