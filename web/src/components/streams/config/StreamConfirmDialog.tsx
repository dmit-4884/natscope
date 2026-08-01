import { useState } from 'react'
import { DestructiveConfirm, Input, Select } from '@/components/ui'
import type { StreamPurgeRequest } from '@/types/management'
import {
  EMPTY_PURGE_FORM,
  PURGE_MODE_OPTIONS,
  type PurgeFormState,
  type PurgeLimitMode,
  buildPurgeRequest,
  getPurgeFormError,
} from './purgeOptions'

export type StreamConfirmType = 'delete' | 'purge' | 'seal'

interface Props {
  type: StreamConfirmType
  streamName: string
  onCancel: () => void
  onConfirm: (purgeOptions?: StreamPurgeRequest) => void
}

const MESSAGES: Record<StreamConfirmType, (name: string) => React.ReactNode> = {
  delete: (name) => (
    <span>
      This will permanently delete the stream <strong>{name}</strong> and all its messages. This action cannot be undone.
    </span>
  ),
  purge: (name) => (
    <span>
      This will delete messages from the stream <strong>{name}</strong>. The stream configuration will be preserved.
    </span>
  ),
  seal: (name) => (
    <span>
      This will seal the stream <strong>{name}</strong>, making it read-only. This action cannot be undone.
    </span>
  ),
}

export function StreamConfirmDialog({ type, streamName, onCancel, onConfirm }: Props) {
  const [purgeForm, setPurgeForm] = useState<PurgeFormState>(EMPTY_PURGE_FORM)
  const verb = type.charAt(0).toUpperCase() + type.slice(1)
  const purgeError = type === 'purge' ? getPurgeFormError(purgeForm) : undefined

  return (
    <DestructiveConfirm
      isOpen
      title={`${verb} Stream`}
      description={MESSAGES[type](streamName)}
      confirmLabel={verb}
      requireTypedName={streamName}
      confirmDisabled={!!purgeError}
      extra={
        type === 'purge' ? (
          <PurgeOptionsFields value={purgeForm} onChange={setPurgeForm} error={purgeError} />
        ) : undefined
      }
      onCancel={onCancel}
      onConfirm={() => onConfirm(type === 'purge' ? buildPurgeRequest(purgeForm) : undefined)}
    />
  )
}

interface PurgeOptionsFieldsProps {
  value: PurgeFormState
  onChange: (next: PurgeFormState) => void
  error?: string
}

function PurgeOptionsFields({ value, onChange, error }: PurgeOptionsFieldsProps) {
  return (
    <div className="space-y-3">
      <div>
        <label htmlFor="purge-filter" className="block text-sm font-medium text-gray-700 mb-1">
          Subject filter
        </label>
        <Input
          id="purge-filter"
          value={value.filter}
          mono
          placeholder="orders.> (leave empty for every subject)"
          onChange={(e) => onChange({ ...value, filter: e.target.value })}
        />
      </div>

      <div>
        <label htmlFor="purge-mode" className="block text-sm font-medium text-gray-700 mb-1">
          Scope
        </label>
        <Select
          id="purge-mode"
          value={value.mode}
          options={PURGE_MODE_OPTIONS as { value: string; label: string }[]}
          onChange={(e) => onChange({ ...value, mode: e.target.value as PurgeLimitMode })}
        />
      </div>

      {value.mode === 'sequence' && (
        <div>
          <label htmlFor="purge-sequence" className="block text-sm font-medium text-gray-700 mb-1">
            Up to sequence (exclusive)
          </label>
          <Input
            id="purge-sequence"
            type="number"
            min={1}
            value={value.sequence}
            errorMessage={error}
            placeholder="1000"
            onChange={(e) => onChange({ ...value, sequence: e.target.value })}
          />
        </div>
      )}

      {value.mode === 'keep' && (
        <div>
          <label htmlFor="purge-keep" className="block text-sm font-medium text-gray-700 mb-1">
            Messages to keep
          </label>
          <Input
            id="purge-keep"
            type="number"
            min={1}
            value={value.keep}
            errorMessage={error}
            placeholder="10"
            onChange={(e) => onChange({ ...value, keep: e.target.value })}
          />
        </div>
      )}
    </div>
  )
}
