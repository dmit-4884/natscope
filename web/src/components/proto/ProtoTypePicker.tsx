import { useMemo } from 'react'
import { SearchableSelect, type SearchableSelectOption } from '@/components/ui'
import { useProtoMessageEntities } from '@/contexts/proto'

interface Props {
  /** Source whose snapshot is queried for available types. */
  sourceId: string
  /** Currently selected fully-qualified message name. */
  value: string
  onChange: (messageType: string) => void
  className?: string
  placeholder?: string
  disabled?: boolean
}

/**
 * ProtoTypePicker — dropdown of proto message types within a single source.
 *
 * Source-scoped: pre-redesign pickers searched a global merged registry, which
 * silently picked the wrong schema when two sources declared the same FQN.
 * This component requires `sourceId` and only shows types from that source's
 * active snapshot.
 *
 * UX: combobox with substring filter and optgroups by package.
 */
export function ProtoTypePicker({
  sourceId,
  value,
  onChange,
  className,
  placeholder = 'Select type…',
  disabled,
}: Props) {
  const { messages, isLoading } = useProtoMessageEntities()

  const sourceMessages = useMemo(
    () => messages.filter((m) => m.sourceId === sourceId),
    [messages, sourceId],
  )

  const options = useMemo<SearchableSelectOption[]>(() => {
    const sorted = [...sourceMessages].sort((a, b) => {
      const pkgCmp = a.packageName.localeCompare(b.packageName)
      if (pkgCmp !== 0) return pkgCmp
      return a.shortName().localeCompare(b.shortName())
    })
    return sorted.map((m) => ({
      value: m.fullName,
      label: m.packageName ? `${m.packageName} · ${m.shortName()}` : m.shortName(),
    }))
  }, [sourceMessages])

  if (!sourceId) {
    return (
      <div className="px-3 py-2 text-xs text-amber-700 bg-status-warning-bg border border-amber-200 rounded-md">
        Pick a source first.
      </div>
    )
  }

  if (isLoading) {
    return (
      <div className="px-3 py-2 text-xs text-content-tertiary bg-surface-secondary border border-border rounded-md">
        Loading types…
      </div>
    )
  }

  if (sourceMessages.length === 0) {
    return (
      <div className="px-3 py-2 text-xs text-amber-700 bg-status-warning-bg border border-amber-200 rounded-md">
        No proto types in this source. Compile or pick another source.
      </div>
    )
  }

  return (
    <SearchableSelect
      options={options}
      value={value}
      onChange={onChange}
      placeholder={placeholder}
      searchPlaceholder="Filter types…"
      disabled={disabled}
      className={className}
    />
  )
}
