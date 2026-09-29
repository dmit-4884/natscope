import { DestructiveConfirm, Input } from '@/components/ui'
import { parseIntOr } from '@/utils/numbers'
import { DontAskAgainCheckbox } from '@/components/common/DontAskAgainCheckbox'
import type { ConsumerInfo } from '@/types/nats'
import { parseResetSequence } from './consumerUtils'

const MIN_PAUSE_MINUTES = 1
const MAX_PAUSE_MINUTES = 525_600

const RESETTABLE_TO_SEQUENCE = new Set(['all', 'by_start_sequence', 'by_start_time'])

type ConsumerConfirmType = 'delete' | 'pause' | 'reset'

export interface ConsumerConfirmAction {
  type: ConsumerConfirmType
  consumer: ConsumerInfo
  pauseMinutes?: number
  /** Reset only: raw sequence input; empty keeps the ack floor. */
  resetSequence?: string
  /** Delete only: persist "skip this confirmation next time". */
  dontAskAgain?: boolean
}

interface Props {
  action: ConsumerConfirmAction
  onChange: (next: ConsumerConfirmAction) => void
  onCancel: () => void
  onConfirm: () => void
}

const VERBS: Record<ConsumerConfirmType, string> = { delete: 'Delete', pause: 'Pause', reset: 'Reset' }

/**
 * Consumers are recoverable, so delete uses a simple confirm (+
 * don't-ask-again), not type-to-confirm.
 */
export function ConsumerConfirmDialog({ action, onChange, onCancel, onConfirm }: Props) {
  const verb = VERBS[action.type]
  const name = <strong>{action.consumer.name}</strong>

  if (action.type === 'delete') {
    return (
      <DestructiveConfirm
        isOpen
        title={`${verb} Consumer`}
        description={<span>This will delete the consumer {name}. It can be recreated if needed.</span>}
        confirmLabel={verb}
        tone="danger"
        extra={
          <DontAskAgainCheckbox
            checked={action.dontAskAgain ?? false}
            onChange={(v) => onChange({ ...action, dontAskAgain: v })}
          />
        }
        onCancel={onCancel}
        onConfirm={onConfirm}
      />
    )
  }

  if (action.type === 'reset') {
    const canSeek = RESETTABLE_TO_SEQUENCE.has(action.consumer.config?.deliver_policy ?? 'all')
    const sequenceInvalid = parseResetSequence(action.resetSequence) === null
    return (
      <DestructiveConfirm
        isOpen
        title={`${verb} Consumer`}
        description={
          <span>
            This will reset the delivery state of the consumer {name}: pending and redelivery state are cleared and
            unacknowledged messages are delivered again.
          </span>
        }
        confirmLabel={verb}
        tone="warning"
        extra={
          <div>
            <label htmlFor="consumer-reset-sequence" className="block text-sm font-medium text-gray-700 mb-1">
              Start from stream sequence (optional)
            </label>
            <Input
              id="consumer-reset-sequence"
              inputMode="numeric"
              placeholder="Keep the ack floor"
              disabled={!canSeek}
              error={sequenceInvalid}
              errorMessage={sequenceInvalid ? 'Enter a positive stream sequence, or leave empty' : undefined}
              value={action.resetSequence ?? ''}
              onChange={(e) => onChange({ ...action, resetSequence: e.target.value })}
            />
            {!canSeek && (
              <p className="text-xs text-content-tertiary mt-1">
                Only consumers delivering all messages, or from a start sequence or time, can reset to a sequence.
              </p>
            )}
          </div>
        }
        confirmDisabled={canSeek && sequenceInvalid}
        onCancel={onCancel}
        onConfirm={onConfirm}
      />
    )
  }

  const pauseMinutes = action.pauseMinutes ?? 5
  const pauseMinutesInvalid =
    !Number.isFinite(pauseMinutes) || pauseMinutes < MIN_PAUSE_MINUTES || pauseMinutes > MAX_PAUSE_MINUTES

  return (
    <DestructiveConfirm
      isOpen
      title={`${verb} Consumer`}
      description={
        <span>This will pause the consumer {name}. Messages will not be delivered until resumed.</span>
      }
      confirmLabel={verb}
      tone="warning"
      extra={
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
      }
      confirmDisabled={pauseMinutesInvalid}
      onCancel={onCancel}
      onConfirm={onConfirm}
    />
  )
}
