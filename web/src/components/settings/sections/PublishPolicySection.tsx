import type { PublishPolicyInput } from '@/contexts/settings'
import { Section, Field, numberClass } from '../SettingsPrimitives'

interface Props {
  value: PublishPolicyInput
  onChange: (patch: Partial<PublishPolicyInput>) => void
  isOpen: boolean
  onToggle: () => void
  onHelp: (key: string) => void
}

export function PublishPolicySection({ value, onChange, isOpen, onToggle, onHelp }: Props) {
  return (
    <Section
      title="Publish"
      description="Configure message publishing behavior"
      isOpen={isOpen}
      onToggle={onToggle}
    >
      <Field label="Publish timeout" description="JetStream ack timeout (seconds)" helpKey="publish.publishTimeoutSec" onHelp={onHelp}>
        <input
          type="number"
          className={numberClass}
          placeholder="10"
          min={1}
          max={60}
          step={1}
          value={value.publishTimeoutSec ?? ''}
          onChange={(e) => onChange({ publishTimeoutSec: e.target.value ? Number(e.target.value) : undefined })}
        />
      </Field>
    </Section>
  )
}
