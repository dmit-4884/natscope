import { useEffect, useRef, useState } from 'react'
import { Input, Button, CloseIcon, PlusIcon } from '@/components/ui'
import Tooltip from '@/components/common/Tooltip'
import { useRowKeys } from '@/hooks/useRowKeys'

interface Props {
  value: number[] | undefined
  onChange: (next: number[]) => void
  placeholder?: string
  disabled?: boolean
}

function coerce(raw: string): number {
  const n = Number(raw)
  return raw.trim() !== '' && Number.isFinite(n) ? n : 0
}

function sameNumbers(a: number[], b: number[]): boolean {
  return a.length === b.length && a.every((n, i) => n === b[i])
}

// NumberArrayInput renders one numeric Input per array element (e.g.
// Consumer.backoff). The raw text is kept while editing so clearing a field
// does not write 0; values coerce to numbers on blur, add, and remove.
export function NumberArrayInput({ value, onChange, placeholder, disabled }: Props) {
  const items = value ?? []
  const [drafts, setDrafts] = useState<string[]>(() => items.map((n) => String(n)))
  const emitted = useRef<number[]>(items)
  const rowKeys = useRowKeys(drafts.length)

  useEffect(() => {
    const next = value ?? []
    if (!sameNumbers(next, emitted.current)) {
      emitted.current = next
      rowKeys.reset()
      setDrafts(next.map((n) => String(n)))
    }
  }, [value, rowKeys])

  const commit = (nextDrafts: string[]) => {
    const nums = nextDrafts.map(coerce)
    setDrafts(nums.map((n) => String(n)))
    emitted.current = nums
    onChange(nums)
  }

  const setAt = (i: number, raw: string) => {
    const next = drafts.slice()
    next[i] = raw
    setDrafts(next)
  }

  const remove = (i: number) => {
    rowKeys.registerRemove(i)
    commit(drafts.filter((_, idx) => idx !== i))
  }
  const add = () => {
    rowKeys.registerAdd()
    commit([...drafts, '0'])
  }

  return (
    <div className="space-y-2">
      {drafts.map((raw, i) => (
        <div key={rowKeys.keys[i]} className="flex gap-2">
          <Input
            type="number"
            value={raw}
            onChange={(e) => setAt(i, e.target.value)}
            onBlur={() => commit(drafts)}
            placeholder={placeholder}
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
          Add
        </Button>
      )}
    </div>
  )
}
