import { useSourceRevisions } from '@/contexts/proto'
import { Dropdown } from '@/components/ui'
import { formatDateTime } from '@/utils/formatters'
import { plural } from '@/utils/plural'

interface Props {
  id?: string
  sourceId: string
  /** Fingerprint of the pinned schema; empty follows the active one. */
  value: string
  onChange: (fingerprint: string) => void
}

/** Picks the schema revision a mapping decodes with, or the source's active one. */
export function SchemaVersionPicker({ id, sourceId, value, onChange }: Props) {
  const { data: revisions = [], isLoading } = useSourceRevisions(sourceId || null)
  const pinnedGone = !!value && !isLoading && !revisions.some((r) => r.fingerprint === value)

  return (
    <div>
      <Dropdown
        value={value}
        onChange={onChange}
        id={id}
        disabled={!sourceId}
        options={[
          { value: '', label: 'Active schema (follows every refresh)' },
          ...revisions.map((r) => ({
            value: r.fingerprint,
            label: `${r.revision.slice(0, 12)} · ${plural(r.messageCount, 'type')} · ${formatDateTime(r.compiledAt)}${r.active ? ' · active' : ''}`,
          })),
          ...(pinnedGone ? [{ value, label: `${value.slice(0, 12)} · no longer stored` }] : []),
        ]}
      />
      {pinnedGone && (
        <p className="text-xs text-status-error-text mt-1">
          The pinned schema is gone, so this mapping cannot decode. Pick another revision or the active schema.
        </p>
      )}
    </div>
  )
}
