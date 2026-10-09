import { useQuery } from '@tanstack/react-query'
import { getSidebarLayout, type SidebarLayout } from '@/api/connections'
import { connectionKeys } from './connectionKeys'

export type { SectionLayout, SidebarLayout, SidebarSection } from '@/api/connections'

export const EMPTY_SIDEBAR_LAYOUT: SidebarLayout = {
  streams: { pinned: [], order: [] },
  kv: { pinned: [], order: [] },
  objects: { pinned: [], order: [] },
}

function useSidebarLayoutQuery(connectionId: string) {
  return useQuery({
    queryKey: connectionKeys.sidebarLayout(connectionId),
    queryFn: ({ signal }) => getSidebarLayout(connectionId, signal),
    enabled: !!connectionId,
    staleTime: Infinity,
  })
}

export function useSidebarLayout(connectionId: string) {
  return useSidebarLayoutQuery(connectionId).data ?? EMPTY_SIDEBAR_LAYOUT
}

export function useSidebarLayoutPending(connectionId: string): boolean {
  return useSidebarLayoutQuery(connectionId).isPending && !!connectionId
}
