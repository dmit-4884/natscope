import type { CapabilityKey } from '@/contexts/connection'
import type { StreamCreateRequest, StreamUpdateRequest } from '@/types/management'

// UI-only metadata for stream config fields (rendering + payload filtering);
// backend is source of truth.

type StreamFieldType =
  | 'text'
  | 'number'
  | 'duration_ns'
  | 'boolean'
  | 'select'
  | 'subjects'

export type StreamFieldSection = 'basic' | 'limits' | 'retention' | 'advanced'

interface StreamFieldOption {
  value: string
  label: string
}

export interface StreamFieldDef {
  /** Property name on StreamCreateRequest. */
  key: keyof StreamCreateRequest
  /** Human-readable label. */
  label: string
  /** Renderer type. */
  type: StreamFieldType
  /** Visual grouping section in the form. */
  section: StreamFieldSection
  /** Editable when the form is in create mode. */
  editableOnCreate: boolean
  /**
   * Editable when the form is in edit (update) mode. False => excluded from
   * update payload.
   */
  editableOnUpdate: boolean
  /** Helper text shown under the field. */
  helperText?: string
  /** Reason shown under the field when it is read-only on update. */
  immutableReason?: string
  /** Options for select fields. */
  options?: StreamFieldOption[]
  /** Placeholder for text inputs. */
  placeholder?: string
  /** Default value used when the field is missing from the form value. */
  defaultValue?: unknown
  /** Server capability required to use this field; resolved by the form via useServerCapabilities. */
  requiresCapability?: CapabilityKey
}

export const STREAM_SECTIONS: ReadonlyArray<{ key: StreamFieldSection; label: string }> = [
  { key: 'basic', label: 'Basic Configuration' },
  { key: 'limits', label: 'Limits' },
  { key: 'retention', label: 'Retention & Storage' },
  { key: 'advanced', label: 'Advanced Options' },
]

export const STREAM_FIELDS: ReadonlyArray<StreamFieldDef> = [
  {
    key: 'name',
    label: 'Name',
    type: 'text',
    section: 'basic',
    editableOnCreate: true,
    editableOnUpdate: false,
    helperText: 'Unique identifier for the stream.',
    immutableReason: 'Stream name cannot be changed after creation.',
    placeholder: 'my-stream',
  },
  {
    key: 'description',
    label: 'Description',
    type: 'text',
    section: 'basic',
    editableOnCreate: true,
    editableOnUpdate: true,
    placeholder: 'Optional description',
  },
  {
    key: 'subjects',
    label: 'Subjects',
    type: 'subjects',
    section: 'basic',
    editableOnCreate: true,
    editableOnUpdate: true,
    helperText: 'NATS subject patterns (supports wildcards * and >).',
  },
  {
    key: 'retention',
    label: 'Retention',
    type: 'select',
    section: 'retention',
    editableOnCreate: true,
    editableOnUpdate: false,
    immutableReason: 'Retention policy cannot be changed after creation.',
    defaultValue: 'limits',
    options: [
      { value: 'limits', label: 'Limits' },
      { value: 'interest', label: 'Interest' },
      { value: 'workqueue', label: 'Work Queue' },
    ],
  },
  {
    key: 'storage',
    label: 'Storage',
    type: 'select',
    section: 'retention',
    editableOnCreate: true,
    editableOnUpdate: false,
    immutableReason: 'Storage backend cannot be changed after creation.',
    defaultValue: 'file',
    options: [
      { value: 'file', label: 'File' },
      { value: 'memory', label: 'Memory' },
    ],
  },
  {
    key: 'discard',
    label: 'Discard Policy',
    type: 'select',
    section: 'retention',
    editableOnCreate: true,
    editableOnUpdate: true,
    defaultValue: 'old',
    options: [
      { value: 'old', label: 'Old' },
      { value: 'new', label: 'New' },
    ],
  },
  {
    key: 'num_replicas',
    label: 'Replicas',
    type: 'number',
    section: 'retention',
    editableOnCreate: true,
    editableOnUpdate: false,
    immutableReason: 'Replica count cannot be changed via this form.',
    defaultValue: 1,
  },
  {
    key: 'max_msgs',
    label: 'Max Messages',
    type: 'number',
    section: 'limits',
    editableOnCreate: true,
    editableOnUpdate: true,
    helperText: '-1 for unlimited.',
    defaultValue: -1,
  },
  {
    key: 'max_bytes',
    label: 'Max Bytes',
    type: 'number',
    section: 'limits',
    editableOnCreate: true,
    editableOnUpdate: true,
    helperText: '-1 for unlimited.',
    defaultValue: -1,
  },
  {
    key: 'max_age',
    label: 'Max Age (ns)',
    type: 'duration_ns',
    section: 'limits',
    editableOnCreate: true,
    editableOnUpdate: true,
    helperText: '0 for unlimited.',
    defaultValue: 0,
  },
  {
    key: 'max_msg_size',
    label: 'Max Message Size',
    type: 'number',
    section: 'limits',
    editableOnCreate: true,
    editableOnUpdate: true,
    helperText: '-1 for unlimited.',
    defaultValue: -1,
  },
  {
    key: 'max_msgs_per_subject',
    label: 'Max Messages Per Subject',
    type: 'number',
    section: 'limits',
    editableOnCreate: true,
    editableOnUpdate: true,
    helperText: '-1 for unlimited.',
    defaultValue: -1,
  },
  {
    key: 'duplicate_window',
    label: 'Duplicate Window (ns)',
    type: 'duration_ns',
    section: 'advanced',
    editableOnCreate: true,
    editableOnUpdate: true,
    helperText: 'Time window for duplicate detection. 0 to disable.',
    defaultValue: 0,
  },
  {
    key: 'deny_delete',
    label: 'Deny Delete',
    type: 'boolean',
    section: 'advanced',
    editableOnCreate: true,
    editableOnUpdate: false,
    immutableReason: 'Cannot be changed after creation.',
  },
  {
    key: 'deny_purge',
    label: 'Deny Purge',
    type: 'boolean',
    section: 'advanced',
    editableOnCreate: true,
    editableOnUpdate: false,
    immutableReason: 'Cannot be changed after creation.',
  },
  {
    key: 'allow_direct',
    label: 'Allow Direct',
    type: 'boolean',
    section: 'advanced',
    editableOnCreate: true,
    editableOnUpdate: true,
  },
  {
    key: 'allow_rollup',
    label: 'Allow Rollup',
    type: 'boolean',
    section: 'advanced',
    editableOnCreate: true,
    editableOnUpdate: false,
    immutableReason: 'Cannot be changed after creation.',
  },
  {
    key: 'compression',
    label: 'Compression',
    type: 'select',
    section: 'advanced',
    editableOnCreate: true,
    editableOnUpdate: true,
    helperText: 'S2 compression reduces stream storage size at a small CPU cost. Mutable since NATS 2.10.',
    defaultValue: 'none',
    options: [
      { value: 'none', label: 'None' },
      { value: 's2', label: 'S2' },
    ],
  },
  {
    key: 'first_seq',
    label: 'First Sequence',
    type: 'number',
    section: 'advanced',
    editableOnCreate: true,
    editableOnUpdate: false,
    immutableReason: 'Initial sequence cannot be changed after creation.',
    helperText: 'Initial sequence number assigned to the first published message. Default 1.',
    defaultValue: 1,
  },
  {
    key: 'allow_msg_ttl',
    label: 'Allow Per-Message TTL',
    type: 'boolean',
    section: 'advanced',
    editableOnCreate: true,
    editableOnUpdate: true,
    helperText: 'Enables Nats-TTL header to expire individual messages. Requires NATS 2.11+.',
    requiresCapability: 'messageTtl',
  },
  {
    key: 'allow_atomic_publish',
    label: 'Allow Atomic Publish',
    type: 'boolean',
    section: 'advanced',
    editableOnCreate: true,
    editableOnUpdate: true,
    helperText: 'Enables atomic batch publish API. Requires NATS 2.12+.',
    requiresCapability: 'atomicPublish',
  },
  {
    key: 'discard_new_per_subject',
    label: 'Discard New Per Subject',
    type: 'boolean',
    section: 'advanced',
    editableOnCreate: true,
    editableOnUpdate: true,
    helperText: 'Apply DiscardNew policy independently per subject.',
  },
  {
    key: 'mirror_direct',
    label: 'Mirror Direct Get',
    type: 'boolean',
    section: 'advanced',
    editableOnCreate: true,
    editableOnUpdate: true,
    helperText: 'Allow direct-get API on mirror streams.',
  },
  {
    key: 'no_ack',
    label: 'No Ack',
    type: 'boolean',
    section: 'advanced',
    editableOnCreate: true,
    editableOnUpdate: false,
    immutableReason: 'Cannot be changed after creation.',
    helperText: 'Disable message acknowledgements (consumers ignore ack policy).',
  },
  {
    key: 'max_consumers',
    label: 'Max Consumers',
    type: 'number',
    section: 'limits',
    editableOnCreate: true,
    editableOnUpdate: true,
    helperText: '-1 for unlimited.',
    defaultValue: -1,
  },
]

// Update-DTO fields not surfaced as form inputs but passed through (edited via
// JSON view).
const STREAM_UPDATE_PASSTHROUGH_KEYS: ReadonlyArray<keyof StreamUpdateRequest> = [
  'metadata',
  'sources',
  'republish',
  'subject_transform',
  'consumer_limits',
]

// Create-DTO fields not surfaced as form inputs but passed through (placement,
// mirror, nested config via JSON view).
const STREAM_CREATE_PASSTHROUGH_KEYS: ReadonlyArray<keyof StreamCreateRequest> = [
  'placement',
  'mirror',
  'sources',
  'metadata',
  'republish',
  'subject_transform',
  'consumer_limits',
]

/** Field keys that cannot be changed when updating an existing stream. */
export function getStreamUpdateLockedKeys(): readonly string[] {
  return STREAM_FIELDS.filter((f) => !f.editableOnUpdate).map((f) => f.key as string)
}

/**
 * Build an update payload by picking only fields that are editable on update.
 */
export function buildStreamUpdatePayload(values: StreamCreateRequest): StreamUpdateRequest {
  const src = values as unknown as Record<string, unknown>
  const out: Record<string, unknown> = {}

  for (const def of STREAM_FIELDS) {
    if (!def.editableOnUpdate) continue
    const v = src[def.key as string]
    if (v !== undefined) out[def.key as string] = v
  }
  for (const k of STREAM_UPDATE_PASSTHROUGH_KEYS) {
    const v = src[k as string]
    if (v !== undefined) out[k as string] = v
  }
  return out as unknown as StreamUpdateRequest
}

/**
 * Build a create payload by picking only fields that are editable on create.
 */
export function buildStreamCreatePayload(values: StreamCreateRequest): StreamCreateRequest {
  const src = values as unknown as Record<string, unknown>
  const out: Record<string, unknown> = {}

  for (const def of STREAM_FIELDS) {
    if (!def.editableOnCreate) continue
    const v = src[def.key as string]
    if (v !== undefined) out[def.key as string] = v
  }
  for (const k of STREAM_CREATE_PASSTHROUGH_KEYS) {
    const v = src[k as string]
    if (v !== undefined) out[k as string] = v
  }
  return out as unknown as StreamCreateRequest
}
