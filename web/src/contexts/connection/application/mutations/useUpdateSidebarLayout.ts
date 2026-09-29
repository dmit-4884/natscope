import { useMutation, useQueryClient } from '@tanstack/react-query'
import { getErrorMessage } from '@/api/errors'
import { updateSidebarLayout, type SidebarLayout } from '@/api/connections'
import { toast } from '@/utils/toast'
import { connectionKeys } from '../queries/connectionKeys'
import { EMPTY_SIDEBAR_LAYOUT } from '../queries/useSidebarLayout'

export function useUpdateSidebarLayout(connectionId: string) {
  const queryClient = useQueryClient()
  const queryKey = connectionKeys.sidebarLayout(connectionId)

  return useMutation({
    mutationFn: (patch: Partial<SidebarLayout>) => updateSidebarLayout(connectionId, patch),
    onMutate: async (patch) => {
      await queryClient.cancelQueries({ queryKey })
      const previous = queryClient.getQueryData<SidebarLayout>(queryKey)
      queryClient.setQueryData<SidebarLayout>(queryKey, { ...(previous ?? EMPTY_SIDEBAR_LAYOUT), ...patch })
      return { previous }
    },
    onError: (error, _patch, context) => {
      queryClient.setQueryData(queryKey, context?.previous)
      toast.error(`Failed to save the sidebar order: ${getErrorMessage(error)}`)
    },
  })
}
