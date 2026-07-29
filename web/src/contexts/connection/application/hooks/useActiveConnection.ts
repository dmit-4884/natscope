import { useMemo, useSyncExternalStore } from 'react'
import { useConnectionEntities } from '../queries/useConnections'
import {
  getActiveConnectionId,
  subscribeActiveConnection,
} from '../activeConnectionStorage'
import type { Connection } from '../../domain/entities/Connection'

export interface UseActiveConnectionResult {
  /** Connection ID from localStorage */
  connectionId: string | null
  connection: Connection | null
  isLoading: boolean
  isConnected: boolean
}

export function useActiveConnection(): UseActiveConnectionResult {
  const connectionId = useSyncExternalStore(subscribeActiveConnection, getActiveConnectionId)
  const { connections, isLoading } = useConnectionEntities()

  const connection = useMemo(() => {
    if (!connectionId) return null
    return connections.find((c: Connection) => c.id === connectionId) ?? null
  }, [connectionId, connections])

  return {
    connectionId,
    connection,
    isLoading,
    isConnected: !!connection,
  }
}
