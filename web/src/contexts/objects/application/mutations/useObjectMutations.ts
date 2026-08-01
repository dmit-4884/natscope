import { useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from '@/utils/toast'
import * as api from '@/api/management'
import { getErrorMessage } from '@/api/errors'
import type { ObjectBucketConfig } from '@/types/management'
import { objectKeys } from '../queries/objectKeys'

export function useCreateObjectBucket(connectionId: string | undefined) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (config: ObjectBucketConfig) => {
      if (!connectionId) throw new Error('No connection')
      return api.createObjectBucket(connectionId, config)
    },
    onSuccess: (bucket) => {
      toast.success(`Object bucket "${bucket.bucket}" created`)
      queryClient.invalidateQueries({ queryKey: objectKeys.buckets(connectionId) })
    },
    onError: (error: Error) => {
      toast.error(`Failed to create object bucket: ${getErrorMessage(error)}`)
    },
  })
}

export function useDeleteObjectBucket(connectionId: string | undefined) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (bucket: string) => {
      if (!connectionId) throw new Error('No connection')
      return api.deleteObjectBucket(connectionId, bucket)
    },
    onSuccess: (_, bucket) => {
      toast.success(`Object bucket "${bucket}" deleted`)
      queryClient.removeQueries({ queryKey: objectKeys.bucket(connectionId, bucket) })
      queryClient.removeQueries({ queryKey: objectKeys.list(connectionId, bucket) })
      queryClient.invalidateQueries({ queryKey: objectKeys.buckets(connectionId) })
    },
    onError: (error: Error) => {
      toast.error(`Failed to delete object bucket: ${getErrorMessage(error)}`)
    },
  })
}

export function useSealObjectBucket(connectionId: string | undefined) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (bucket: string) => {
      if (!connectionId) throw new Error('No connection')
      return api.sealObjectBucket(connectionId, bucket)
    },
    onSuccess: (_, bucket) => {
      toast.success(`Object bucket "${bucket}" sealed`)
      queryClient.invalidateQueries({ queryKey: objectKeys.buckets(connectionId) })
      queryClient.invalidateQueries({ queryKey: objectKeys.bucket(connectionId, bucket) })
    },
    onError: (error: Error) => {
      toast.error(`Failed to seal object bucket: ${getErrorMessage(error)}`)
    },
  })
}

export function usePutObject(connectionId: string | undefined, bucket: string | undefined) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      name,
      data,
      options,
    }: {
      name: string
      data: string | Uint8Array
      options?: { description?: string; metadata?: Record<string, string> }
    }) => {
      if (!connectionId || !bucket) throw new Error('No connection or bucket')
      return api.putObject(connectionId, bucket, name, data, options)
    },
    onSuccess: (info) => {
      toast.success(`Object "${info.name}" uploaded`)
      queryClient.invalidateQueries({ queryKey: objectKeys.list(connectionId, bucket) })
      queryClient.invalidateQueries({ queryKey: objectKeys.bucket(connectionId, bucket) })
      queryClient.invalidateQueries({ queryKey: objectKeys.object(connectionId, bucket, info.name) })
      queryClient.invalidateQueries({ queryKey: objectKeys.buckets(connectionId) })
    },
    onError: (error: Error) => {
      toast.error(`Failed to upload object: ${getErrorMessage(error)}`)
    },
  })
}

export function useDeleteObject(connectionId: string | undefined, bucket: string | undefined) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (name: string) => {
      if (!connectionId || !bucket) throw new Error('No connection or bucket')
      return api.deleteObject(connectionId, bucket, name)
    },
    onSuccess: (_, name) => {
      toast.success(`Object "${name}" deleted`)
      queryClient.removeQueries({ queryKey: objectKeys.object(connectionId, bucket, name) })
      queryClient.invalidateQueries({ queryKey: objectKeys.list(connectionId, bucket) })
      queryClient.invalidateQueries({ queryKey: objectKeys.bucket(connectionId, bucket) })
    },
    onError: (error: Error) => {
      toast.error(`Failed to delete object: ${getErrorMessage(error)}`)
    },
  })
}
