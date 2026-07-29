import { useQuery } from '@tanstack/react-query'
import { useMemo, useCallback } from 'react'
import { getConnections, type SavedConnection } from '@/api/connections'
import { Connection } from '../../domain/entities/Connection'
import { connectionKeys } from './connectionKeys'

const toConnectionEntities = (data: SavedConnection[]): Connection[] =>
  data.map((c) => Connection.fromApi(c))

/** Fetch all connections as domain entities. */
export function useConnectionEntities() {
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: connectionKeys.all,
    queryFn: getConnections,
    select: toConnectionEntities,
  })

  const connections = useMemo(() => data ?? [], [data])

  const sortedConnections = useMemo(
    () => [...connections].sort((a, b) => a.name.localeCompare(b.name)),
    [connections]
  )

  const getConnectionById = useCallback(
    (id: string): Connection | undefined => {
      return connections.find((c) => c.id === id)
    },
    [connections]
  )

  const filterConnections = useCallback(
    (search: string): Connection[] => {
      if (!search) return sortedConnections
      const lower = search.toLowerCase()
      return sortedConnections.filter(
        (c) =>
          c.name.toLowerCase().includes(lower) ||
          (c.primaryUrl?.host ?? '').toLowerCase().includes(lower)
      )
    },
    [sortedConnections]
  )

  return {
    connections: sortedConnections,
    isLoading,
    error: error as Error | null,
    refetch,
    getConnectionById,
    filterConnections,
  }
}
