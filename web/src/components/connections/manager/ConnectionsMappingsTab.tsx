import { useMemo, useState, useEffect } from 'react'
import { Button, Dropdown, SearchInput, DestructiveConfirm, PlusIcon } from '@/components/ui'
import { toast } from '@/utils/toast'
import { useStreamEntities } from '@/contexts/streams'
import {
  useCreateMapping,
  useDeleteMapping,
  useMappingItems,
  useMappingHealthBatch,
  useUpdateMapping,
} from '@/contexts/mappings'
import type { MappingItem } from '@/api/mappings'
import { useProtoSources } from '@/contexts/proto'
import { MappingAddForm } from '@/components/mappings/MappingAddForm'
import { MappingTable } from '@/components/mappings/MappingTable'

interface Props {
  connectionId: string | null
  initialSubjectPattern?: string | null
}

/**
 * Mappings tab inside the connection manager. Source-aware: every mutation
 * carries sourceId; rows show source/health badges so the user can see why
 * a mapping isn't decoding.
 */
export function ConnectionsMappingsTab({ connectionId, initialSubjectPattern }: Props) {
  const [newPattern, setNewPattern] = useState('')
  const [newProtoType, setNewProtoType] = useState('')
  const [newSourceId, setNewSourceId] = useState('')
  const [searchFilter, setSearchFilter] = useState('')
  const [sourceFilter, setSourceFilter] = useState('')
  const [showAddForm, setShowAddForm] = useState(false)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [pendingDelete, setPendingDelete] = useState<MappingItem | null>(null)

  const createMappingMutation = useCreateMapping()
  const deleteMappingMutation = useDeleteMapping()
  const updateMappingMutation = useUpdateMapping()
  const { data: items = [] } = useMappingItems()
  const { data: sources = [] } = useProtoSources()
  const { streams } = useStreamEntities(connectionId)

  useEffect(() => {
    if (initialSubjectPattern) {
      setShowAddForm(true)
      setNewPattern(initialSubjectPattern)
    }
  }, [initialSubjectPattern])

  const availablePatterns = useMemo(() => {
    if (!streams || streams.length === 0) return []
    const patterns = new Set<string>()
    for (const stream of streams) {
      for (const subject of stream.subjects) patterns.add(subject)
    }
    return Array.from(patterns).sort()
  }, [streams])

  const unmappedPatterns = useMemo(() => {
    const taken = new Set<string>()
    for (const m of items) taken.add(m.pattern)
    return availablePatterns.filter((p) => !taken.has(p))
  }, [availablePatterns, items])

  const visibleItems = useMemo(() => {
    let result = items
    if (sourceFilter) result = result.filter((m) => m.sourceId === sourceFilter)
    if (searchFilter) {
      const lower = searchFilter.toLowerCase()
      result = result.filter(
        (m) =>
          m.pattern.toLowerCase().includes(lower) ||
          m.messageType.toLowerCase().includes(lower) ||
          m.sourceId.toLowerCase().includes(lower),
      )
    }
    return result
  }, [items, searchFilter, sourceFilter])

  const visibleIds = useMemo(() => visibleItems.map((m) => m.id), [visibleItems])
  const { data: healths } = useMappingHealthBatch(visibleIds)
  const healthById = useMemo(() => {
    const out: Record<string, NonNullable<typeof healths>[number]> = {}
    if (healths) for (const h of healths) out[h.id] = h
    return out
  }, [healths])

  const sourceNamesById = useMemo(() => {
    const out: Record<string, string> = {}
    for (const s of sources) out[s.id] = s.name
    return out
  }, [sources])

  const resetForm = () => {
    setNewPattern('')
    setNewProtoType('')
    setNewSourceId('')
    setShowAddForm(false)
    setEditingId(null)
  }

  const handleAdd = async () => {
    if (!newPattern || !newProtoType || !newSourceId) return
    try {
      if (editingId) {
        await updateMappingMutation.mutateAsync({
          id: editingId,
          pattern: newPattern,
          messageType: newProtoType,
          sourceId: newSourceId,
        })
        toast.success('Mapping updated')
      } else {
        await createMappingMutation.mutateAsync({
          pattern: newPattern,
          messageType: newProtoType,
          sourceId: newSourceId,
        })
      }
      resetForm()
    } catch { /* toasted by the global mutation handler */ }
  }

  const handleEdit = (item: MappingItem) => {
    setEditingId(item.id)
    setNewPattern(item.pattern)
    setNewProtoType(item.messageType)
    setNewSourceId(item.sourceId)
    setShowAddForm(true)
  }

  const handleDelete = (id: string) => {
    const item = items.find((m) => m.id === id)
    if (item) setPendingDelete(item)
  }

  const performDelete = async () => {
    if (!pendingDelete) return
    try {
      if (pendingDelete.id === editingId) resetForm()
      await deleteMappingMutation.mutateAsync(pendingDelete.id)
      toast.success('Mapping deleted')
    } catch {
      /* toasted by the global mutation error handler */
    } finally {
      setPendingDelete(null)
    }
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex flex-col sm:flex-row gap-3 mb-4">
        <SearchInput
          value={searchFilter}
          onChange={setSearchFilter}
          placeholder="Search by pattern, type, or source…"
          debounce={200}
          resultsCount={visibleItems.length}
          className="flex-1"
        />
        <div className="flex gap-3">
          <Dropdown
            className="min-w-[180px] flex-1 sm:flex-none"
            value={sourceFilter}
            onChange={(v) => setSourceFilter(v)}
            options={[
              { value: '', label: 'All sources' },
              ...sources.map((s) => ({ value: s.id, label: s.name })),
            ]}
          />
          <Button
            onClick={() => {
              if (editingId) {
                resetForm()
              } else {
                setShowAddForm((v) => !v)
              }
            }}
            icon={<PlusIcon />}
          >
            Add mapping
          </Button>
        </div>
      </div>

      {showAddForm && (
        <MappingAddForm
          mode={editingId ? 'edit' : 'create'}
          pattern={newPattern}
          onPatternChange={setNewPattern}
          protoType={newProtoType}
          onProtoTypeChange={setNewProtoType}
          sourceId={newSourceId}
          onSourceIdChange={setNewSourceId}
          unmappedPatterns={unmappedPatterns}
          isSubmitting={createMappingMutation.isPending || updateMappingMutation.isPending}
          onSubmit={handleAdd}
          onCancel={resetForm}
        />
      )}

      <div className="flex-1 overflow-auto rounded-lg border border-border bg-surface-primary">
        <MappingTable
          items={visibleItems}
          healthById={healthById}
          sourceNamesById={sourceNamesById}
          onEdit={handleEdit}
          onDelete={handleDelete}
          onAdd={showAddForm ? undefined : () => setShowAddForm(true)}
          searchFilter={searchFilter}
        />
      </div>

      <DestructiveConfirm
        isOpen={pendingDelete !== null}
        title="Delete mapping"
        description={
          <span>
            Delete the mapping for <strong className="font-mono">{pendingDelete?.pattern}</strong>? Messages on this
            subject will no longer be decoded as <span className="font-mono">{pendingDelete?.messageType}</span>.
          </span>
        }
        isPending={deleteMappingMutation.isPending}
        onCancel={() => setPendingDelete(null)}
        onConfirm={performDelete}
      />
    </div>
  )
}
