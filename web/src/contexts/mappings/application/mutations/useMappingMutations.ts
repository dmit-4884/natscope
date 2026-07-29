import { useMutation, useQueryClient, type QueryClient } from '@tanstack/react-query'
import { createMapping, bulkSaveMappings, deleteMapping } from '@/api/mappings'
import { CONNECTION_QUERY_PREFIX } from '@/hooks/useConnectionQuery'
import { mappingKeys } from '../queries/mappingKeys'

function invalidateAllMessageQueries(queryClient: QueryClient) {
  queryClient.invalidateQueries({
    predicate: (q) => {
      const k = q.queryKey
      return Array.isArray(k) && k[0] === CONNECTION_QUERY_PREFIX && k[2] === 'messages'
    },
  })
}

function invalidateMappings(queryClient: QueryClient) {
  queryClient.invalidateQueries({ queryKey: mappingKeys.all })
  invalidateAllMessageQueries(queryClient)
}

export function useCreateMapping() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      pattern,
      messageType,
      sourceId,
    }: {
      pattern: string
      messageType: string
      sourceId: string
    }) => createMapping(pattern, messageType, sourceId),
    onSuccess: () => invalidateMappings(queryClient),
  })
}

export function useBulkSaveMappings() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: bulkSaveMappings,
    onSuccess: () => invalidateMappings(queryClient),
  })
}

export function useDeleteMapping() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: deleteMapping,
    onSuccess: () => invalidateMappings(queryClient),
  })
}

export function useUpdateMapping() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async ({
      id,
      pattern,
      messageType,
      sourceId,
    }: {
      id: string
      pattern: string
      messageType: string
      sourceId: string
    }) => {
      const created = await createMapping(pattern, messageType, sourceId)
      try {
        await deleteMapping(id)
      } catch (err) {
        queryClient.invalidateQueries({ queryKey: mappingKeys.all })
        throw err
      }
      return created
    },
    onSuccess: () => invalidateMappings(queryClient),
  })
}
