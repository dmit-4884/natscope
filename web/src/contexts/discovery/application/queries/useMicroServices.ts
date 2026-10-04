import { listServices } from '@/api/discovery'
import { useConnectionQuery } from '@/hooks/useConnectionQuery'
import { isDenied } from '@/shared/domain/access'

export const MICRO_REFRESH_MS = 5000

export function useMicroServices(connectionId: string | null, { autoRefresh }: { autoRefresh: boolean }) {
  return useConnectionQuery({
    key: ['discovery', 'services'],
    connectionId,
    fetcher: (signal) => listServices(connectionId!, signal),
    refetchInterval: autoRefresh ? (data) => (data && isDenied(data.info_access) ? false : MICRO_REFRESH_MS) : false,
    refetchOnWindowFocus: false,
    retry: false,
  })
}
