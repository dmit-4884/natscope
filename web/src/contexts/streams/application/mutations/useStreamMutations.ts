import { useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from '@/utils/toast'
import * as api from '@/api/management'
import { getErrorMessage } from '@/api/errors'
import type {
  StreamCreateRequest,
  StreamUpdateRequest,
  StreamPurgeRequest,
} from '@/types/management'
import { streamKeys } from '../queries/streamKeys'

export function useCreateStream(connectionId: string | undefined) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (config: StreamCreateRequest) => {
      if (!connectionId) throw new Error('No connection')
      return api.createStream(connectionId, config)
    },
    onSuccess: (stream) => {
      toast.success(`Stream "${stream.name}" created`)
      queryClient.invalidateQueries({ queryKey: streamKeys.list(connectionId) })
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
      queryClient.invalidateQueries({ queryKey: streamKeys.list(connectionId) })
      queryClient.invalidateQueries({ queryKey: streamKeys.detail(connectionId, stream.name) })
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
      queryClient.invalidateQueries({ queryKey: streamKeys.list(connectionId) })
      queryClient.invalidateQueries({ queryKey: streamKeys.messages(connectionId, name) })
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
      queryClient.invalidateQueries({ queryKey: streamKeys.list(connectionId) })
      queryClient.invalidateQueries({ queryKey: streamKeys.detail(connectionId, name) })
      queryClient.invalidateQueries({ queryKey: streamKeys.messages(connectionId, name) })
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
      queryClient.invalidateQueries({ queryKey: streamKeys.list(connectionId) })
      queryClient.invalidateQueries({ queryKey: streamKeys.detail(connectionId, name) })
    },
    onError: (error: Error) => {
      toast.error(`Failed to seal stream: ${getErrorMessage(error)}`)
    },
  })
}
