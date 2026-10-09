import { useMemo } from 'react'
import type { ConnectionLabel } from '@/api/connections'
import { getStoredConnectionInfo } from '../activeConnectionStorage'
import { useActiveConnection } from './useActiveConnection'

export interface ConnectionPolicy {
  readOnly: boolean
  label: ConnectionLabel | null
  known: boolean
}

export function useConnectionPolicy(): ConnectionPolicy {
  const { connectionId, connection } = useActiveConnection()
  const stored = connection ? null : getStoredConnectionInfo()
  const storedPolicy = stored && stored.id === connectionId && stored.readOnly !== undefined ? stored : null
  const source = connection ?? storedPolicy
  const readOnly = source?.readOnly ?? true
  const label = source?.label ?? null
  const known = source !== null
  return useMemo(() => ({ readOnly, label, known }), [readOnly, label, known])
}
