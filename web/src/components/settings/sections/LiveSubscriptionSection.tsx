import type { LiveSubscriptionPolicyInput } from '@/contexts/settings'
import { Dropdown, WarningIcon } from '@/components/ui'
import { Section, Field, numberClass } from '../SettingsPrimitives'

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
            className={`${numberClass} ${value.maxDisplayRate != null && value.maxDisplayRate > 0 ? 'pr-9' : ''}`}
            placeholder="0"
            min={0}
            max={10000}
            step={1}
            value={value.maxDisplayRate ?? ''}
            onChange={(e) => onChange({ maxDisplayRate: e.target.value ? Number(e.target.value) : undefined })}
          />
          {value.maxDisplayRate != null && value.maxDisplayRate > 0 && (
            <span
              className="group absolute inset-y-0 right-2 flex items-center text-amber-500 cursor-help"
              tabIndex={0}
              aria-label="Throttle is display-only — messages over the limit are skipped in the live view but remain in the stream."
            >
              <WarningIcon className="w-4 h-4" />
              <span
                role="tooltip"
                className="pointer-events-none absolute right-full top-1/2 -translate-y-1/2 mr-2 w-64 rounded-md bg-surface-inverse px-3 py-2 text-xs leading-relaxed text-content-inverse shadow-lg opacity-0 invisible transition-opacity duration-75 group-hover:opacity-100 group-hover:visible group-focus-within:opacity-100 group-focus-within:visible z-50"
              >
                Throttle is display-only — messages over the limit are skipped in the live view but{' '}
                <span className="font-semibold">remain in the stream</span>.
              </span>
            </span>
          )}
        </div>
      </Field>
    </Section>
  )
}
