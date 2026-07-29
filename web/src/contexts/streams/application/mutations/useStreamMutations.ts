import { useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from '@/utils/toast'
import * as api from '@/api/management'
import { getErrorMessage } from '@/api/errors'
import { CONNECTION_QUERY_PREFIX } from '@/hooks/useConnectionQuery'
import type {
  StreamCreateRequest,
  StreamUpdateRequest,
  StreamPurgeRequest,
} from '@/types/management'

// Cache-invalidation keys must match useConnectionQuery's prefix
// ([CONNECTION_QUERY_PREFIX, connectionId, …]) or they silently miss.
const streamsListKey = (connectionId: string | undefined) =>
  [CONNECTION_QUERY_PREFIX, connectionId ?? null, 'streams'] as const
const streamDetailKey = (connectionId: string | undefined, name: string) =>
  [CONNECTION_QUERY_PREFIX, connectionId ?? null, 'stream', name] as const
// Prefix key matching every messages query for a stream (see messages adapter).
const streamMessagesKey = (connectionId: string | undefined, name: string) =>
  [CONNECTION_QUERY_PREFIX, connectionId ?? null, 'messages', name] as const

export function useCreateStream(connectionId: string | undefined) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (config: StreamCreateRequest) => {
      if (!connectionId) throw new Error('No connection')
      return api.createStream(connectionId, config)
    },
    onSuccess: (stream) => {
      toast.success(`Stream "${stream.name}" created`)
      queryClient.invalidateQueries({ queryKey: streamsListKey(connectionId) })
    },
    onError: (error: Error) => {
      toast.error(`Failed to create stream: ${getErrorMessage(error)}`)
    },
  })
}

export function useUpdateStream(connectionId: string | undefined) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ name, config }: { name: string; config: StreamUpdateRequest }) => {
      if (!connectionId) throw new Error('No connection')
      return api.updateStream(connectionId, name, config)
    },
    onSuccess: (stream) => {
      toast.success(`Stream "${stream.name}" updated`)
      queryClient.invalidateQueries({ queryKey: streamsListKey(connectionId) })
      queryClient.invalidateQueries({ queryKey: streamDetailKey(connectionId, stream.name) })
    },
    onError: (error: Error) => {
      toast.error(`Failed to update stream: ${getErrorMessage(error)}`)
    },
  })
}

export function useDeleteStream(connectionId: string | undefined) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (name: string) => {
      if (!connectionId) throw new Error('No connection')
      return api.deleteStream(connectionId, name)
    },
    onSuccess: (_, name) => {
      toast.success(`Stream "${name}" deleted`)
      queryClient.invalidateQueries({ queryKey: streamsListKey(connectionId) })
      queryClient.invalidateQueries({ queryKey: streamMessagesKey(connectionId, name) })
    },
    onError: (error: Error) => {
      toast.error(`Failed to delete stream: ${getErrorMessage(error)}`)
    },
  })
}

export function usePurgeStream(connectionId: string | undefined) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ name, options }: { name: string; options?: StreamPurgeRequest }) => {
      if (!connectionId) throw new Error('No connection')
      return api.purgeStream(connectionId, name, options)
    },
    onSuccess: (result, { name }) => {
      toast.success(`Stream "${name}" purged (${result.purged} messages)`)
      queryClient.invalidateQueries({ queryKey: streamsListKey(connectionId) })
      queryClient.invalidateQueries({ queryKey: streamDetailKey(connectionId, name) })
      queryClient.invalidateQueries({ queryKey: streamMessagesKey(connectionId, name) })
    },
    onError: (error: Error) => {
      toast.error(`Failed to purge stream: ${getErrorMessage(error)}`)
    },
  })
}

export function useSealStream(connectionId: string | undefined) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (name: string) => {
      if (!connectionId) throw new Error('No connection')
      return api.sealStream(connectionId, name)
    },
    onSuccess: (_, name) => {
      toast.success(`Stream "${name}" sealed`)
      queryClient.invalidateQueries({ queryKey: streamsListKey(connectionId) })
      queryClient.invalidateQueries({ queryKey: streamDetailKey(connectionId, name) })
    },
    onError: (error: Error) => {
      toast.error(`Failed to seal stream: ${getErrorMessage(error)}`)
    },
  })
}
