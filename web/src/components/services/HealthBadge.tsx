import Tooltip from '@/components/common/Tooltip'
import type { Health } from './serviceRates'

const HEALTH: Record<Exclude<Health, 'unknown'>, { label: string; dot: string; text: string; hint: string }> = {
  ok: {
    label: 'Healthy',
    dot: 'bg-status-success-border',
    text: 'text-status-success-text',
    hint: 'Fewer than 1% of the requests in the last refresh window failed',
  },
  degraded: {
    label: 'Degraded',
    dot: 'bg-status-warning-border',
    text: 'text-status-warning-text',
    hint: 'More than 1% of the requests in the last refresh window failed',
  },
  failing: {
    label: 'Failing',
    dot: 'bg-status-error-border',
    text: 'text-status-error-text',
    hint: 'More than 10% of the requests in the last refresh window failed',
  },
  idle: {
    label: 'Idle',
    dot: 'bg-content-muted',
    text: 'text-content-tertiary',
    hint: 'No requests in the last refresh window',
  },
}

export function HealthBadge({ health, compact = false }: { health: Health; compact?: boolean }) {
  if (health === 'unknown') return null
  const h = HEALTH[health]
  return (
    <Tooltip content={h.hint}>
      <span className={`inline-flex items-center gap-1.5 text-xs font-medium ${h.text}`} data-testid="service-health">
        <span aria-hidden="true" className={`w-2 h-2 rounded-full ${h.dot}`} />
        {compact ? <span className="sr-only">{h.label}</span> : h.label}
      </span>
    </Tooltip>
  )
}
