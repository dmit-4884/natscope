import { useState } from 'react'
import { parseIntOr } from '@/utils/numbers'
import { Input, ImmutableField, Dropdown } from '@/components/ui'
import type { KVBucketConfig } from '@/types/management'
import { SectionPanel } from '@/components/common/forms/SectionPanel'
import { PlacementEditor } from '@/components/common/forms/editors/PlacementEditor'
import { RePublishEditor } from '@/components/common/forms/editors/RePublishEditor'
import { StreamSourceEditor } from '@/components/common/forms/editors/StreamSourceEditor'
import { StreamSourcesArrayEditor } from '@/components/common/forms/editors/StreamSourcesArrayEditor'
import { KeyValueInput } from '@/components/common/forms/inputs/KeyValueInput'

interface KVBucketFormFieldsProps {
  value: KVBucketConfig
  onChange: (value: KVBucketConfig) => void
  isEditMode: boolean
}

export function KVBucketFormFields({
  value,
  onChange,
  isEditMode,
}: KVBucketFormFieldsProps) {
  const [openSections, setOpenSections] = useState<Record<string, boolean>>({
    basic: true,
    limits: false,
    storage: false,
    placement: false,
    mirror: false,
    sources: false,
    republish: false,
    metadata: false,
  })

  const toggleSection = (section: string) => {
    setOpenSections((prev) => ({ ...prev, [section]: !prev[section] }))
  }

  const updateField = <K extends keyof KVBucketConfig>(
    field: K,
    fieldValue: KVBucketConfig[K]
  ) => {
    onChange({ ...value, [field]: fieldValue })
  }

  return (
    <div className="space-y-4">
      {/* Basic Section */}
      <SectionPanel label="Basic Configuration" isOpen={openSections.basic} onToggle={() => toggleSection('basic')}>
        <div className="space-y-4">
          {/* Bucket Name */}
          <ImmutableField label="Bucket Name" isImmutable={isEditMode} helpText="Unique identifier for the KV bucket">
            <Input
              value={value.bucket || ''}
              onChange={(e) => updateField('bucket', e.target.value)}
              placeholder="my-kv-bucket"
              disabled={isEditMode}
            />
          </ImmutableField>

          {/* Description */}
          <div>
            <label htmlFor="kv-description" className="block text-sm font-medium text-gray-700 mb-1">Description</label>
            <Input
              id="kv-description"
              value={value.description || ''}
              onChange={(e) => updateField('description', e.target.value)}
              placeholder="Optional description"
            />
          </div>

          {/* History */}
          <div>
            <label htmlFor="kv-history" className="block text-sm font-medium text-gray-700 mb-1">History (revisions per key)</label>
            <Input
              id="kv-history"
              type="number"
              min={1}
              max={64}
              value={value.history ?? 1}
              onChange={(e) => updateField('history', parseIntOr(e.target.value, 1))}
            />
            <p className="text-xs text-content-tertiary mt-1">
              Number of historical values to keep per key (1-64)
            </p>
          </div>
        </div>
      </SectionPanel>

      {/* Limits Section */}
      <SectionPanel label="Limits" isOpen={openSections.limits} onToggle={() => toggleSection('limits')}>
        <div className="grid grid-cols-2 gap-4">
          {/* Max Value Size */}
          <div>
            <label htmlFor="kv-max-value-size" className="block text-sm font-medium text-gray-700 mb-1">Max Value Size</label>
            <Input
              id="kv-max-value-size"
              type="number"
              value={value.max_value_size ?? -1}
              onChange={(e) => updateField('max_value_size', parseIntOr(e.target.value, -1))}
            />
            <p className="text-xs text-content-tertiary mt-1">-1 for unlimited</p>
          </div>

          {/* Max Bytes */}
          <div>
            <label htmlFor="kv-max-bytes" className="block text-sm font-medium text-gray-700 mb-1">Max Bytes</label>
            <Input
              id="kv-max-bytes"
              type="number"
              value={value.max_bytes ?? -1}
              onChange={(e) => updateField('max_bytes', parseIntOr(e.target.value, -1))}
            />
            <p className="text-xs text-content-tertiary mt-1">-1 for unlimited</p>
          </div>

          {/* TTL */}
          <div>
            <label htmlFor="kv-ttl" className="block text-sm font-medium text-gray-700 mb-1">TTL (nanoseconds)</label>
            <Input
              id="kv-ttl"
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

      {/* Placement Section */}
      <SectionPanel label="Placement" isOpen={openSections.placement} onToggle={() => toggleSection('placement')}>
        <p className="text-xs text-content-tertiary mb-3">Pin the bucket to a specific cluster or set of server tags.</p>
        <PlacementEditor value={value.placement} onChange={(next) => updateField('placement', next)} />
      </SectionPanel>

      {/* Mirror Section (NATS 2.11+) */}
      <SectionPanel label="Mirror" isOpen={openSections.mirror} onToggle={() => toggleSection('mirror')}>
        <p className="text-xs text-content-tertiary mb-3">
          Mirror exactly one upstream KV bucket. Requires NATS 2.11+. Immutable after creation.
        </p>
        <ImmutableField label="" isImmutable={isEditMode} helpText="Mirror cannot be changed after creation.">
          <StreamSourceEditor
            value={value.mirror}
            onChange={(next) => updateField('mirror', next)}
            hideRemove
          />
        </ImmutableField>
      </SectionPanel>

      {/* Sources Section (NATS 2.11+) */}
      <SectionPanel label="Sources" isOpen={openSections.sources} onToggle={() => toggleSection('sources')}>
        <p className="text-xs text-content-tertiary mb-3">
          Aggregate from one or more upstream KV buckets. Requires NATS 2.11+.
        </p>
        <StreamSourcesArrayEditor
          value={value.sources}
          onChange={(next) => updateField('sources', next)}
        />
      </SectionPanel>

      {/* Republish Section */}
      <SectionPanel label="Republish" isOpen={openSections.republish} onToggle={() => toggleSection('republish')}>
        <p className="text-xs text-content-tertiary mb-3">Re-publish bucket operations to a different subject.</p>
        <RePublishEditor value={value.republish} onChange={(next) => updateField('republish', next)} />
      </SectionPanel>

      {/* Metadata Section */}
      <SectionPanel label="Metadata" isOpen={openSections.metadata} onToggle={() => toggleSection('metadata')}>
        <p className="text-xs text-content-tertiary mb-3">User-defined key/value labels attached to the bucket.</p>
        <KeyValueInput
          value={value.metadata}
          onChange={(next) => updateField('metadata', next)}
          keyPlaceholder="env"
          valuePlaceholder="production"
        />
      </SectionPanel>
    </div>
  )
}
