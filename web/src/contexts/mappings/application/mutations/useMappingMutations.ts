import { useMutation, useQueryClient, type QueryClient } from '@tanstack/react-query'
import { createMapping, bulkSaveMappings, deleteMapping, updateMapping, type MappingPatch } from '@/api/mappings'
import type { Framing } from '@/api/framing'
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
      framing,
    }: {
      pattern: string
      messageType: string
      sourceId: string
      framing?: Framing
    }) => createMapping(pattern, messageType, sourceId, framing),
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
    mutationFn: ({ id, patch }: { id: string; patch: MappingPatch }) => updateMapping(id, patch),
    onSuccess: () => invalidateMappings(queryClient),
  })
}
