import type { LiveSubscriptionPolicyInput } from '@/contexts/settings'
import Tooltip from '@/components/common/Tooltip'
import { Dropdown, WarningIcon } from '@/components/ui'
import { Section, Field, numberClass } from '../SettingsPrimitives'

const DISPLAY_ONLY = 'Throttle is display-only — messages over the limit are skipped in the live view but remain in the stream.'

interface Props {
  value: LiveSubscriptionPolicyInput
  onChange: (patch: Partial<LiveSubscriptionPolicyInput>) => void
  isOpen: boolean
  onToggle: () => void
  onHelp: (key: string) => void
}

export function LiveSubscriptionSection({ value, onChange, isOpen, onToggle, onHelp }: Props) {
  return (
    <Section
      title="Live"
      description="Configure real-time message subscription behavior"
      isOpen={isOpen}
      onToggle={onToggle}
    >
      <Field label="Subscription mode" description="How to subscribe for live messages" helpKey="live.subscriptionMode" onHelp={onHelp}>
        <Dropdown
          size="sm"
          value={value.subscriptionMode ?? 'core_nats'}
          onChange={(v) => onChange({ subscriptionMode: v })}
          options={[
            { value: 'core_nats', label: 'Core NATS' },
            { value: 'jetstream_ordered', label: 'JetStream Ordered' },
          ]}
        />
      </Field>

      <Field label="Max display rate" description="Throttle messages to browser (msg/s, 0 = unlimited)" helpKey="live.maxDisplayRate" onHelp={onHelp}>
        <div className="relative">
          <input
            type="number"
            aria-label="Max display rate"
            className={`${numberClass} ${value.maxDisplayRate != null && value.maxDisplayRate > 0 ? 'pr-9' : ''}`}
            placeholder="0"
            min={0}
            max={10000}
            step={1}
            value={value.maxDisplayRate ?? ''}
            onChange={(e) => onChange({ maxDisplayRate: e.target.value ? Number(e.target.value) : undefined })}
          />
          {value.maxDisplayRate != null && value.maxDisplayRate > 0 && (
            <span className="absolute inset-y-0 right-2 flex items-center">
              <Tooltip content={DISPLAY_ONLY} position="left">
                <span className="text-amber-500 cursor-help" tabIndex={0} aria-label={DISPLAY_ONLY}>
                  <WarningIcon className="w-4 h-4" />
                </span>
              </Tooltip>
            </span>
          )}
        </div>
      </Field>
    </Section>
  )
}
