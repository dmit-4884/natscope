import { Input, Button } from '@/components/ui'
import Tooltip from '@/components/common/Tooltip'
import { parseIntOr } from '@/utils/numbers'
import type { StreamSource } from '@/types/management'

interface Props {
  value: StreamSource | undefined
  onChange: (next: StreamSource | undefined) => void
  /** Hides the "remove" button when used inside an array editor that owns deletion. */
  hideRemove?: boolean
  onRemove?: () => void
}

// StreamSourceEditor edits a single Mirror or Sources entry. Empty name
// collapses to undefined; external/transforms are JSON-only (rare, omitted here).
export function StreamSourceEditor({ value, onChange, hideRemove, onRemove }: Props) {
  const v = value ?? { name: '', opt_start_seq: 0, filter_subject: '' }

  const update = (patch: Partial<StreamSource>) => {
    const next: StreamSource = { ...v, ...patch }
    if (!next.name) {
      onChange(undefined)
      return
    }
    onChange(next)
  }

  return (
    <div className="space-y-3 p-3 border rounded-md bg-surface-primary">
      <div className="grid grid-cols-2 gap-3">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Source stream name</label>
          <Input
            value={v.name}
            onChange={(e) => update({ name: e.target.value })}
            placeholder="ORDERS"
          />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Opt start seq</label>
          <Input
            type="number"
            value={v.opt_start_seq ?? 0}
            onChange={(e) => update({ opt_start_seq: parseIntOr(e.target.value, 0) })}
            placeholder="0"
          />
        </div>
      </div>
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">Filter subject</label>
        <Input
          value={v.filter_subject ?? ''}
          onChange={(e) => update({ filter_subject: e.target.value })}
          placeholder="orders.created"
        />
        <p className="text-xs text-content-tertiary mt-1">
          Optional filter on the source stream (advanced fields like external/transforms
          are JSON-only).
        </p>
      </div>
      {!hideRemove && onRemove && (
        <div className="flex justify-end">
          <Tooltip content="Remove">
            <Button variant="ghost" size="sm" onClick={onRemove}>
              Remove
            </Button>
          </Tooltip>
        </div>
      )}
    </div>
  )
}
