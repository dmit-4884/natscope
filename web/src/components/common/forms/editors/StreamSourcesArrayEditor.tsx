import { Button, PlusIcon } from '@/components/ui'
import type { StreamSource } from '@/types/management'
import { useRowKeys } from '@/hooks/useRowKeys'
import { StreamSourceEditor } from './StreamSourceEditor'

interface Props {
  value: StreamSource[] | undefined
  onChange: (next: StreamSource[] | undefined) => void
}

// StreamSourcesArrayEditor edits the optional Sources list — each entry is a
// StreamSourceEditor card; empty list collapses to undefined (omitted).
export function StreamSourcesArrayEditor({ value, onChange }: Props) {
  const items = value ?? []
  const rowKeys = useRowKeys(items.length)

  const setAt = (i: number, next: StreamSource | undefined) => {
    const out = items.slice()
    if (!next) {
      rowKeys.registerRemove(i)
      out.splice(i, 1)
    } else {
      out[i] = next
    }
    onChange(out.length === 0 ? undefined : out)
  }

  const remove = (i: number) => {
    rowKeys.registerRemove(i)
    const out = items.filter((_, idx) => idx !== i)
    onChange(out.length === 0 ? undefined : out)
  }

  const add = () => {
    rowKeys.registerAdd()
    onChange([...items, { name: '' }])
  }

  return (
    <div className="space-y-3">
      {items.map((src, i) => (
        <StreamSourceEditor
          key={rowKeys.keys[i]}
          value={src}
          onChange={(next) => setAt(i, next)}
          onRemove={() => remove(i)}
        />
      ))}
      <Button variant="ghost" size="sm" onClick={add}>
        <PlusIcon className="w-4 h-4 mr-1" />
        Add source
      </Button>
    </div>
  )
}
