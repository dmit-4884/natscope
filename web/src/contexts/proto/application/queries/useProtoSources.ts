import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  getProtoSources,
  getProtoSource,
  createProtoSource,
  updateProtoSource,
  deleteProtoSource,
  setSourceEnabled,
  setWatcher,
  compileLocal,
  validateLocalPath,
  compileFiles,
  getProtoSourceTags,
  getProtoSelections,
  selectProtoVersion,
  deleteProtoSelection,
  loadProtoFiles,
  type CreateProtoSourceRequest,
  type UpdateProtoSourceRequest,
} from '@/api/protoSources'
import { protoKeys } from './useProto'

const protoSourcesKeys = {
  all: ['protoSources'] as const,
  sources: () => [...protoSourcesKeys.all, 'sources'] as const,
  source: (id: string) => [...protoSourcesKeys.all, 'source', id] as const,
  tags: (sourceId: string) => [...protoSourcesKeys.all, 'tags', sourceId] as const,
  selections: () => [...protoSourcesKeys.all, 'selections'] as const,
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
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: protoSourcesKeys.sources() })
      queryClient.invalidateQueries({ queryKey: protoSourcesKeys.selections() })
      queryClient.invalidateQueries({ queryKey: protoKeys.messages() })
      // Toggling a source changes active schema-conflicts, flipping mapping
      // health badges; without this they stay stale until next reload.
      queryClient.invalidateQueries({ queryKey: ['mappings'] })
    },
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

export function useCompileLocal() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ sourceId }: { sourceId: string }) => compileLocal(sourceId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: protoSourcesKeys.sources() })
      queryClient.invalidateQueries({ queryKey: protoSourcesKeys.selections() })
      queryClient.invalidateQueries({ queryKey: protoKeys.messages() })
      queryClient.invalidateQueries({ queryKey: ['mappings'] })
    },
  })
}

export function useValidateLocalPath() {
  return useMutation({
    mutationFn: ({ path }: { path: string }) => validateLocalPath(path),
  })
}

export function useCompileFiles() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ sourceId }: { sourceId: string }) => compileFiles(sourceId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: protoSourcesKeys.sources() })
      queryClient.invalidateQueries({ queryKey: protoKeys.messages() })
      queryClient.invalidateQueries({ queryKey: ['mappings'] })
    },
  })
}

export function useProtoSourceTags(sourceId: string | null) {
  return useQuery({
    queryKey: protoSourcesKeys.tags(sourceId || ''),
    queryFn: () => getProtoSourceTags(sourceId!),
    enabled: !!sourceId,
  })
}

export function useProtoSelections() {
  return useQuery({
    queryKey: protoSourcesKeys.selections(),
    queryFn: getProtoSelections,
  })
}

export function useSelectProtoVersion() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({ sourceId, tag }: { sourceId: string; tag: string }) => {
      const selection = await selectProtoVersion(sourceId, tag)
      await loadProtoFiles()
      return selection
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: protoSourcesKeys.selections() })
      queryClient.invalidateQueries({ queryKey: protoKeys.messages() })
      queryClient.invalidateQueries({ queryKey: protoSourcesKeys.all })
    },
  })
}

export function useDeleteProtoSelection() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: deleteProtoSelection,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: protoSourcesKeys.selections() })
    },
  })
}

