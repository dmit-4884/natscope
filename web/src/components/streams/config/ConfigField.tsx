import { useId } from 'react'
import { useRowKeys } from '@/hooks/useRowKeys'
import { Input, Select, Badge, CloseIcon, LockClosedIcon } from '@/components/ui'
import { cn } from '@/utils/cn'
import type { StreamFieldDef } from './streamFieldDefinitions'

export type ConfigFieldMode = 'create' | 'edit'

interface Props {
  def: StreamFieldDef
  value: unknown
  onChange: (next: unknown) => void
  mode: ConfigFieldMode
  /** When set, the connected server doesn't support this field — control is disabled and the reason shown. */
  unsupportedReason?: string
  lockedReason?: string
}

export function ConfigField({ def, value, onChange, mode, unsupportedReason, lockedReason }: Props) {
  const isUnsupported = !!unsupportedReason
  const isLocked = (mode === 'edit' && !def.editableOnUpdate) || isUnsupported || !!lockedReason
  const labelId = useId()

  return (
    <div>
      <div className="flex items-center gap-2 mb-1">
        <label id={labelId} className="block text-sm font-medium text-gray-700">
          {def.label}
        </label>
        {isUnsupported ? (
          <Badge variant="warning" size="sm" aria-label="Not supported by the connected server">
            Unsupported
          </Badge>
        ) : isLocked ? (
          <Badge variant="warning" size="sm" aria-label="Read-only field">
            <LockClosedIcon className="w-3 h-3 mr-1" />
            Read-only
          </Badge>
        ) : null}
      </div>

      <FieldControl def={def} value={value} onChange={onChange} isLocked={isLocked} labelId={labelId} />

      {isUnsupported ? (
        <p className="text-xs text-status-warning-text mt-1">{unsupportedReason}</p>
      ) : lockedReason ? (
        <p className="text-xs text-status-warning-text mt-1">{lockedReason}</p>
      ) : (isLocked && def.immutableReason) ? (
        <p className="text-xs text-content-tertiary mt-1">{def.immutableReason}</p>
      ) : def.helperText ? (
        <p className="text-xs text-content-tertiary mt-1">{def.helperText}</p>
      ) : null}
    </div>
  )
}

interface ControlProps {
  def: StreamFieldDef
  value: unknown
  onChange: (next: unknown) => void
  isLocked: boolean
  labelId: string
}

function FieldControl({ def, value, onChange, isLocked, labelId }: ControlProps) {
  switch (def.type) {
    case 'text':
      return (
        <Input
          value={value == null ? '' : String(value)}
          readOnly={isLocked}
          placeholder={def.placeholder}
          onChange={isLocked ? undefined : (e) => onChange(e.target.value)}
          aria-labelledby={labelId}
        />
      )

    case 'number':
    case 'duration_ns': {
      const fallback = (def.defaultValue as number | undefined) ?? 0
      return (
        <Input
          type="number"
          value={value == null ? fallback : Number(value)}
          readOnly={isLocked}
          onChange={
            isLocked
              ? undefined
              : (e) => {
                  const raw = e.target.value
                  if (raw === '' || raw === '-') {
                    onChange(fallback)
                    return
                  }
                  const n = parseInt(raw, 10)
                  onChange(Number.isNaN(n) ? fallback : n)
                }
          }
          aria-labelledby={labelId}
        />
      )
    }

    case 'boolean':
      // Checkbox ignores readOnly; use disabled when locked but keep label text
      // selectable.
      return (
        <label className="inline-flex items-center gap-2">
          <input
            type="checkbox"
            checked={!!value}
            disabled={isLocked}
            onChange={isLocked ? undefined : (e) => onChange(e.target.checked)}
            className={cn(
              'rounded border-border-strong',
              isLocked && 'cursor-not-allowed'
            )}
            aria-labelledby={labelId}
          />
          <span className="text-sm text-gray-700 select-text">
            {value ? 'Enabled' : 'Disabled'}
          </span>
        </label>
      )

    case 'select': {
      const opts = def.options ?? []
      const stringValue = value == null ? '' : String(value)
      if (isLocked) {
        // Resolved label as readOnly input so it's copyable (disabled <select>
        // blocks copying).
        const label = opts.find((o) => o.value === stringValue)?.label ?? stringValue
        return <Input value={label} readOnly aria-labelledby={labelId} />
      }
      return (
        <Select
          value={stringValue || ((def.defaultValue as string | undefined) ?? '')}
          options={opts as { value: string; label: string }[]}
          onChange={(e) => onChange(e.target.value)}
          aria-labelledby={labelId}
        />
      )
    }

    case 'subjects':
      return (
        <SubjectsInput
          value={Array.isArray(value) ? (value as string[]) : []}
          readOnly={isLocked}
          onChange={onChange as (v: string[]) => void}
        />
      )
  }
}

interface SubjectsInputProps {
  value: string[]
  readOnly: boolean
  onChange: (v: string[]) => void
}

function SubjectsInput({ value, readOnly, onChange }: SubjectsInputProps) {
  const list = value.length > 0 ? value : ['']
  const rowKeys = useRowKeys(list.length)

  const update = (idx: number, next: string) => {
    const copy = [...list]
    copy[idx] = next
    onChange(copy)
  }
  const add = () => {
    rowKeys.registerAdd()
    onChange([...list, ''])
  }
  const remove = (idx: number) => {
    rowKeys.registerRemove(idx)
    const next = list.filter((_, i) => i !== idx)
    onChange(next.length > 0 ? next : [''])
  }

  return (
    <div className="space-y-2">
      {list.map((subject, index) => (
        <div key={rowKeys.keys[index]} className="flex gap-2 items-center">
          <div className="flex-1 min-w-0">
            <Input
              value={subject}
              readOnly={readOnly}
              placeholder="orders.>"
              onChange={readOnly ? undefined : (e) => update(index, e.target.value)}
            />
          </div>
          {!readOnly && list.length > 1 && (
            <button
              type="button"
              onClick={() => remove(index)}
              className="p-1.5 text-red-500 hover:text-red-700 hover:bg-status-error-bg rounded shrink-0"
              aria-label={`Remove subject ${index + 1}`}
            >
              <CloseIcon className="w-5 h-5" />
            </button>
          )}
        </div>
      ))}
      {!readOnly && (
        <button type="button" onClick={add} className="text-sm text-accent hover:text-blue-800">
          + Add Subject
        </button>
      )}
    </div>
  )
}
