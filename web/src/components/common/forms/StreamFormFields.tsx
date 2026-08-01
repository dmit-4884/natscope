import { useState } from 'react'
import type { StreamCreateRequest, StreamSource } from '@/types/management'
import { ConfigField, type ConfigFieldMode } from '@/components/streams/config/ConfigField'
import {
  STREAM_FIELDS,
  STREAM_SECTIONS,
  type StreamFieldDef,
  type StreamFieldSection,
} from '@/components/streams/config/streamFieldDefinitions'
import { isMirrorConfigured } from '@/components/streams/config/streamConfigUtils'
import { useActiveConnection, useServerCapabilities } from '@/contexts/connection'
import { ImmutableField } from '@/components/ui'
import { SectionPanel } from './SectionPanel'
import { PlacementEditor } from './editors/PlacementEditor'
import { RePublishEditor } from './editors/RePublishEditor'
import { SubjectTransformEditor } from './editors/SubjectTransformEditor'
import { ConsumerLimitsEditor } from './editors/ConsumerLimitsEditor'
import { StreamSourceEditor } from './editors/StreamSourceEditor'
import { StreamSourcesArrayEditor } from './editors/StreamSourcesArrayEditor'

export interface StreamFormFieldsProps {
  value: StreamCreateRequest
  onChange: (value: StreamCreateRequest) => void
  isEditMode: boolean
  defaultExpanded?: boolean
}

// Sections beyond the declarative STREAM_FIELDS registry: nested-object
// editors that don't fit its flat key/value contract.
type ComplexSectionKey =
  | 'placement'
  | 'mirror'
  | 'sources'
  | 'republish'
  | 'subjectTransform'
  | 'consumerLimits'

const MIRROR_SUBJECTS_REASON = 'Mirror streams take no subjects — they replicate the source stream instead.'

const COMPLEX_SECTIONS: ReadonlyArray<{ key: ComplexSectionKey; label: string; helper: string }> = [
  { key: 'placement', label: 'Placement', helper: 'Pin the stream to a specific cluster or set of server tags.' },
  { key: 'mirror', label: 'Mirror', helper: 'Replicate exactly one source stream. Immutable after creation.' },
  { key: 'sources', label: 'Sources', helper: 'Aggregate messages from one or more upstream streams. Sources can be added but not removed via update.' },
  { key: 'republish', label: 'Republish', helper: 'Re-publish every incoming message to a different subject (mutable).' },
  { key: 'subjectTransform', label: 'Subject Transform', helper: 'Rewrite subjects on storage (mutable).' },
  { key: 'consumerLimits', label: 'Consumer Limits', helper: 'Default limits inherited by all consumers on this stream (mutable).' },
]

export function StreamFormFields({
  value,
  onChange,
  isEditMode,
  defaultExpanded = false,
}: StreamFormFieldsProps) {
  const mode: ConfigFieldMode = isEditMode ? 'edit' : 'create'

  const { connectionId } = useActiveConnection()
  const { unsupportedReason } = useServerCapabilities(connectionId)

  const [openSections, setOpenSections] = useState<Record<StreamFieldSection | ComplexSectionKey, boolean>>({
    basic: true,
    limits: defaultExpanded,
    retention: defaultExpanded,
    advanced: defaultExpanded,
    placement: false,
    mirror: false,
    sources: false,
    republish: false,
    subjectTransform: false,
    consumerLimits: false,
  })

  const toggleSection = (section: StreamFieldSection | ComplexSectionKey) =>
    setOpenSections((prev) => ({ ...prev, [section]: !prev[section] }))

  const setField = (key: string, next: unknown) =>
    onChange({ ...value, [key]: next } as StreamCreateRequest)

  const mirrorConfigured = isMirrorConfigured(value)

  const setMirror = (next: StreamSource | undefined) => {
    const draft = { ...value, mirror: next } as StreamCreateRequest
    if (isMirrorConfigured(draft)) draft.subjects = []
    onChange(draft)
  }

  return (
    <div className="space-y-4">
      {STREAM_SECTIONS.map((section) => {
        const fields = STREAM_FIELDS.filter((f) => f.section === section.key)
        if (fields.length === 0) return null

        return (
          <SectionPanel
            key={section.key}
            label={section.label}
            isOpen={openSections[section.key]}
            onToggle={() => toggleSection(section.key)}
          >
            <div
              className={section.key === 'basic' ? 'space-y-4' : 'grid grid-cols-2 gap-4'}
            >
              {fields.map((def) => (
                <ConfigField
                  key={def.key as string}
                  def={def as StreamFieldDef}
                  value={(value as unknown as Record<string, unknown>)[def.key as string]}
                  onChange={(v) => setField(def.key as string, v)}
                  mode={mode}
                  unsupportedReason={
                    def.requiresCapability ? unsupportedReason(def.requiresCapability) : undefined
                  }
                  lockedReason={
                    def.key === 'subjects' && mirrorConfigured ? MIRROR_SUBJECTS_REASON : undefined
                  }
                />
              ))}
            </div>
          </SectionPanel>
        )
      })}

      {/* Complex nested-object sections rendered by hand because they don't
          fit the flat key/value shape STREAM_FIELDS expects. */}
      {COMPLEX_SECTIONS.map((s) => (
        <SectionPanel
          key={s.key}
          label={s.label}
          isOpen={openSections[s.key]}
          onToggle={() => toggleSection(s.key)}
        >
          <p className="text-xs text-content-tertiary mb-3">{s.helper}</p>

          {s.key === 'placement' && (
            <PlacementEditor
              value={value.placement}
              onChange={(next) => setField('placement', next)}
            />
          )}

          {s.key === 'mirror' && (
            <ImmutableField
              label=""
              isImmutable={isEditMode}
              helpText="Mirror cannot be changed after creation."
            >
              <StreamSourceEditor
                value={value.mirror}
                onChange={setMirror}
                hideRemove
              />
            </ImmutableField>
          )}

          {s.key === 'sources' && (
            <StreamSourcesArrayEditor
              value={value.sources}
              onChange={(next) => setField('sources', next)}
            />
          )}

          {s.key === 'republish' && (
            <RePublishEditor
              value={value.republish}
              onChange={(next) => setField('republish', next)}
            />
          )}

          {s.key === 'subjectTransform' && (
            <SubjectTransformEditor
              value={value.subject_transform}
              onChange={(next) => setField('subject_transform', next)}
            />
          )}

          {s.key === 'consumerLimits' && (
            <ConsumerLimitsEditor
              value={value.consumer_limits}
              onChange={(next) => setField('consumer_limits', next)}
            />
          )}
        </SectionPanel>
      ))}
    </div>
  )
}
