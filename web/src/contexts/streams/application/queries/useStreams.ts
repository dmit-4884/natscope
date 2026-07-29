import { useMemo, useCallback } from 'react'
import { Stream } from '../../domain/entities/Stream'
import { useStreams } from './useStreamList'

/**
 * All streams as domain entities. Shares the useStreams cache (one request),
 * adding domain mapping, sorting, search filtering, and aggregate stats.
 */
export function useStreamEntities(connectionId: string | null) {
  const { data, isLoading, isFetching, error, refetch } = useStreams(connectionId)

  const streams = useMemo(
    () => (data?.streams ?? []).map((s) => Stream.fromApi(s)),
    [data],
  )

  const sortedStreams = useMemo(
    () => [...streams].sort((a, b) => a.name.value.localeCompare(b.name.value)),
    [streams]
  )

  const getStreamByName = useCallback(
    (name: string): Stream | undefined => {
      return streams.find((s) => s.name.value === name)
    },
    [streams]
  )

  const filterStreams = useCallback(
    (search: string): Stream[] => {
      if (!search) return sortedStreams
      const lower = search.toLowerCase()
      return sortedStreams.filter(
        (s) =>
          s.name.value.toLowerCase().includes(lower) ||
          s.description.toLowerCase().includes(lower) ||
          s.subjects.some((sub) => sub.toLowerCase().includes(lower))
      )
    },
    [sortedStreams]
  )

  const stats = useMemo(() => {
    const totalMessages = streams.reduce((sum, s) => sum + s.messageCount, 0)
    const totalBytes = streams.reduce((sum, s) => sum + s.bytes, 0)
    const totalConsumers = streams.reduce((sum, s) => sum + s.consumerCount, 0)

    return {
      streamCount: streams.length,
      totalMessages,
      totalBytes,
      totalConsumers,
    }
  }, [streams])

  return {
    streams: sortedStreams,
    isLoading,
    isFetching,
    error: error as Error | null,
    refetch,
    getStreamByName,
    filterStreams,
    stats,
  }
}
