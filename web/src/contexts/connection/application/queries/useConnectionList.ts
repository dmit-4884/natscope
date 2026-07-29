import { useQuery } from '@tanstack/react-query'
import { getConnections } from '@/api/connections'
import { connectionKeys } from './connectionKeys'

export function useConnections(options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: connectionKeys.all,
    queryFn: getConnections,
    enabled: options?.enabled ?? true,
  })
}
