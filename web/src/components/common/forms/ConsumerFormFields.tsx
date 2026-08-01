import { useState } from 'react'
import { parseIntOr } from '@/utils/numbers'
import { Input, Badge, ImmutableField, Dropdown, Toggle } from '@/components/ui'
import { CONSUMER_IMMUTABLE_FIELDS } from '@/types/management'
import type { ConsumerCreateRequest } from '@/types/management'
import { SectionPanel } from './SectionPanel'
import { StringArrayInput } from './inputs/StringArrayInput'
import { NumberArrayInput } from './inputs/NumberArrayInput'
import { KeyValueInput } from './inputs/KeyValueInput'

export interface ConsumerFormFieldsProps {
  value: ConsumerCreateRequest
  onChange: (value: ConsumerCreateRequest) => void
  isEditMode: boolean
  immutableFields?: string[]
}

function utcToLocalInput(iso: string | undefined): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  return new Date(d.getTime() - d.getTimezoneOffset() * 60000).toISOString().slice(0, 16)
}

function localInputToUtc(local: string): string | undefined {
  if (!local) return undefined
  const d = new Date(local)
  return Number.isNaN(d.getTime()) ? undefined : d.toISOString()
}

export function ConsumerFormFields({
  value,
  onChange,
  isEditMode,
  immutableFields = [],
}: ConsumerFormFieldsProps) {
  const [openSections, setOpenSections] = useState<Record<string, boolean>>({
    basic: true,
    delivery: false,
    ack: false,
    limits: false,
    push: false,
    replication: false,
    metadata: false,
  })

  const toggleSection = (section: string) => {
    setOpenSections((prev) => ({ ...prev, [section]: !prev[section] }))
  }

  const isImmutable = (field: string) => {
    return isEditMode && (immutableFields.includes(field) || CONSUMER_IMMUTABLE_FIELDS.includes(field as typeof CONSUMER_IMMUTABLE_FIELDS[number]))
  }

  const updateField = <K extends keyof ConsumerCreateRequest>(
    field: K,
    fieldValue: ConsumerCreateRequest[K]
  ) => {
    onChange({ ...value, [field]: fieldValue })
  }

  const isPushConsumer = !!value.deliver_subject
  const isEphemeral = value.ephemeral ?? false
  const filterSubjects = value.filter_subjects ?? []
  const singleFilterLocked = isImmutable('filter_subject') || filterSubjects.length > 0

  return (
    <div className="space-y-4">
      {/* Basic Section */}
      <SectionPanel label="Basic Configuration" isOpen={openSections.basic} onToggle={() => toggleSection('basic')}>
        <div className="space-y-4">
            {/* Name */}
            <ImmutableField label="Name" isImmutable={isImmutable('name')} helpText="Unique identifier for the consumer">
              <Input
                value={value.name || ''}
                onChange={(e) => updateField('name', e.target.value)}
                placeholder="my-consumer"
                disabled={isImmutable('name')}
              />
            </ImmutableField>

            <ImmutableField
              label="Ephemeral consumer"
              isImmutable={isImmutable('ephemeral')}
              helpText={
                isEphemeral
                  ? 'The server removes this consumer once it stays inactive for the inactivity threshold.'
                  : 'Durable consumers survive restarts and stay until deleted.'
              }
            >
              <Toggle
                checked={isEphemeral}
                onChange={(next) => updateField('ephemeral', next)}
                disabled={isImmutable('ephemeral')}
                label="Ephemeral consumer"
                testId="consumer-ephemeral-toggle"
              />
            </ImmutableField>

            {/* Description */}
            <div>
              <label htmlFor="consumer-description" className="block text-sm font-medium text-gray-700 mb-1">Description</label>
              <Input
                id="consumer-description"
                value={value.description || ''}
                onChange={(e) => updateField('description', e.target.value)}
                placeholder="Optional description"
              />
            </div>

            {/* Filter Subject */}
            <ImmutableField
              label="Filter Subject"
              isImmutable={isImmutable('filter_subject')}
              helpText={
                singleFilterLocked
                  ? 'Disabled while Filter Subjects (multiple) is in use — empty that list to go back to a single filter.'
                  : 'Single subject pattern to filter messages (supports wildcards). Leave empty to receive every subject.'
              }
            >
              <Input
                value={value.filter_subject || ''}
                onChange={(e) =>
                  onChange({
                    ...value,
                    filter_subject: e.target.value,
                    filter_subjects: e.target.value ? [] : value.filter_subjects,
                  })
                }
                placeholder="orders.>"
                disabled={singleFilterLocked}
              />
            </ImmutableField>

            {/* Filter Subjects (NATS 2.10+) */}
            <ImmutableField
              label="Filter Subjects (multiple)"
              isImmutable={isImmutable('filter_subjects')}
              helpText="List of subject patterns (NATS 2.10+). Adding an entry clears the single Filter Subject — the two are mutually exclusive."
            >
              <StringArrayInput
                value={value.filter_subjects}
                onChange={(next) =>
                  onChange({
                    ...value,
                    filter_subjects: next,
                    filter_subject: next.length > 0 ? '' : value.filter_subject,
                  })
                }
                placeholder="orders.created"
                disabled={isImmutable('filter_subjects')}
              />
            </ImmutableField>
        </div>
      </SectionPanel>

      {/* Delivery Section */}
      <SectionPanel label="Delivery Policy" isOpen={openSections.delivery} onToggle={() => toggleSection('delivery')}>
        <div className="space-y-4">
            {/* Deliver Policy */}
            <ImmutableField label="Deliver Policy" isImmutable={isImmutable('deliver_policy')}>
              <Dropdown
                value={value.deliver_policy || 'all'}
                onChange={(v) => updateField('deliver_policy', v as ConsumerCreateRequest['deliver_policy'])}
                disabled={isImmutable('deliver_policy')}
                options={[
                  { value: 'all', label: 'All' },
                  { value: 'last', label: 'Last' },
                  { value: 'new', label: 'New' },
                  { value: 'by_start_sequence', label: 'By Start Sequence' },
                  { value: 'by_start_time', label: 'By Start Time' },
                  { value: 'last_per_subject', label: 'Last Per Subject' },
                ]}
              />
            </ImmutableField>

            {/* Conditional: Start Sequence */}
            {value.deliver_policy === 'by_start_sequence' && (
              <div>
                <label htmlFor="consumer-opt-start-seq" className="block text-sm font-medium text-gray-700 mb-1">Start Sequence</label>
                <Input
                  id="consumer-opt-start-seq"
                  type="number"
                  value={value.opt_start_seq ?? 1}
                  onChange={(e) => updateField('opt_start_seq', parseIntOr(e.target.value, 1))}
                  disabled={isImmutable('opt_start_seq')}
                />
              </div>
            )}

            {/* Conditional: Start Time */}
            {value.deliver_policy === 'by_start_time' && (
              <div>
                <label htmlFor="consumer-opt-start-time" className="block text-sm font-medium text-gray-700 mb-1">Start Time</label>
                <Input
                  id="consumer-opt-start-time"
                  type="datetime-local"
                  value={utcToLocalInput(value.opt_start_time)}
                  onChange={(e) => updateField('opt_start_time', localInputToUtc(e.target.value))}
                  disabled={isImmutable('opt_start_time')}
                />
              </div>
            )}

            {/* Replay Policy */}
            <ImmutableField label="Replay Policy" isImmutable={isImmutable('replay_policy')} helpText="Instant: deliver as fast as possible. Original: match original timing.">
              <Dropdown
                value={value.replay_policy || 'instant'}
                onChange={(v) => updateField('replay_policy', v as 'instant' | 'original')}
                disabled={isImmutable('replay_policy')}
                options={[
                  { value: 'instant', label: 'Instant' },
                  { value: 'original', label: 'Original' },
                ]}
              />
            </ImmutableField>

            {/* Headers Only */}
            <ImmutableField label="Headers Only" isImmutable={isImmutable('headers_only')}>
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={value.headers_only ?? false}
                  onChange={(e) => updateField('headers_only', e.target.checked)}
                  disabled={isImmutable('headers_only')}
                  className="rounded border-border-strong disabled:cursor-not-allowed"
                />
                <span className="text-sm text-gray-700">Enabled</span>
              </label>
            </ImmutableField>
        </div>
      </SectionPanel>

      {/* Ack Section */}
      <SectionPanel label="Acknowledgement" isOpen={openSections.ack} onToggle={() => toggleSection('ack')}>
        <div className="space-y-4">
            {/* Ack Policy */}
            <ImmutableField label="Ack Policy" isImmutable={isImmutable('ack_policy')}>
              <Dropdown
                value={value.ack_policy || 'explicit'}
                onChange={(v) => updateField('ack_policy', v as 'none' | 'all' | 'explicit')}
                disabled={isImmutable('ack_policy')}
                options={[
                  { value: 'explicit', label: 'Explicit' },
                  { value: 'all', label: 'All' },
                  { value: 'none', label: 'None' },
                ]}
              />
            </ImmutableField>

            {/* Ack Wait */}
            <div>
              <label htmlFor="consumer-ack-wait" className="block text-sm font-medium text-gray-700 mb-1">Ack Wait (ns)</label>
              <Input
                id="consumer-ack-wait"
                type="number"
                value={value.ack_wait ?? 30000000000}
                onChange={(e) => updateField('ack_wait', parseIntOr(e.target.value, 30000000000))}
              />
              <p className="text-xs text-content-tertiary mt-1">
                Time to wait for ack before redelivery (default: 30s)
              </p>
            </div>

            {/* Max Deliver */}
            <div>
              <label htmlFor="consumer-max-deliver" className="block text-sm font-medium text-gray-700 mb-1">Max Deliver</label>
              <Input
                id="consumer-max-deliver"
                type="number"
                value={value.max_deliver ?? -1}
                onChange={(e) => updateField('max_deliver', parseIntOr(e.target.value, -1))}
              />
              <p className="text-xs text-content-tertiary mt-1">
                Maximum number of delivery attempts (-1 for unlimited)
              </p>
            </div>

            {/* Max Ack Pending */}
            <div>
              <label htmlFor="consumer-max-ack-pending" className="block text-sm font-medium text-gray-700 mb-1">Max Ack Pending</label>
              <Input
                id="consumer-max-ack-pending"
                type="number"
                value={value.max_ack_pending ?? 1000}
                onChange={(e) => updateField('max_ack_pending', parseIntOr(e.target.value, 1000))}
              />
              <p className="text-xs text-content-tertiary mt-1">
                Maximum outstanding acks allowed
              </p>
            </div>

            {/* Backoff: a variable-length list of inputs, so no single htmlFor target. */}
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Backoff (ns per attempt)</label>
              <NumberArrayInput
                value={value.backoff}
                onChange={(next) => updateField('backoff', next)}
                placeholder="1000000000"
              />
              <p className="text-xs text-content-tertiary mt-1">
                Per-attempt redelivery delays in nanoseconds. Empty array = use Ack Wait for all retries.
              </p>
            </div>

            {/* Sample Frequency */}
            <div>
              <label htmlFor="consumer-sample-freq" className="block text-sm font-medium text-gray-700 mb-1">Sample Frequency</label>
              <Input
                id="consumer-sample-freq"
                value={value.sample_freq || ''}
                onChange={(e) => updateField('sample_freq', e.target.value)}
                placeholder="100 or 50%"
              />
              <p className="text-xs text-content-tertiary mt-1">
                Acknowledgement sampling for observability (e.g. "100" or "50%"). Empty = disabled.
              </p>
            </div>
        </div>
      </SectionPanel>

      {/* Limits Section */}
      <SectionPanel label="Limits" isOpen={openSections.limits} onToggle={() => toggleSection('limits')}>
        <div className="grid grid-cols-2 gap-4">
            {/* Max Waiting */}
            <div>
              <label htmlFor="consumer-max-waiting" className="block text-sm font-medium text-gray-700 mb-1">Max Waiting</label>
              <Input
                id="consumer-max-waiting"
                type="number"
                value={value.max_waiting ?? 512}
                onChange={(e) => updateField('max_waiting', parseIntOr(e.target.value, 512))}
              />
              <p className="text-xs text-content-tertiary mt-1">Max pull requests waiting</p>
            </div>

            {/* Max Batch */}
            <div>
              <label htmlFor="consumer-max-batch" className="block text-sm font-medium text-gray-700 mb-1">Max Batch</label>
              <Input
                id="consumer-max-batch"
                type="number"
                value={value.max_batch ?? 0}
                onChange={(e) => updateField('max_batch', parseIntOr(e.target.value, 0))}
              />
              <p className="text-xs text-content-tertiary mt-1">Max messages per pull</p>
            </div>

            {/* Max Bytes */}
            <div>
              <label htmlFor="consumer-max-bytes" className="block text-sm font-medium text-gray-700 mb-1">Max Bytes</label>
              <Input
                id="consumer-max-bytes"
                type="number"
                value={value.max_bytes ?? 0}
                onChange={(e) => updateField('max_bytes', parseIntOr(e.target.value, 0))}
              />
              <p className="text-xs text-content-tertiary mt-1">Max bytes per pull</p>
            </div>

            {/* Max Expires */}
            <div>
              <label htmlFor="consumer-max-expires" className="block text-sm font-medium text-gray-700 mb-1">Max Expires (ns)</label>
              <Input
                id="consumer-max-expires"
                type="number"
                value={value.max_expires ?? 0}
                onChange={(e) => updateField('max_expires', parseIntOr(e.target.value, 0))}
              />
              <p className="text-xs text-content-tertiary mt-1">Max wait time for pull</p>
            </div>

            {/* Inactive Threshold */}
            <div>
              <label htmlFor="consumer-inactive-threshold" className="block text-sm font-medium text-gray-700 mb-1">Inactive Threshold (ns)</label>
              <Input
                id="consumer-inactive-threshold"
                type="number"
                value={value.inactive_threshold ?? 0}
                onChange={(e) => updateField('inactive_threshold', parseIntOr(e.target.value, 0))}
              />
              <p className="text-xs text-content-tertiary mt-1">Cleanup ephemeral after</p>
            </div>

            {/* Rate Limit */}
            <div>
              <label htmlFor="consumer-rate-limit-bps" className="block text-sm font-medium text-gray-700 mb-1">Rate Limit (bps)</label>
              <Input
                id="consumer-rate-limit-bps"
                type="number"
                value={value.rate_limit_bps ?? 0}
                onChange={(e) => updateField('rate_limit_bps', parseIntOr(e.target.value, 0))}
              />
              <p className="text-xs text-content-tertiary mt-1">0 for unlimited</p>
            </div>
        </div>
      </SectionPanel>

      {/* Push Consumer Section */}
      <SectionPanel
        label="Push Consumer"
        isOpen={openSections.push}
        onToggle={() => toggleSection('push')}
        badge={isPushConsumer ? <Badge variant="primary" size="sm">Active</Badge> : undefined}
      >
        <div className="space-y-4">
            <p className="text-xs text-content-tertiary">
              Set a deliver subject to create a push consumer. Leave empty for a pull consumer.
            </p>

            {/* Deliver Subject */}
            <ImmutableField label="Deliver Subject" isImmutable={isImmutable('deliver_subject')}>
              <Input
                value={value.deliver_subject || ''}
                onChange={(e) => updateField('deliver_subject', e.target.value || undefined)}
                placeholder="my.delivery.subject"
                disabled={isImmutable('deliver_subject')}
              />
            </ImmutableField>

            {isPushConsumer && (
              <>
                {/* Deliver Group */}
                <ImmutableField label="Deliver Group" isImmutable={isImmutable('deliver_group')}>
                  <Input
                    value={value.deliver_group || ''}
                    onChange={(e) => updateField('deliver_group', e.target.value || undefined)}
                    placeholder="Optional queue group"
                    disabled={isImmutable('deliver_group')}
                  />
                </ImmutableField>

                {/* Flow Control */}
                <ImmutableField label="Flow Control" isImmutable={isImmutable('flow_control')}>
                  <label className="flex items-center gap-2">
                    <input
                      type="checkbox"
                      checked={value.flow_control ?? false}
                      onChange={(e) => updateField('flow_control', e.target.checked)}
                      disabled={isImmutable('flow_control')}
                      className="rounded border-border-strong disabled:cursor-not-allowed"
                    />
                    <span className="text-sm text-gray-700">Enabled</span>
                  </label>
                </ImmutableField>

                {/* Idle Heartbeat */}
                <ImmutableField label="Idle Heartbeat (ns)" isImmutable={isImmutable('idle_heartbeat')} helpText="Heartbeat interval when no messages">
                  <Input
                    id="consumer-idle-heartbeat"
                    type="number"
                    value={value.idle_heartbeat ?? 0}
                    onChange={(e) => updateField('idle_heartbeat', parseIntOr(e.target.value, 0))}
                    disabled={isImmutable('idle_heartbeat')}
                  />
                </ImmutableField>
              </>
            )}
        </div>
      </SectionPanel>

      {/* Replication & Storage Section */}
      <SectionPanel label="Replication & Storage" isOpen={openSections.replication} onToggle={() => toggleSection('replication')}>
        <div className="space-y-4">
            {/* Replicas */}
            <ImmutableField
              label="Replicas"
              isImmutable={isImmutable('num_replicas')}
              helpText="Number of consumer replicas across the cluster (1, 3, or 5). 0 = inherit from stream."
            >
              <Input
                type="number"
                value={value.num_replicas ?? 0}
                onChange={(e) => updateField('num_replicas', parseIntOr(e.target.value, 0))}
                disabled={isImmutable('num_replicas')}
              />
            </ImmutableField>

            {/* Memory Storage */}
            <ImmutableField
              label="Memory Storage"
              isImmutable={isImmutable('memory_storage')}
              helpText="Store consumer state in memory (no disk persistence). Cannot be changed after creation."
            >
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={value.memory_storage ?? false}
                  onChange={(e) => updateField('memory_storage', e.target.checked)}
                  disabled={isImmutable('memory_storage')}
                  className="rounded border-border-strong disabled:cursor-not-allowed"
                />
                <span className="text-sm text-gray-700">Enabled</span>
              </label>
            </ImmutableField>
        </div>
      </SectionPanel>

      {/* Metadata Section */}
      <SectionPanel label="Metadata" isOpen={openSections.metadata} onToggle={() => toggleSection('metadata')}>
        <div className="space-y-4">
            <p className="text-xs text-content-tertiary">
              User-defined key/value labels attached to the consumer. Visible to NATS-aware tooling
              (CLI, dashboards) for grouping and tagging — not interpreted by the server.
            </p>
            <KeyValueInput
              value={value.metadata}
              onChange={(next) => updateField('metadata', next)}
              keyPlaceholder="env"
              valuePlaceholder="production"
            />
        </div>
      </SectionPanel>
    </div>
  )
}
