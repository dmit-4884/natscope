import type { DisplayPreferencesInput } from '@/contexts/settings'
import { Dropdown, Toggle } from '@/components/ui'
import { Section, Field } from '../SettingsPrimitives'

interface Props {
  value: DisplayPreferencesInput
  onChange: (patch: Partial<DisplayPreferencesInput>) => void
  isOpen: boolean
  onToggle: () => void
  onHelp: (key: string) => void
}

export function DisplayPreferencesSection({ value, onChange, isOpen, onToggle, onHelp }: Props) {
  return (
    <Section
      title="Display"
      description="Customize the visual appearance and layout"
      isOpen={isOpen}
      onToggle={onToggle}
    >
      <Field label="Density" description="Visual density of message lists" helpKey="display.density" onHelp={onHelp}>
        <Dropdown
          size="sm"
          value={value.density ?? 'comfortable'}
          onChange={(v) => onChange({ density: v })}
          options={[
            { value: 'comfortable', label: 'Comfortable' },
            { value: 'compact', label: 'Compact' },
          ]}
        />
      </Field>

      <Field label="Default view mode" description="Initial view when opening a stream" helpKey="display.defaultViewMode" onHelp={onHelp}>
        <Dropdown
          size="sm"
          value={value.defaultViewMode ?? 'history'}
          onChange={(v) => onChange({ defaultViewMode: v })}
          options={[
            { value: 'history', label: 'History' },
            { value: 'realtime', label: 'Realtime' },
          ]}
        />
      </Field>

      <Field label="Timestamp format" description="How timestamps are displayed" helpKey="display.timestampFormat" onHelp={onHelp}>
        <Dropdown
          size="sm"
          value={value.timestampFormat ?? 'relative'}
          onChange={(v) => onChange({ timestampFormat: v })}
          options={[
            { value: 'relative', label: 'Relative' },
            { value: 'absolute', label: 'Absolute' },
            { value: 'iso', label: 'ISO 8601' },
          ]}
        />
      </Field>

      <Field label="JSON indent size" description="Spaces for JSON formatting" helpKey="display.jsonIndentSize" onHelp={onHelp}>
        <Dropdown
          size="sm"
          value={String(value.jsonIndentSize ?? 2)}
          onChange={(v) => onChange({ jsonIndentSize: Number(v) })}
          options={[
            { value: '2', label: '2 spaces' },
            { value: '4', label: '4 spaces' },
          ]}
        />
      </Field>

      <Field label="Auto-scroll live" description="Scroll to new messages in live mode" helpKey="display.autoScrollLive" onHelp={onHelp}>
        <div className="flex items-center justify-end h-8">
          <Toggle
            checked={value.autoScrollLive ?? true}
            onChange={(next) => onChange({ autoScrollLive: next })}
            label="Auto-scroll live"
          />
        </div>
      </Field>
    </Section>
  )
}
