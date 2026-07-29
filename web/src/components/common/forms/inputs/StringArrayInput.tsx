import { Input, Button, CloseIcon, PlusIcon } from '@/components/ui'
import Tooltip from '@/components/common/Tooltip'
import { useRowKeys } from '@/hooks/useRowKeys'

interface Props {
  value: string[] | undefined
  onChange: (next: string[]) => void
  placeholder?: string
  disabled?: boolean
}

// StringArrayInput renders one Input per array element with add/remove buttons.
// Empty entries are allowed in-progress; callers should filter them on submit.
export function StringArrayInput({ value, onChange, placeholder, disabled }: Props) {
  const items = value ?? []
  const rowKeys = useRowKeys(items.length)

  const setAt = (i: number, next: string) => {
    const out = [...items]
    out[i] = next
    onChange(out)
  }

  const remove = (i: number) => {
    rowKeys.registerRemove(i)
    onChange(items.filter((_, idx) => idx !== i))
  }
  const add = () => {
    rowKeys.registerAdd()
    onChange([...items, ''])
  }

  return (
    <div className="space-y-2">
      {items.map((s, i) => (
        <div key={rowKeys.keys[i]} className="flex gap-2">
          <Input
            value={s}
            onChange={(e) => setAt(i, e.target.value)}
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
