import { useEffect, useRef, useState } from 'react'
import { Input, Button, CloseIcon, PlusIcon } from '@/components/ui'
import Tooltip from '@/components/common/Tooltip'
import { useRowKeys } from '@/hooks/useRowKeys'

interface Props {
  value: Record<string, string> | undefined
  onChange: (next: Record<string, string>) => void
  keyPlaceholder?: string
  valuePlaceholder?: string
  disabled?: boolean
}

function project(rows: Array<[string, string]>): Record<string, string> {
  const out: Record<string, string> = {}
  for (const [k, v] of rows) {
    if (k !== '') out[k] = v
  }
  return out
}

function sameRecord(a: Record<string, string>, b: Record<string, string>): boolean {
  const ak = Object.keys(a)
  return ak.length === Object.keys(b).length && ak.every((k) => a[k] === b[k])
}

function duplicateRows(rows: Array<[string, string]>): boolean[] {
  const seen = new Set<string>()
  return rows.map(([k]) => {
    if (k === '') return false
    if (seen.has(k)) return true
    seen.add(k)
    return false
  })
}

// KeyValueInput renders a list of key/value rows backed by an object. Rows live
// in local state so empty-key rows survive while typing; only non-empty keys are
// projected into the object passed to onChange.
export function KeyValueInput({
  value,
  onChange,
  keyPlaceholder = 'key',
  valuePlaceholder = 'value',
  disabled,
}: Props) {
  const [rows, setRows] = useState<Array<[string, string]>>(() => Object.entries(value ?? {}))
  const emitted = useRef<Record<string, string>>(value ?? {})
  const rowKeys = useRowKeys(rows.length)

  useEffect(() => {
    const incoming = value ?? {}
    if (!sameRecord(incoming, emitted.current)) {
      emitted.current = incoming
      rowKeys.reset()
      setRows(Object.entries(incoming))
    }
  }, [value, rowKeys])

  const commit = (next: Array<[string, string]>) => {
    setRows(next)
    const projected = project(next)
    emitted.current = projected
    onChange(projected)
  }

  const setKey = (i: number, k: string) => {
    const next = rows.slice()
    next[i] = [k, next[i]?.[1] ?? '']
    commit(next)
  }

  const setVal = (i: number, v: string) => {
    const next = rows.slice()
    next[i] = [next[i]?.[0] ?? '', v]
    commit(next)
  }

  const remove = (i: number) => {
    rowKeys.registerRemove(i)
    commit(rows.filter((_, idx) => idx !== i))
  }
  const add = () => {
    rowKeys.registerAdd()
    setRows((prev) => [...prev, ['', '']])
  }

  const isDuplicate = duplicateRows(rows)

  return (
    <div className="space-y-2">
      {rows.map(([k, v], i) => (
        <div key={rowKeys.keys[i]} className="flex gap-2">
          <Input
            value={k}
            onChange={(e) => setKey(i, e.target.value)}
            placeholder={keyPlaceholder}
            disabled={disabled}
            error={isDuplicate[i]}
            errorMessage={isDuplicate[i] ? 'Duplicate key — only the last value is kept' : undefined}
            className="flex-1"
          />
          <Input
            value={v}
            onChange={(e) => setVal(i, e.target.value)}
            placeholder={valuePlaceholder}
            disabled={disabled}
            className="flex-1"
          />
          {!disabled && (
            <Tooltip content="Remove">
              <Button variant="ghost" size="sm" onClick={() => remove(i)} aria-label="Remove">
                <CloseIcon className="w-4 h-4" />
              </Button>
            </Tooltip>
          )}
        </div>
      ))}
      {!disabled && (
        <Button variant="ghost" size="sm" onClick={add}>
          <PlusIcon className="w-4 h-4 mr-1" />
          Add entry
        </Button>
      )}
    </div>
  )
}
