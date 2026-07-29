import { Input } from '@/components/ui'
import { parseIntOr } from '@/utils/numbers'
import type { ConsumerLimitsConfig } from '@/types/management'

interface Props {
  value: ConsumerLimitsConfig | undefined
  onChange: (next: ConsumerLimitsConfig | undefined) => void
}

// Default consumer limits inherited by all consumers; both fields = 0 collapses
// to undefined.
export function ConsumerLimitsEditor({ value, onChange }: Props) {
  const v = value ?? { inactive_threshold: 0, max_ack_pending: 0 }

  const update = (patch: Partial<ConsumerLimitsConfig>) => {
    const next = { ...v, ...patch }
    if (!next.inactive_threshold && !next.max_ack_pending) {
      onChange(undefined)
      return
    }
    onChange(next)
  }

  return (
    <div className="grid grid-cols-2 gap-3">
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">Inactive Threshold (ns)</label>
        <Input
          type="number"
          value={v.inactive_threshold ?? 0}
          onChange={(e) => update({ inactive_threshold: parseIntOr(e.target.value, 0) })}
        />
        <p className="text-xs text-content-tertiary mt-1">Default cleanup window for ephemeral consumers.</p>
      </div>
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">Max Ack Pending</label>
        <Input
          type="number"
          value={v.max_ack_pending ?? 0}
          onChange={(e) => update({ max_ack_pending: parseIntOr(e.target.value, 0) })}
        />
        <p className="text-xs text-content-tertiary mt-1">Default cap on outstanding acks per consumer.</p>
      </div>
    </div>
  )
}
