import { useQuery, useMutation, useQueryClient, type QueryClient } from '@tanstack/react-query'
import {
  getProtoSources,
  getProtoSource,
  createProtoSource,
  updateProtoSource,
  deleteProtoSource,
  setSourceEnabled,
  setWatcher,
  validateLocalPath,
  validateRepository,
  refreshSource,
  listSourceRefs,
  selectSourceRef,
  listSourceRevisions,
  uploadSchema,
  type CreateProtoSourceRequest,
  type SchemaUploadContent,
  type UpdateProtoSourceRequest,
} from '@/api/protoSources'
import { protoKeys } from './useProto'

const protoSourcesKeys = {
  all: ['protoSources'] as const,
  sources: () => [...protoSourcesKeys.all, 'sources'] as const,
  source: (id: string) => [...protoSourcesKeys.all, 'source', id] as const,
  refs: (sourceId: string) => [...protoSourcesKeys.all, 'refs', sourceId] as const,
  revisions: (sourceId: string) => [...protoSourcesKeys.all, 'revisions', sourceId] as const,
}

function invalidateSchemas(queryClient: QueryClient) {
  queryClient.invalidateQueries({ queryKey: protoSourcesKeys.all })
  queryClient.invalidateQueries({ queryKey: protoKeys.all })
  queryClient.invalidateQueries({ queryKey: ['mappings'] })
}

export function useProtoSources() {
  return useQuery({
    queryKey: protoSourcesKeys.sources(),
    queryFn: async () => {
      const result = await getProtoSources()
      return result.items || []
    },
  })
}

export function useProtoSource(id: string | null) {
  return useQuery({
    queryKey: protoSourcesKeys.source(id || ''),
    queryFn: () => getProtoSource(id!),
    enabled: !!id,
  })
}

export function useCreateProtoSource() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: CreateProtoSourceRequest) => createProtoSource(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: protoSourcesKeys.sources() })
    },
  })
}

export function useUpdateProtoSource() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateProtoSourceRequest }) =>
      updateProtoSource(id, data),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: protoSourcesKeys.sources() })
      queryClient.invalidateQueries({ queryKey: protoSourcesKeys.source(variables.id) })
    },
  })
}

export function useDeleteProtoSource() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id }: { id: string }) => deleteProtoSource(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: protoSourcesKeys.sources() })
    },
  })
}

export function useSetSourceEnabled() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ sourceId, enabled }: { sourceId: string; enabled: boolean }) =>
      setSourceEnabled(sourceId, enabled),
    onSuccess: () => invalidateSchemas(queryClient),
  })
}

export function useSetWatcher() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ sourceId, enabled }: { sourceId: string; enabled: boolean }) =>
      setWatcher(sourceId, enabled),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: protoSourcesKeys.sources() })
    },
  })
}

export function useRefreshSource() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ sourceId }: { sourceId: string }) => refreshSource(sourceId),
    onSuccess: () => invalidateSchemas(queryClient),
  })
}

export function useValidateLocalPath() {
  return useMutation({
    mutationFn: ({ path }: { path: string }) => validateLocalPath(path),
  })
}

export function useValidateRepository() {
  return useMutation({
    mutationFn: ({ repository, token }: { repository: string; token?: string }) =>
      validateRepository(repository, token),
  })
}

export function useSourceRefs(sourceId: string | null) {
  return useQuery({
    queryKey: protoSourcesKeys.refs(sourceId || ''),
    queryFn: () => listSourceRefs(sourceId!),
    enabled: !!sourceId,
  })
}

export function useSelectSourceRef() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ sourceId, ref }: { sourceId: string; ref: string }) => selectSourceRef(sourceId, ref),
    onSuccess: () => invalidateSchemas(queryClient),
  })
}

export function useSourceRevisions(sourceId: string | null) {
  return useQuery({
    queryKey: protoSourcesKeys.revisions(sourceId || ''),
    queryFn: () => listSourceRevisions(sourceId!),
    enabled: !!sourceId,
  })
}

export function useUploadSchema() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ sourceId, content }: { sourceId: string; content: SchemaUploadContent }) => uploadSchema(sourceId, content),
    onSuccess: () => invalidateSchemas(queryClient),
  })
}
