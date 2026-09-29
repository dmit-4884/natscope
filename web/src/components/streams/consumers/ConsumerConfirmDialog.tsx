import { DestructiveConfirm, Input } from '@/components/ui'
import { parseIntOr } from '@/utils/numbers'
import { DontAskAgainCheckbox } from '@/components/common/DontAskAgainCheckbox'
import type { ConsumerInfo } from '@/types/nats'

const MIN_PAUSE_MINUTES = 1
const MAX_PAUSE_MINUTES = 525_600

type ConsumerConfirmType = 'delete' | 'pause'

export interface ConsumerConfirmAction {
  type: ConsumerConfirmType
  consumer: ConsumerInfo
  pauseMinutes?: number
  /** Delete only: persist "skip this confirmation next time". */
  dontAskAgain?: boolean
}

interface Props {
  action: ConsumerConfirmAction
  onChange: (next: ConsumerConfirmAction) => void
  onCancel: () => void
  onConfirm: () => void
}

/**
 * Consumers are recoverable, so delete uses a simple confirm (+
 * don't-ask-again), not type-to-confirm.
 */
export function ConsumerConfirmDialog({ action, onChange, onCancel, onConfirm }: Props) {
  const isDelete = action.type === 'delete'
  const verb = isDelete ? 'Delete' : 'Pause'

  const description = isDelete ? (
    <span>
      This will delete the consumer <strong>{action.consumer.name}</strong>. It can be recreated if needed.
    </span>
  ) : (
    <span>
      This will pause the consumer <strong>{action.consumer.name}</strong>. Messages will not be delivered until resumed.
    </span>
  )

  const pauseMinutes = action.pauseMinutes ?? 5
  const pauseMinutesInvalid =
    !isDelete && (!Number.isFinite(pauseMinutes) || pauseMinutes < MIN_PAUSE_MINUTES || pauseMinutes > MAX_PAUSE_MINUTES)

  const extra = isDelete ? (
    <DontAskAgainCheckbox
      checked={action.dontAskAgain ?? false}
      onChange={(v) => onChange({ ...action, dontAskAgain: v })}
    />
  ) : (
    <div>
      <label htmlFor="consumer-pause-minutes" className="block text-sm font-medium text-gray-700 mb-1">
        Pause Duration (minutes)
      </label>
      <Input
        id="consumer-pause-minutes"
        type="number"
        min={MIN_PAUSE_MINUTES}
        max={MAX_PAUSE_MINUTES}
        error={pauseMinutesInvalid}
        errorMessage={pauseMinutesInvalid ? `Enter ${MIN_PAUSE_MINUTES}–${MAX_PAUSE_MINUTES} minutes` : undefined}
        value={pauseMinutes}
        onChange={(e) => onChange({ ...action, pauseMinutes: parseIntOr(e.target.value, 0) })}
      />
    </div>
  )

  return (
    <DestructiveConfirm
      isOpen
      title={`${verb} Consumer`}
      description={description}
      confirmLabel={verb}
      tone={isDelete ? 'danger' : 'warning'}
      extra={extra}
      confirmDisabled={pauseMinutesInvalid}
      onCancel={onCancel}
      onConfirm={onConfirm}
    />
  )
}
