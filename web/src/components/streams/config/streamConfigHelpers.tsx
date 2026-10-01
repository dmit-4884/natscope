import type { ReactNode } from 'react'
import Tooltip from '@/components/common/Tooltip'
import { Badge, InfoIcon } from '@/components/ui'
import type { ReplicaInfo } from '@/types/nats'
import { replicaState } from './streamConfigUtils'

/** Small stat card rendered in the stream config header. */
export function StatCard({ label, value, hint }: { label: string; value: string; hint?: string }) {
  return (
    <div className="bg-surface-primary rounded-lg border p-3">
      <div className="text-xs text-content-tertiary mb-1 flex items-center gap-1">
        {label}
        {hint && <HelpIcon hint={hint} />}
      </div>
      <div className="text-xl font-semibold text-content-primary">{value}</div>
    </div>
  )
}

/** Two-column label/value row. */
export function ConfigRow({ label, value, hint }: { label: string; value: string; hint?: string }) {
  return (
    <div className="flex items-center gap-2 py-2 border-b border-gray-100 last:border-b-0">
      <span className="text-sm text-content-tertiary flex items-center gap-1 shrink-0">
        {label}
        {hint && <HelpIcon hint={hint} />}
      </span>
      <span className="text-sm text-content-primary font-medium min-w-0 break-all" title={value}>
        {value}
      </span>
    </div>
  )
}

/** Small flag badge. */
export function Flag({ label, color, hint }: { label: string; color: 'red' | 'green'; hint?: string }) {
  const colors = {
    red: 'bg-status-error-bg text-red-700',
    green: 'bg-status-success-bg text-green-700',
  }
  const badge = <span className={`px-2 py-1 text-xs rounded ${colors[color]}`}>{label}</span>
  if (hint) {
    return (
      <Tooltip content={hint} position="bottom">
        {badge}
      </Tooltip>
    )
  }
  return badge
}

export function ReplicaBadge({ replica }: { replica: ReplicaInfo }) {
  const state = replicaState(replica)
  return (
    <Tooltip content={state.detail} position="bottom">
      <Badge variant={state.variant} className="gap-1.5" data-testid={`replica-${replica.name}`}>
        <span className="w-1.5 h-1.5 rounded-full bg-current shrink-0" aria-hidden="true" />
        {replica.name}
        {state.label && <span className="font-normal">{state.label}</span>}
      </Badge>
    </Tooltip>
  )
}

function HelpIcon({ hint }: { hint: string }): ReactNode {
  return (
    <Tooltip content={hint} position="bottom">
      <InfoIcon className="w-3.5 h-3.5 text-content-muted hover:text-content-tertiary" />
    </Tooltip>
  )
}

// `formatConfigValue`/`hasMetadata` live in ./streamConfigUtils so this file
// stays pure-component (HMR fast-refresh).
