import type { ReactNode } from 'react'
import Tooltip from '@/components/common/Tooltip'
import { InfoIcon } from '@/components/ui'

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
      <span className="text-sm text-content-primary font-medium">{value}</span>
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

function HelpIcon({ hint }: { hint: string }): ReactNode {
  return (
    <Tooltip content={hint} position="bottom">
      <InfoIcon className="w-3.5 h-3.5 text-content-muted hover:text-content-tertiary" />
    </Tooltip>
  )
}

// `formatConfigValue`/`hasMetadata` live in ./streamConfigUtils so this file
// stays pure-component (HMR fast-refresh).
