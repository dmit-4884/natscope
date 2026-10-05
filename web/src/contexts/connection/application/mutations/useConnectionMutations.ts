import { useMutation, useQueryClient } from '@tanstack/react-query'
import {
  createConnection,
  updateConnection,
  deleteConnection,
  testConnection,
  duplicateConnection,
  importCliContexts,
  type CliContextFile,
  type CreateConnectionRequest,
  type UpdateConnectionRequest,
  type TestConnectionRequest,
} from '@/api/connections'
import { connectionKeys } from '../queries/connectionKeys'

export function useCreateConnection({ silent = false }: { silent?: boolean } = {}) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (connection: CreateConnectionRequest) => createConnection(connection),
    meta: { silent },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: connectionKeys.all })
    },
  })
}

export function useUpdateConnection({ silent = false }: { silent?: boolean } = {}) {
  const queryClient = useQueryClient()

  return useMutation({
    meta: { silent },
    mutationFn: ({ id, connection }: { id: string; connection: UpdateConnectionRequest }) =>
      updateConnection(id, connection),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: connectionKeys.all })
    },
  })
}

export function useDeleteConnection() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ id }: { id: string }) => deleteConnection(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: connectionKeys.all })
    },
  })
}

export function useTestConnection({ silent = false }: { silent?: boolean } = {}) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (req: TestConnectionRequest) => testConnection(req),
    meta: { silent },
    onSettled: (_data, _err, req) => {
      if (req.connectionId) {
        queryClient.invalidateQueries({ queryKey: connectionKeys.all })
      }
    },
  })
}

export function useDuplicateConnection() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ id, name }: { id: string; name: string }) => duplicateConnection(id, name),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: connectionKeys.all })
    },
  })
}

export function useImportCliContexts() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ names, files }: { names: string[]; files: CliContextFile[] }) => importCliContexts(names, files),
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: connectionKeys.all })
    },
  })
}
