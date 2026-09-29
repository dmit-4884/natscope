import { useQuery } from '@tanstack/react-query'
import { getSidebarLayout, type SidebarLayout } from '@/api/connections'
import { connectionKeys } from './connectionKeys'

export type { SectionLayout, SidebarLayout, SidebarSection } from '@/api/connections'

export const EMPTY_SIDEBAR_LAYOUT: SidebarLayout = {
  streams: { pinned: [], order: [] },
  kv: { pinned: [], order: [] },
  objects: { pinned: [], order: [] },
}

export function useSidebarLayout(connectionId: string) {
  const { data } = useQuery({
    queryKey: connectionKeys.sidebarLayout(connectionId),
    queryFn: ({ signal }) => getSidebarLayout(connectionId, signal),
    enabled: !!connectionId,
    staleTime: Infinity,
  })
  return data ?? EMPTY_SIDEBAR_LAYOUT
}
