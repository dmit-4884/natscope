import { useMemo, useState } from 'react'
import type { ProtoSource } from '@/api/protoSources'
import type { SchemaType } from '@/api/proto'
import { groupByPackage, shortTypeName, useSchemaTypes } from '@/contexts/proto'
import { EmptyState, QueryErrorState, SearchInput, Select, SkeletonRows, Toggle } from '@/components/ui'
import { plural } from '@/utils/plural'
import { cn } from '@/utils/cn'
import { KindBadge, SchemaTypeDetail } from './SchemaTypeDetail'

interface Props {
  sources: ProtoSource[]
}

export default function SchemaBrowser({ sources }: Props) {
  const [search, setSearch] = useState('')
  const [sourceFilter, setSourceFilter] = useState('')
  const [showImported, setShowImported] = useState(false)
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const { data: types = [], isLoading, error, refetch } = useSchemaTypes()

  const sourceNames = useMemo(() => new Map(sources.map((s) => [s.id, s.name])), [sources])
  const typeSources = useMemo(() => [...new Set(types.map((t) => t.sourceId))], [types])

  const visible = useMemo(() => {
    const needle = search.trim().toLowerCase()
    return types.filter(
      (t) =>
        (showImported || !t.dependency) &&
        (!sourceFilter || t.sourceId === sourceFilter) &&
        (!needle || t.fullName.toLowerCase().includes(needle) || t.comment.toLowerCase().includes(needle)),
    )
  }, [types, search, sourceFilter, showImported])

  const groups = useMemo(() => groupByPackage(visible), [visible])
  const selected = types.find((t) => t.id === selectedId) ?? null

  const openType = (sourceId: string, fullName: string) => {
    const target = types.find((t) => t.sourceId === sourceId && t.fullName === fullName)
    if (!target) return
    if (target.dependency) setShowImported(true)
    setSelectedId(target.id)
  }

  if (isLoading) return <SkeletonRows count={6} />
  if (error) return <QueryErrorState error={error} onRetry={() => void refetch()} />

  return (
    <div className="grid grid-cols-1 lg:grid-cols-[minmax(0,2fr)_minmax(0,3fr)] gap-4" data-testid="schema-browser">
      <div className="space-y-2 min-w-0">
        <SearchInput value={search} onChange={setSearch} placeholder="Search types or comments…" size="sm" debounce={0} clearable />
        <div className="flex flex-wrap items-center gap-3">
          {typeSources.length > 1 && (
            <Select
              size="sm"
              aria-label="Source"
              value={sourceFilter}
              onChange={(e) => setSourceFilter(e.target.value)}
              options={[
                { value: '', label: 'All sources' },
                ...typeSources.map((id) => ({ value: id, label: sourceNames.get(id) ?? id })),
              ]}
            />
          )}
          <span className="flex items-center gap-2 text-xs text-content-secondary">
            <Toggle checked={showImported} onChange={setShowImported} size="xs" label="Show imported types" testId="schema-show-imported" />
            Imported types
          </span>
          <span className="ml-auto text-xs text-content-tertiary tabular-nums">{plural(visible.length, 'type')}</span>
        </div>
        {groups.length === 0 ? (
          <EmptyState size="sm" title="No types match" description="Change the search or show imported types." />
        ) : (
          <div className="max-h-[32rem] overflow-y-auto border border-border rounded-md divide-y divide-gray-100">
            {groups.map(([pkg, list]) => (
              <div key={pkg} className="py-1">
                <div className="px-3 pt-1 pb-0.5 text-2xs font-semibold uppercase tracking-wide text-content-muted">{pkg}</div>
                <ul>
                  {list.map((t) => (
                    <li key={t.id}>
                      <TypeRow type={t} sourceName={typeSources.length > 1 ? sourceNames.get(t.sourceId) : undefined} active={t.id === selectedId} onSelect={() => setSelectedId(t.id)} />
                    </li>
                  ))}
                </ul>
              </div>
            ))}
          </div>
        )}
      </div>
      <div className="min-w-0">
        {selected ? (
          <SchemaTypeDetail
            type={selected}
            sourceName={sourceNames.get(selected.sourceId) ?? selected.sourceId}
            onOpenType={(fullName) => openType(selected.sourceId, fullName)}
          />
        ) : (
          <EmptyState size="sm" title="Pick a type" description="Fields, comments and an example payload show up here." />
        )}
      </div>
    </div>
  )
}

interface TypeRowProps {
  type: SchemaType
  sourceName?: string
  active: boolean
  onSelect: () => void
}

function TypeRow({ type, sourceName, active, onSelect }: TypeRowProps) {
  return (
    <button
      type="button"
      onClick={onSelect}
      aria-current={active ? 'true' : undefined}
      data-testid="schema-type"
      className={cn(
        'w-full text-left px-3 py-1.5 flex items-center gap-2 min-w-0 transition-colors',
        active ? 'bg-accent-muted' : 'hover:bg-surface-hover',
      )}
    >
      <KindBadge kind={type.kind} />
      <span className="font-mono text-xs text-content-primary shrink-0">{shortTypeName(type.fullName)}</span>
      {type.comment && <span className="text-xs text-content-tertiary truncate">{type.comment}</span>}
      {sourceName && <span className="ml-auto text-2xs text-content-muted shrink-0">{sourceName}</span>}
    </button>
  )
}
