import { useMemo } from 'react'
import type { ConnectionLabel } from '@/api/connections'
import { getStoredConnectionInfo } from '../activeConnectionStorage'
import { useActiveConnection } from './useActiveConnection'

export interface ConnectionPolicy {
  readOnly: boolean
  label: ConnectionLabel | null
}

export function useConnectionPolicy(): ConnectionPolicy {
  const { connectionId, connection } = useActiveConnection()
  const stored = connection ? null : getStoredConnectionInfo()
  const known = connection ?? (stored && stored.id === connectionId ? stored : null)
  const readOnly = known?.readOnly ?? true
  const label = known?.label ?? null
  return useMemo(() => ({ readOnly, label }), [readOnly, label])
}
