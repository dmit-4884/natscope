import { Input } from '@/components/ui'
import type { PlacementConfig } from '@/types/management'
import { StringArrayInput } from '../inputs/StringArrayInput'

interface Props {
  value: PlacementConfig | undefined
  onChange: (next: PlacementConfig | undefined) => void
}

// PlacementEditor pins a stream to a cluster and/or server tags (NATS
// scheduling hints). Empty cluster + no tags collapses to undefined.
export function PlacementEditor({ value, onChange }: Props) {
  const v = value ?? { cluster: '', tags: [] }

  const update = (patch: Partial<PlacementConfig>) => {
    const next = { ...v, ...patch }
    if (!next.cluster && (!next.tags || next.tags.length === 0)) {
      onChange(undefined)
      return
    }
    onChange(next)
  }

  return (
    <div className="grid grid-cols-1 gap-3">
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">Cluster</label>
        <Input
          value={v.cluster ?? ''}
          onChange={(e) => update({ cluster: e.target.value })}
          placeholder="us-east-1"
        />
      </div>
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">Tags</label>
        <StringArrayInput
          value={v.tags}
          onChange={(next) => update({ tags: next })}
          placeholder="ssd"
        />
        <p className="text-xs text-content-tertiary mt-1">Server tags the stream must be placed on.</p>
      </div>
    </div>
  )
}
