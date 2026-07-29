import { useState } from 'react'
import { parseIntOr } from '@/utils/numbers'
import { Input, ImmutableField, Dropdown } from '@/components/ui'
import type { ObjectBucketConfig } from '@/types/management'
import { SectionPanel } from '@/components/common/forms/SectionPanel'

interface ObjectBucketFormFieldsProps {
  value: ObjectBucketConfig
  onChange: (value: ObjectBucketConfig) => void
  isEditMode: boolean
}

export function ObjectBucketFormFields({
  value,
  onChange,
  isEditMode,
}: ObjectBucketFormFieldsProps) {
  const [openSections, setOpenSections] = useState<Record<string, boolean>>({
    basic: true,
    limits: false,
    storage: false,
  })

  const toggleSection = (section: string) => {
    setOpenSections((prev) => ({ ...prev, [section]: !prev[section] }))
  }

  const updateField = <K extends keyof ObjectBucketConfig>(
    field: K,
    fieldValue: ObjectBucketConfig[K]
  ) => {
    onChange({ ...value, [field]: fieldValue })
  }

  return (
    <div className="space-y-4">
      {/* Basic Section */}
      <SectionPanel label="Basic Configuration" isOpen={openSections.basic} onToggle={() => toggleSection('basic')}>
        <div className="space-y-4">
          {/* Bucket Name */}
          <ImmutableField label="Bucket Name" isImmutable={isEditMode} helpText="Unique identifier for the object store bucket">
            <Input
              value={value.bucket || ''}
              onChange={(e) => updateField('bucket', e.target.value)}
              placeholder="my-object-bucket"
              disabled={isEditMode}
            />
          </ImmutableField>

          {/* Description */}
          <div>
            <label htmlFor="obj-description" className="block text-sm font-medium text-gray-700 mb-1">Description</label>
            <Input
              id="obj-description"
              value={value.description || ''}
              onChange={(e) => updateField('description', e.target.value)}
              placeholder="Optional description"
            />
          </div>
        </div>
      </SectionPanel>

      {/* Limits Section */}
      <SectionPanel label="Limits" isOpen={openSections.limits} onToggle={() => toggleSection('limits')}>
        <div className="grid grid-cols-2 gap-4">
          {/* Max Bytes */}
          <div>
            <label htmlFor="obj-max-bytes" className="block text-sm font-medium text-gray-700 mb-1">Max Bytes</label>
            <Input
              id="obj-max-bytes"
              type="number"
              value={value.max_bytes ?? -1}
              onChange={(e) => updateField('max_bytes', parseIntOr(e.target.value, -1))}
            />
            <p className="text-xs text-content-tertiary mt-1">-1 for unlimited</p>
          </div>

          {/* TTL */}
          <div>
            <label htmlFor="obj-ttl" className="block text-sm font-medium text-gray-700 mb-1">TTL (nanoseconds)</label>
            <Input
              id="obj-ttl"
              type="number"
              value={value.ttl ?? 0}
              onChange={(e) => updateField('ttl', parseIntOr(e.target.value, 0))}
            />
            <p className="text-xs text-content-tertiary mt-1">0 for no expiry</p>
          </div>
        </div>
      </SectionPanel>

      {/* Storage Section */}
      <SectionPanel label="Storage Options" isOpen={openSections.storage} onToggle={() => toggleSection('storage')}>
        <div className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            {/* Storage */}
            <ImmutableField label="Storage Type" isImmutable={isEditMode}>
              <Dropdown
                value={value.storage || 'file'}
                onChange={(v) => updateField('storage', v as 'file' | 'memory')}
                disabled={isEditMode}
                options={[
                  { value: 'file', label: 'File' },
                  { value: 'memory', label: 'Memory' },
                ]}
              />
            </ImmutableField>

            {/* Replicas */}
            <ImmutableField label="Replicas" isImmutable={isEditMode}>
              <Input
                type="number"
                min={1}
                max={5}
                value={value.num_replicas ?? 1}
                onChange={(e) => updateField('num_replicas', parseIntOr(e.target.value, 1))}
                disabled={isEditMode}
              />
            </ImmutableField>
          </div>

          {/* Compression */}
          <ImmutableField label="Compression" isImmutable={isEditMode}>
            <label className="flex items-center gap-2">
              <input
                type="checkbox"
                checked={value.compression ?? false}
                onChange={(e) => updateField('compression', e.target.checked)}
                disabled={isEditMode}
                className="rounded border-border-strong disabled:cursor-not-allowed"
              />
              <span className="text-sm text-gray-700">Enabled</span>
            </label>
          </ImmutableField>
        </div>
      </SectionPanel>
    </div>
  )
}
