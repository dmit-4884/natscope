import { Input } from '@/components/ui'
import type { RePublishConfig } from '@/types/management'

interface Props {
  value: RePublishConfig | undefined
  onChange: (next: RePublishConfig | undefined) => void
}

// RePublishEditor edits the optional re-publish config that mirrors each
// message to another subject. Empty src+dest collapses to undefined (omitted = disabled).
export function RePublishEditor({ value, onChange }: Props) {
  const v = value ?? { src: '', dest: '', headers_only: false }

  const update = (patch: Partial<RePublishConfig>) => {
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
        <label className="block text-sm font-medium text-gray-700 mb-1">Source subject</label>
        <Input
          value={v.src}
          onChange={(e) => update({ src: e.target.value })}
          placeholder="orders.>"
        />
        <p className="text-xs text-content-tertiary mt-1">Pattern matched against incoming subjects.</p>
      </div>
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">Destination subject</label>
        <Input
          value={v.dest}
          onChange={(e) => update({ dest: e.target.value })}
          placeholder="audit.orders.>"
        />
        <p className="text-xs text-content-tertiary mt-1">May reference $1, $2 wildcards captured from source.</p>
      </div>
      <label className="flex items-center gap-2">
        <input
          type="checkbox"
          checked={v.headers_only ?? false}
          onChange={(e) => update({ headers_only: e.target.checked })}
          className="rounded border-border-strong"
        />
        <span className="text-sm text-gray-700">Headers only (omit payload)</span>
      </label>
    </div>
  )
}
