import { useQuery } from '@tanstack/react-query'
import { listMappings, getMappingHealth } from '@/api/mappings'
import { mappingKeys } from './mappingKeys'

const MAPPINGS_STALE_TIME = 30_000
const MAPPINGS_GC_TIME = 5 * 60_000

export function useMappingItems() {
  return useQuery({
    queryKey: mappingKeys.list(),
    queryFn: listMappings,
    select: (result) => result.items,
    staleTime: MAPPINGS_STALE_TIME,
    gcTime: MAPPINGS_GC_TIME,
  })
}

export function useMappingHealthBatch(ids: string[]) {
  return useQuery({
    queryKey: mappingKeys.health(ids),
    queryFn: () => getMappingHealth(ids),
    enabled: ids.length > 0,
    staleTime: MAPPINGS_STALE_TIME,
    gcTime: MAPPINGS_GC_TIME,
    placeholderData: (prev) => prev,
  })
}
