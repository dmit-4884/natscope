import * as api from '@/api/management'
import { useConnectionQuery } from '@/hooks/useConnectionQuery'

export function useObjectBuckets(connectionId: string | undefined) {
  return useConnectionQuery({
    key: ['objects', 'buckets'],
    connectionId: connectionId ?? null,
    fetcher: (signal) => api.listObjectBuckets(connectionId!, signal),
  })
}

export function useObjects(connectionId: string | undefined, bucket: string | undefined) {
  return useConnectionQuery({
    key: ['objects', 'list', bucket],
    connectionId: connectionId ?? null,
    enabled: !!bucket,
    fetcher: (signal) => api.listObjects(connectionId!, bucket!, signal),
  })
}

export function useObject(
  connectionId: string | undefined,
  bucket: string | undefined,
  name: string | undefined,
) {
  return useConnectionQuery({
    key: ['objects', 'object', bucket, name],
    connectionId: connectionId ?? null,
    enabled: !!bucket && !!name,
    fetcher: (signal) => api.getObject(connectionId!, bucket!, name!, signal),
  })
}
