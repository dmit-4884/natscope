import { useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from '@/utils/toast'
import * as api from '@/api/management'
import { getErrorMessage } from '@/api/errors'
import { CONNECTION_QUERY_PREFIX } from '@/hooks/useConnectionQuery'
import type { ConsumerCreateRequest, ConsumerUpdateRequest } from '@/types/management'
import type { ConsumerInfo } from '@/types/nats'

// Must mirror keys from useStreamDetail and useConsumers (both via
// useConnectionQuery).
const streamDetailKey = (connectionId: string | undefined, streamName: string | undefined) =>
  [CONNECTION_QUERY_PREFIX, connectionId ?? null, 'stream', streamName] as const
const consumersListKey = (connectionId: string | undefined, streamName: string | undefined) =>
  [CONNECTION_QUERY_PREFIX, connectionId ?? null, 'consumers', streamName] as const

export function useCreateConsumer(
  connectionId: string | undefined,
  streamName: string | undefined,
) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (config: ConsumerCreateRequest) => {
      if (!connectionId || !streamName) throw new Error('No connection or stream')
      return api.createConsumer(connectionId, streamName, config)
    },
    onSuccess: (consumer) => {
      toast.success(`Consumer "${consumer.name}" created`)
      queryClient.invalidateQueries({ queryKey: streamDetailKey(connectionId, streamName) })
      queryClient.invalidateQueries({ queryKey: consumersListKey(connectionId, streamName) })
    },
    onError: (error: Error) => {
      toast.error(`Failed to create consumer: ${getErrorMessage(error)}`)
    },
  })
}

export function useUpdateConsumer(
  connectionId: string | undefined,
  streamName: string | undefined,
) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ name, config }: { name: string; config: ConsumerUpdateRequest }) => {
      if (!connectionId || !streamName) throw new Error('No connection or stream')
      return api.updateConsumer(connectionId, streamName, name, config)
    },
    onSuccess: (consumer) => {
      toast.success(`Consumer "${consumer.name}" updated`)
      queryClient.invalidateQueries({ queryKey: streamDetailKey(connectionId, streamName) })
      queryClient.invalidateQueries({ queryKey: consumersListKey(connectionId, streamName) })
    },
    onError: (error: Error) => {
      toast.error(`Failed to update consumer: ${getErrorMessage(error)}`)
    },
  })
}

export function useDeleteConsumer(
  connectionId: string | undefined,
  streamName: string | undefined,
) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (consumerName: string) => {
      if (!connectionId || !streamName) throw new Error('No connection or stream')
      return api.deleteConsumer(connectionId, streamName, consumerName)
    },
    onSuccess: (_, consumerName) => {
      toast.success(`Consumer "${consumerName}" deleted`)
      queryClient.setQueryData<ConsumerInfo[]>(consumersListKey(connectionId, streamName), (old) =>
        old ? old.filter((c) => c.name !== consumerName) : old,
      )
      queryClient.invalidateQueries({ queryKey: streamDetailKey(connectionId, streamName) })
      queryClient.invalidateQueries({ queryKey: consumersListKey(connectionId, streamName) })
    },
    onError: (error: Error) => {
      toast.error(`Failed to delete consumer: ${getErrorMessage(error)}`)
    },
  })
}

export function usePauseConsumer(
  connectionId: string | undefined,
  streamName: string | undefined,
) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ name, pauseUntil }: { name: string; pauseUntil: Date }) => {
      if (!connectionId || !streamName) throw new Error('No connection or stream')
      return api.pauseConsumer(connectionId, streamName, name, pauseUntil)
    },
    onSuccess: (_result, { name }) => {
      toast.success(`Consumer "${name}" paused`)
      queryClient.invalidateQueries({ queryKey: streamDetailKey(connectionId, streamName) })
      queryClient.invalidateQueries({ queryKey: consumersListKey(connectionId, streamName) })
    },
    onError: (error: Error) => {
      toast.error(`Failed to pause consumer: ${getErrorMessage(error)}`)
    },
  })
}

export function useResumeConsumer(
  connectionId: string | undefined,
  streamName: string | undefined,
) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (consumerName: string) => {
      if (!connectionId || !streamName) throw new Error('No connection or stream')
      return api.resumeConsumer(connectionId, streamName, consumerName)
    },
    onSuccess: (_, consumerName) => {
      toast.success(`Consumer "${consumerName}" resumed`)
      queryClient.invalidateQueries({ queryKey: streamDetailKey(connectionId, streamName) })
      queryClient.invalidateQueries({ queryKey: consumersListKey(connectionId, streamName) })
    },
    onError: (error: Error) => {
      toast.error(`Failed to resume consumer: ${getErrorMessage(error)}`)
    },
  })
}
