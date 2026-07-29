import Tooltip from '@/components/common/Tooltip'
import { InfoIcon } from '@/components/ui'

function HelpIcon({ hint }: { hint: string }) {
  return (
    <Tooltip content={hint} position="bottom">
      <InfoIcon className="w-3.5 h-3.5 text-content-muted hover:text-content-tertiary" />
    </Tooltip>
  )
}

export function StatCard({
  label,
  value,
  highlight,
  hint,
}: {
  label: string
  value: number
  highlight?: 'warning' | 'error'
  hint?: string
}) {
  const valueColor =
    highlight === 'warning' ? 'text-orange-600' : highlight === 'error' ? 'text-status-error-text' : 'text-content-primary'
  return (
    <div className="bg-surface-primary rounded-lg border p-3">
      <div className="text-xs text-content-tertiary mb-1 flex items-center gap-1">
        {label}
        {hint && <HelpIcon hint={hint} />}
      </div>
      <div className={`text-xl font-semibold ${valueColor}`}>{value.toLocaleString()}</div>
    </div>
  )
}

export function ConfigRow({
  label,
  value,
  mono,
  hint,
}: {
  label: string
  value: string
  mono?: boolean
  hint?: string
}) {
  return (
    <div className="flex items-center gap-2 py-2 border-b border-gray-100 last:border-b-0">
      <span className="text-sm text-content-tertiary flex items-center gap-1 shrink-0">
        {label}
        {hint && <HelpIcon hint={hint} />}
      </span>
      <span className={`text-sm text-content-primary font-medium ${mono ? 'font-mono text-xs' : ''}`}>{value}</span>
    </div>
  )
}
