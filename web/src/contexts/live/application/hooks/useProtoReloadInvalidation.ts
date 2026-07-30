import { useEffect } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { plural } from '@/utils/plural'
import { toast } from '@/utils/toast'
import { CONNECTION_QUERY_PREFIX } from '@/hooks/useConnectionQuery'

import type { LiveStreamClient } from '../LiveStreamClient'

/**
 * On proto-reload events, invalidate react-query caches that depend on the
 * proto registry (proto, mappings, protoSources, messages) and toast so the
 * operator knows decoded views changed.
 */
export function useProtoReloadInvalidation(client: LiveStreamClient | null) {
  const queryClient = useQueryClient()

  useEffect(() => {
    if (!client) return

    const previous = client.onProtoReload
    client.onProtoReload = (payload) => {
      previous?.(payload)

      queryClient.invalidateQueries({ queryKey: ['proto'] })
      queryClient.invalidateQueries({ queryKey: ['mappings'] })
      queryClient.invalidateQueries({ queryKey: ['protoSources'] })
      // Messages live under [CONNECTION_QUERY_PREFIX, connId, 'messages', ...].
      // Invalidate by predicate because we don't know the connection ids here.
      queryClient.invalidateQueries({
        predicate: (q) => {
          const k = q.queryKey
          return Array.isArray(k) && k[0] === CONNECTION_QUERY_PREFIX && k[2] === 'messages'
        },
      })

      toast.info(`Proto schemas reloaded — ${plural(payload.messages_count, 'message type')} available`)
    }

    return () => {
      client.onProtoReload = previous
    }
  }, [client, queryClient])
}
