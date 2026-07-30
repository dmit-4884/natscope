import { useEffect, useRef } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { streamKeys } from '@/contexts/streams'
import type { ViewMode } from './messageListUtils'

/**
 * History is cached with a long staleTime and refetchOnMount: false, so
 * returning from live tail would otherwise render whatever was fetched before
 * the tail started. Invalidate the stream's messages and detail (the stats
 * header source) on the realtime -> history transition.
 */
export function useHistoryRefreshOnModeChange(
  mode: ViewMode,
  connectionId: string | null,
  streamName: string | null,
) {
  const queryClient = useQueryClient()
  const previousModeRef = useRef<ViewMode>(mode)

  useEffect(() => {
    const previousMode = previousModeRef.current
    previousModeRef.current = mode

    if (previousMode === mode || mode !== 'history') return
    if (!connectionId || !streamName) return

    queryClient.invalidateQueries({ queryKey: streamKeys.messages(connectionId, streamName) })
    queryClient.invalidateQueries({ queryKey: streamKeys.detail(connectionId, streamName) })
  }, [mode, connectionId, streamName, queryClient])
}
