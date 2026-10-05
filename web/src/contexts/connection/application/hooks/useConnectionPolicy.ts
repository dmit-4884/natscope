import { useMemo } from 'react'
import type { ConnectionLabel } from '@/api/connections'
import { useActiveConnection } from './useActiveConnection'

export interface ConnectionPolicy {
  readOnly: boolean
  label: ConnectionLabel | null
}

export function useConnectionPolicy(): ConnectionPolicy {
  const { connection } = useActiveConnection()
  const readOnly = connection?.readOnly ?? false
  const label = connection?.label ?? null
  return useMemo(() => ({ readOnly, label }), [readOnly, label])
}
