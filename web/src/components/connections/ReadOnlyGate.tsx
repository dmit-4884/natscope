import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { EmptyState, LockClosedIcon } from '@/components/ui'
import { useActiveConnection, useConnectionPolicy } from '@/contexts/connection'

export function ReadOnlyGate({ children }: { children: ReactNode }) {
  const { readOnly } = useConnectionPolicy()
  const { connectionId } = useActiveConnection()
  if (!readOnly) return children
  return (
    <div className="flex-1 flex items-center justify-center">
      <EmptyState
        icon={<LockClosedIcon className="w-full h-full" />}
        title="This connection is read-only"
        description="Natscope refuses every write through it. Turn read-only off in the connection settings to publish or change anything."
        action={
          connectionId ? (
            <Link to={`/settings/connections/${connectionId}/edit`} className="text-sm font-medium text-accent hover:text-accent-text">
              Connection settings
            </Link>
          ) : undefined
        }
      />
    </div>
  )
}
