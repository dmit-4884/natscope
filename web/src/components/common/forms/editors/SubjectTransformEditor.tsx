import { Input } from '@/components/ui'
import type { SubjectTransformConfig } from '@/types/management'

interface Props {
  value: SubjectTransformConfig | undefined
  onChange: (next: SubjectTransformConfig | undefined) => void
}

// SubjectTransformEditor edits the optional storage-time subject transform.
// Empty src+dest collapses to undefined (transform disabled).
export function SubjectTransformEditor({ value, onChange }: Props) {
  const v = value ?? { src: '', dest: '' }

  const update = (patch: Partial<SubjectTransformConfig>) => {
    const next = { ...v, ...patch }
    if (!next.src && !next.dest) {
      onChange(undefined)
      return
    }
    onChange(next)
  }

  return (
    <div className="grid grid-cols-1 gap-3">
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">Source pattern</label>
        <Input
          value={v.src}
          onChange={(e) => update({ src: e.target.value })}
          placeholder="orders.*"
        />
        <p className="text-xs text-content-tertiary mt-1">Subject pattern (supports wildcards).</p>
      </div>
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">Destination pattern</label>
        <Input
          value={v.dest}
          onChange={(e) => update({ dest: e.target.value })}
          placeholder="archived.orders.$1"
        />
        <p className="text-xs text-content-tertiary mt-1">May reference $1, $2 captures from source.</p>
      </div>
    </div>
  )
}
