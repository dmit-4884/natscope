import type { ReactNode } from 'react'
import type { BehaviorPolicyInput } from '@/contexts/settings'
import { Toggle } from '@/components/ui'
import { Section } from '../SettingsPrimitives'

interface Props {
  value: BehaviorPolicyInput
  onChange: (patch: Partial<BehaviorPolicyInput>) => void
  isOpen: boolean
  onToggle: () => void
}

/** A label/description row with a right-aligned toggle switch. */
function ToggleRow({
  label,
  description,
  checked,
  onChange,
  testId,
}: {
  label: string
  description: ReactNode
  checked: boolean
  onChange: (next: boolean) => void
  testId?: string
}) {
  return (
    <div className="flex items-start justify-between gap-4">
      <div className="flex-1 min-w-0">
        <label className="text-sm font-medium text-gray-700">{label}</label>
        <p className="text-xs text-content-tertiary mt-0.5">{description}</p>
      </div>
      <div className="flex-shrink-0 flex items-center h-8">
        <Toggle checked={checked} onChange={onChange} label={label} testId={testId} />
      </div>
    </div>
  )
}

/**
 * Behavior settings — confirm-toggles and destructive-op defaults.
 *
 * Each "Confirm before …" toggle defaults ON; turning it off opts the user
 * out of that prompt. Only recoverable-scope operations are listed here;
 * delete/purge stream and delete bucket always keep type-to-confirm and have
 * no toggle by design.
 */
export function BehaviorSettingsSection({ value, onChange, isOpen, onToggle }: Props) {
  const confirmAll = () =>
    onChange({
      confirmDeleteConsumer: true,
      confirmDeleteMessage: true,
      confirmDeleteKvKey: true,
      confirmDeleteObject: true,
      confirmPurgeKvHistory: true,
    })

  return (
    <Section
      title="Behavior"
      description="Confirmation prompts and defaults for destructive operations"
      isOpen={isOpen}
      onToggle={onToggle}
    >
      <div data-testid="behavior-confirmations" className="space-y-4">
        <div className="flex items-center justify-between">
          <h4 className="text-xs font-semibold uppercase tracking-wide text-content-muted">
            Confirmations
          </h4>
          <button
            type="button"
            onClick={confirmAll}
            data-testid="reset-confirmations"
            className="text-xs font-medium text-accent hover:text-accent-text hover:underline"
          >
            Reset all confirmations
          </button>
        </div>

        <ToggleRow
          label="Confirm before deleting a consumer"
          description="Ask before removing a consumer from a stream."
          checked={value.confirmDeleteConsumer ?? true}
          onChange={(v) => onChange({ confirmDeleteConsumer: v })}
          testId="confirm-delete-consumer"
        />
        <ToggleRow
          label="Confirm before deleting a message"
          description="Ask before deleting a single message from a stream."
          checked={value.confirmDeleteMessage ?? true}
          onChange={(v) => onChange({ confirmDeleteMessage: v })}
          testId="confirm-delete-message"
        />
        <ToggleRow
          label="Confirm before deleting a KV key"
          description="Ask before deleting a key from a KV bucket."
          checked={value.confirmDeleteKvKey ?? true}
          onChange={(v) => onChange({ confirmDeleteKvKey: v })}
          testId="confirm-delete-kv-key"
        />
        <ToggleRow
          label="Confirm before deleting an object"
          description="Ask before deleting an object from an object store."
          checked={value.confirmDeleteObject ?? true}
          onChange={(v) => onChange({ confirmDeleteObject: v })}
          testId="confirm-delete-object"
        />
        <ToggleRow
          label="Confirm before purging KV key history"
          description="Ask before purging the revision history of a KV key."
          checked={value.confirmPurgeKvHistory ?? true}
          onChange={(v) => onChange({ confirmPurgeKvHistory: v })}
          testId="confirm-purge-kv-history"
        />
      </div>

      <div className="border-t border-gray-100 pt-4 space-y-4">
        <h4 className="text-xs font-semibold uppercase tracking-wide text-content-muted">Defaults</h4>
        <ToggleRow
          label="Secure delete by default"
          description="Pre-select the secure-erase (overwrite) option in the message-delete dialog."
          checked={value.secureDeleteDefault ?? false}
          onChange={(v) => onChange({ secureDeleteDefault: v })}
          testId="secure-delete-default"
        />
      </div>
    </Section>
  )
}
