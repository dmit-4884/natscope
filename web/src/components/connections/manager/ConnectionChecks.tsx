import type { ReactNode } from 'react'
import { CheckIcon, CloseIcon, MinusIcon, WarningIcon } from '@/components/ui'
import type { ConnectionCheck, ConnectionCheckStatus, ConnectionCheckStep } from '@/api/connections'

const STEP_LABELS: Record<ConnectionCheckStep, string> = {
  dns: 'DNS',
  tcp: 'TCP',
  protocol: 'NATS protocol',
  tls: 'TLS',
  auth: 'Authentication',
  jetstream: 'JetStream',
}

const STATUS_STYLES: Record<ConnectionCheckStatus, { icon: ReactNode; label: string; className: string }> = {
  ok: { icon: <CheckIcon className="w-3.5 h-3.5" />, label: 'Passed', className: 'text-status-success-text' },
  warning: { icon: <WarningIcon className="w-3.5 h-3.5" />, label: 'Warning', className: 'text-status-warning-text' },
  failed: { icon: <CloseIcon className="w-3.5 h-3.5" />, label: 'Failed', className: 'text-status-error-text' },
  skipped: { icon: <MinusIcon className="w-3.5 h-3.5" />, label: 'Skipped', className: 'text-content-muted' },
}

export function ConnectionChecks({ checks }: { checks: ConnectionCheck[] }) {
  return (
    <ol className="space-y-2 text-xs" aria-label="Connection checks">
      {checks.map((c) => {
        const style = STATUS_STYLES[c.status]
        return (
          <li key={c.step} data-status={c.status} className="flex items-start gap-2">
            <span className={`mt-0.5 shrink-0 ${style.className}`} role="img" aria-label={style.label}>
              {style.icon}
            </span>
            <div className="min-w-0 flex-1">
              <div className="flex items-baseline justify-between gap-2">
                <span className={`font-medium ${c.status === 'skipped' ? 'text-content-muted' : 'text-content-primary'}`}>
                  {STEP_LABELS[c.step]}
                </span>
                {c.status !== 'skipped' && <span className="font-mono text-2xs text-content-muted">{c.durationMs} ms</span>}
              </div>
              <p className="break-words text-content-secondary">{c.detail}</p>
              {c.hint && <p className={`mt-0.5 break-words ${c.status === 'failed' ? 'text-status-error-text' : 'text-status-warning-text'}`}>{c.hint}</p>}
            </div>
          </li>
        )
      })}
    </ol>
  )
}
