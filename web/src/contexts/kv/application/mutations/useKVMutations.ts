import { useMutation, useQueryClient, type QueryClient } from '@tanstack/react-query'
import { toast } from '@/utils/toast'
import * as api from '@/api/management'
import { getErrorMessage } from '@/api/errors'
import { CONNECTION_QUERY_PREFIX } from '@/hooks/useConnectionQuery'
import type { KVBucketConfig } from '@/types/management'
import { kvKeys } from '../queries/kvKeys'

/**
 * KV buckets live in three caches: kv.buckets (list), kv.bucket (metadata),
 * and streams (sidebar filters KV_*). A mutation must invalidate all three or
 * the UI lies until Refresh.
 */
function invalidateAllKVViews(queryClient: QueryClient, connectionId: string | undefined, bucket?: string) {
  queryClient.invalidateQueries({ queryKey: kvKeys.buckets(connectionId) })
  if (bucket) {
    queryClient.invalidateQueries({ queryKey: kvKeys.bucket(connectionId, bucket) })
  }
  // Sidebar KV list derives from the streams query — also refresh.
  queryClient.invalidateQueries({
    queryKey: [CONNECTION_QUERY_PREFIX, connectionId ?? null, 'streams'],
  })
}

export function useCreateKVBucket(connectionId: string | undefined) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (config: KVBucketConfig) => {
      if (!connectionId) throw new Error('No connection')
      return api.createKVBucket(connectionId, config)
    },
    onSuccess: (bucket) => {
      toast.success(`KV bucket "${bucket.bucket}" created`)
      invalidateAllKVViews(queryClient, connectionId, bucket.bucket)
    },
    onError: (error: Error) => {
      toast.error(`Failed to create KV bucket: ${getErrorMessage(error)}`)
    },
  })
}

export function useDeleteKVBucket(connectionId: string | undefined) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (bucket: string) => {
      if (!connectionId) throw new Error('No connection')
      return api.deleteKVBucket(connectionId, bucket)
    },
    onSuccess: (_, bucket) => {
      toast.success(`KV bucket "${bucket}" deleted`)
      queryClient.removeQueries({ queryKey: kvKeys.bucket(connectionId, bucket) })
      queryClient.removeQueries({ queryKey: kvKeys.keys(connectionId, bucket) })
      invalidateAllKVViews(queryClient, connectionId)
    },
    onError: (error: Error) => {
      toast.error(`Failed to delete KV bucket: ${getErrorMessage(error)}`)
    },
  })
}

export function usePutKVKey(connectionId: string | undefined, bucket: string | undefined) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      key,
      value,
      expectedRevision,
    }: {
      key: string
      value: string | Uint8Array
      expectedRevision?: number
    }) => {
      if (!connectionId || !bucket) throw new Error('No connection or bucket')
      return api.putKVKey(connectionId, bucket, key, value, expectedRevision)
    },
    onSuccess: (result, { key }) => {
      toast.success(`Key "${key}" saved (revision ${result.revision})`)
      queryClient.invalidateQueries({ queryKey: kvKeys.keys(connectionId, bucket) })
      queryClient.invalidateQueries({ queryKey: kvKeys.key(connectionId, bucket, key) })
      queryClient.invalidateQueries({ queryKey: kvKeys.history(connectionId, bucket, key) })
      queryClient.invalidateQueries({ queryKey: kvKeys.bucket(connectionId, bucket) })
      // Header "{n} keys | {bytes}" reads from the bucket list — refresh too.
      queryClient.invalidateQueries({ queryKey: kvKeys.buckets(connectionId) })
    },
    onError: (error: Error) => {
      toast.error(`Failed to save key: ${getErrorMessage(error)}`)
    },
  })
}

export function useDeleteKVKey(connectionId: string | undefined, bucket: string | undefined) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (key: string) => {
      if (!connectionId || !bucket) throw new Error('No connection or bucket')
      return api.deleteKVKey(connectionId, bucket, key)
    },
    onSuccess: (_, key) => {
      toast.success(`Key "${key}" deleted`)
      queryClient.removeQueries({ queryKey: kvKeys.key(connectionId, bucket, key) })
      queryClient.invalidateQueries({ queryKey: kvKeys.keys(connectionId, bucket) })
      queryClient.invalidateQueries({ queryKey: kvKeys.history(connectionId, bucket, key) })
      queryClient.invalidateQueries({ queryKey: kvKeys.bucket(connectionId, bucket) })
      queryClient.invalidateQueries({ queryKey: kvKeys.buckets(connectionId) })
    },
    onError: (error: Error) => {
      toast.error(`Failed to delete key: ${getErrorMessage(error)}`)
    },
  })
}

export function usePurgeKVKey(connectionId: string | undefined, bucket: string | undefined) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (key: string) => {
      if (!connectionId || !bucket) throw new Error('No connection or bucket')
      return api.purgeKVKey(connectionId, bucket, key)
    },
    onSuccess: (_, key) => {
      toast.success(`Key "${key}" purged`)
      queryClient.removeQueries({ queryKey: kvKeys.key(connectionId, bucket, key) })
      queryClient.invalidateQueries({ queryKey: kvKeys.keys(connectionId, bucket) })
      queryClient.invalidateQueries({ queryKey: kvKeys.history(connectionId, bucket, key) })
      queryClient.invalidateQueries({ queryKey: kvKeys.bucket(connectionId, bucket) })
      queryClient.invalidateQueries({ queryKey: kvKeys.buckets(connectionId) })
    },
    onError: (error: Error) => {
      toast.error(`Failed to purge key: ${getErrorMessage(error)}`)
    },
  })
}
