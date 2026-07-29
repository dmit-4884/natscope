import type { MessageFetchPolicyInput } from '@/contexts/settings'
import { Dropdown, Input } from '@/components/ui'
import { Section, Field } from '../SettingsPrimitives'

interface Props {
  value: MessageFetchPolicyInput
  onChange: (patch: Partial<MessageFetchPolicyInput>) => void
  isOpen: boolean
  onToggle: () => void
  onHelp: (key: string) => void
}

export function MessageFetchPolicySection({ value, onChange, isOpen, onToggle, onHelp }: Props) {
  return (
    <Section
      title="Messages"
      description="Configure how messages are fetched and displayed in history mode"
      isOpen={isOpen}
      onToggle={onToggle}
    >
      <Field label="Fetch method" description="How messages are retrieved from streams" helpKey="messages.fetchMethod" onHelp={onHelp}>
        <Dropdown
          size="sm"
          value={value.fetchMethod ?? 'consumer'}
          onChange={(v) => onChange({ fetchMethod: v })}
          options={[
            { value: 'direct', label: 'Direct' },
            { value: 'consumer', label: 'Consumer' },
          ]}
        />
      </Field>

      <Field label="Default page size" description="Number of messages per page" helpKey="messages.defaultPageSize" onHelp={onHelp}>
        <Dropdown
          size="sm"
          value={String(value.defaultPageSize ?? 50)}
          onChange={(v) => onChange({ defaultPageSize: Number(v) })}
          options={[
            { value: '25', label: '25' },
            { value: '50', label: '50' },
            { value: '100', label: '100' },
            { value: '250', label: '250' },
          ]}
        />
      </Field>

      <Field label="Default direction" description="Initial sort direction for history" helpKey="messages.defaultDirection" onHelp={onHelp}>
        <Dropdown
          size="sm"
          value={value.defaultDirection ?? 'backward'}
          onChange={(v) => onChange({ defaultDirection: v })}
          options={[
            { value: 'backward', label: 'Backward' },
            { value: 'forward', label: 'Forward' },
          ]}
        />
      </Field>

      <Field
        label="Preview payload cap (KB)"
        description="Max payload bytes shown per message in list & live. 0 = unlimited. Detail view always loads the full payload."
        helpKey="messages.maxPayloadBytesInList"
        onHelp={onHelp}
      >
        <Input
          type="number"
          size="sm"
          min={0}
          max={64}
          step={1}
          // VO holds bytes; UI talks KB so the number stays readable.
          value={String(Math.round((value.maxPayloadBytesInList ?? 64 * 1024) / 1024))}
          onChange={(e) => {
            if (e.currentTarget.value === '') return
            const kb = Number(e.currentTarget.value)
            if (!Number.isFinite(kb)) return
            onChange({ maxPayloadBytesInList: Math.min(64, Math.max(0, Math.round(kb))) * 1024 })
          }}
        />
      </Field>
    </Section>
  )
}
