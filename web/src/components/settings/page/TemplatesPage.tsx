import { useEffect, useMemo, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { toast } from '@/utils/toast'
import { plural } from '@/utils/plural'
import {
  Button,
  DataTable,
  DestructiveConfirm,
  EmptyState,
  RowActionButton,
  SearchInput,
  DocumentIcon,
  type DataTableColumn,
} from '@/components/ui'
import {
  useTemplates,
  useCreateTemplate,
  useUpdateTemplate,
  useDeleteTemplate,
  useBulkCreateTemplates,
  useDeleteAllTemplates,
  type MessageTemplate,
} from '@/contexts/templates'
import { TemplateEditModal, type TemplateValues } from '../templates/TemplateEditModal'
import { ImportTemplatesModal } from '../templates/ImportTemplatesModal'
import { exportToFile, exportToClipboard } from '../templates/templateImportExport'
import { SettingsPage } from './SettingsPage'

type ModalState =
  | { kind: 'closed' }
  | { kind: 'create'; initial: Partial<TemplateValues> }
  | { kind: 'edit'; id: string; initial: TemplateValues }
  | { kind: 'duplicate'; initial: TemplateValues }

const sortByUpdatedDesc = (a: MessageTemplate, b: MessageTemplate) => b.updatedAt - a.updatedAt

function formatRelative(ts: number): string {
  const diff = Date.now() - ts
  if (diff < 60_000) return 'just now'
  if (diff < 3_600_000) return `${Math.floor(diff / 60_000)}m ago`
  if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)}h ago`
  if (diff < 7 * 86_400_000) return `${Math.floor(diff / 86_400_000)}d ago`
  return new Date(ts).toLocaleDateString()
}

function headerCount(h?: Record<string, string>): number {
  return h ? Object.keys(h).length : 0
}

export default function TemplatesPage() {
  const { data: templates = [], isFetched: templatesLoaded } = useTemplates()
  const createMutation = useCreateTemplate()
  const updateMutation = useUpdateTemplate()
  const deleteMutation = useDeleteTemplate()
  const bulkCreateMutation = useBulkCreateTemplates()
  const deleteAllMutation = useDeleteAllTemplates()

  const [searchParams, setSearchParams] = useSearchParams()

  const [query, setQuery] = useState('')
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [modal, setModal] = useState<ModalState>({ kind: 'closed' })
  const [showImport, setShowImport] = useState(false)
  const [pendingDelete, setPendingDelete] = useState<
    | { kind: 'one'; id: string; name: string }
    | { kind: 'bulk'; ids: string[] }
    | { kind: 'all' }
    | null
  >(null)

  // Deep-link handling: ?edit=<id> opens edit, ?create=1 opens create.
  useEffect(() => {
    const editId = searchParams.get('edit')
    const create = searchParams.get('create')
    if (editId) {
      if (!templatesLoaded) return // wait for load, don't strip the param yet
      const t = templates.find((x) => x.id === editId)
      if (t) {
        setModal({
          kind: 'edit',
          id: t.id,
          initial: { name: t.name, subject: t.subject, messageType: t.messageType, data: t.data, headers: t.headers, wildcards: t.wildcards },
        })
      } else {
        toast.error('Template not found')
      }
      const next = new URLSearchParams(searchParams)
      next.delete('edit')
      setSearchParams(next, { replace: true })
    } else if (create) {
      setModal({ kind: 'create', initial: {} })
      const next = new URLSearchParams(searchParams)
      next.delete('create')
      setSearchParams(next, { replace: true })
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchParams, templates, templatesLoaded])

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase()
    const list = q
      ? templates.filter(
          (t) =>
            t.name.toLowerCase().includes(q) ||
            t.subject.toLowerCase().includes(q) ||
            t.messageType.toLowerCase().includes(q),
        )
      : [...templates]
    return list.sort(sortByUpdatedDesc)
  }, [templates, query])

  const allFilteredSelected = filtered.length > 0 && filtered.every((t) => selected.has(t.id))

  const toggleAll = () => {
    if (allFilteredSelected) setSelected(new Set())
    else setSelected(new Set(filtered.map((t) => t.id)))
  }

  const toggleOne = (id: string) => {
    const next = new Set(selected)
    if (next.has(id)) next.delete(id)
    else next.add(id)
    setSelected(next)
  }

  const closeModal = () => setModal({ kind: 'closed' })

  const handleSave = (values: TemplateValues) => {
    if (modal.kind === 'edit') {
      updateMutation.mutate(
        { id: modal.id, patch: values },
        {
          onSuccess: () => {
            toast.success(`Updated "${values.name}"`)
            closeModal()
          },
          onError: (e) => toast.error(`Update failed: ${e instanceof Error ? e.message : String(e)}`),
        },
      )
    } else {
      createMutation.mutate(values, {
        onSuccess: () => {
          toast.success(`Saved "${values.name}"`)
          closeModal()
        },
        onError: (e) => toast.error(`Save failed: ${e instanceof Error ? e.message : String(e)}`),
      })
    }
  }

  const handleDelete = (id: string, name: string) => {
    setPendingDelete({ kind: 'one', id, name })
  }

  const handleBulkDelete = () => {
    const ids = Array.from(selected)
    if (ids.length === 0) return
    setPendingDelete({ kind: 'bulk', ids })
  }

  const handleClearAll = () => {
    if (templates.length === 0) return
    setPendingDelete({ kind: 'all' })
  }

  const performPendingDelete = async () => {
    if (!pendingDelete) return
    if (pendingDelete.kind === 'one') {
      const { id, name } = pendingDelete
      deleteMutation.mutate(id, {
        onSuccess: () => {
          setSelected((s) => {
            const next = new Set(s)
            next.delete(id)
            return next
          })
          toast.success(`Deleted "${name}"`)
        },
        onError: (e) => toast.error(`Delete failed: ${e instanceof Error ? e.message : String(e)}`),
      })
      setPendingDelete(null)
    } else if (pendingDelete.kind === 'bulk') {
      const { ids } = pendingDelete
      // No backend bulk-delete RPC exists yet — issue concurrent deletes.
      // Tolerable for the cap of ~100 templates we expect users to have.
      const results = await Promise.allSettled(ids.map((id) => deleteMutation.mutateAsync(id)))
      const failedIds = ids.filter((_, i) => results[i].status === 'rejected')
      setSelected(new Set(failedIds))
      if (failedIds.length === 0) {
        toast.success(`Deleted ${plural(ids.length, 'template')}`)
      } else {
        const firstError = results.find((r): r is PromiseRejectedResult => r.status === 'rejected')?.reason
        toast.error(
          `Deleted ${ids.length - failedIds.length} of ${ids.length}, ${failedIds.length} failed: ` +
            `${firstError instanceof Error ? firstError.message : String(firstError)}`,
        )
      }
      setPendingDelete(null)
    } else {
      deleteAllMutation.mutate(undefined, {
        onSuccess: (n) => {
          setSelected(new Set())
          toast.success(`Cleared ${n} templates`)
        },
        onError: (e) => toast.error(`Clear failed: ${e instanceof Error ? e.message : String(e)}`),
      })
      setPendingDelete(null)
    }
  }

  const pendingDeleteDescription = (() => {
    if (!pendingDelete) return null
    if (pendingDelete.kind === 'one') return <span>Delete template <strong>{pendingDelete.name}</strong>?</span>
    if (pendingDelete.kind === 'bulk') {
      const n = pendingDelete.ids.length
      return <span>Delete {plural(n, 'template')}?</span>
    }
    return (
      <span>
        Delete <strong>ALL {templates.length}</strong> templates? This cannot be undone.
      </span>
    )
  })()

  const handleImport = (incoming: TemplateValues[]) => {
    bulkCreateMutation.mutate(incoming, {
      onSuccess: (n) => toast.success(`Imported ${plural(n, 'template')}`),
      onError: (e) => toast.error(`Import failed: ${e instanceof Error ? e.message : String(e)}`),
    })
  }

  const handleExportFile = () => {
    if (templates.length === 0) {
      toast.info('No templates to export')
      return
    }
    exportToFile(templates)
  }

  const handleExportClipboard = async () => {
    if (templates.length === 0) {
      toast.info('No templates to export')
      return
    }
    try {
      await exportToClipboard(templates)
      toast.success('Templates copied to clipboard')
    } catch {
      toast.error('Failed to copy')
    }
  }

  const total = templates.length
  const visible = filtered.length

  const columns: DataTableColumn<MessageTemplate>[] = [
    {
      key: 'select',
      width: 'w-10',
      header: (
        <input
          type="checkbox"
          checked={allFilteredSelected}
          onChange={toggleAll}
          aria-label="Select all"
        />
      ),
      render: (t) => (
        <input
          type="checkbox"
          checked={selected.has(t.id)}
          onChange={() => toggleOne(t.id)}
          aria-label={`Select ${t.name}`}
        />
      ),
    },
    {
      key: 'name',
      header: 'Name',
      render: (t) => <span className="font-medium text-gray-800">{t.name}</span>,
    },
    {
      key: 'subject',
      header: 'Subject',
      render: (t) => (
        <span className="block font-mono text-xs text-content-secondary truncate max-w-[200px]">
          {t.subject || '—'}
        </span>
      ),
    },
    {
      key: 'messageType',
      header: 'Message type',
      render: (t) => (
        <span className="block font-mono text-xs text-accent-text truncate max-w-[220px]">
          {t.messageType || '—'}
        </span>
      ),
    },
    {
      key: 'headers',
      header: 'Headers',
      render: (t) => <span className="text-xs text-content-tertiary">{headerCount(t.headers) || '—'}</span>,
    },
    {
      key: 'updated',
      header: 'Updated',
      render: (t) => (
        <span className="text-xs text-content-tertiary" title={new Date(t.updatedAt).toLocaleString()}>
          {formatRelative(t.updatedAt)}
        </span>
      ),
    },
  ]

  return (
    <SettingsPage
      title="Templates"
      description="Reusable publish payloads — saved subject, message type, JSON body, and headers"
      meta={`${plural(total, 'template')}${query ? ` · ${visible} matching` : ''}`}
      scroll="fill"
      actions={
        <>
          <Button variant="secondary" onClick={() => setShowImport(true)}>
            Import
          </Button>
          <Button variant="secondary" onClick={handleExportFile}>
            Export
          </Button>
          <Button variant="secondary" onClick={handleExportClipboard}>
            Copy
          </Button>
          <Button onClick={() => setModal({ kind: 'create', initial: {} })}>
            New template
          </Button>
        </>
      }
    >
      <div className="flex flex-col sm:flex-row sm:items-center gap-3 mb-4">
        <SearchInput
          value={query}
          onChange={setQuery}
          placeholder="Search by name, subject, or message type…"
          debounce={200}
          resultsCount={filtered.length}
          className="flex-1"
        />
        {selected.size > 0 && (
          <Button variant="danger" size="sm" onClick={handleBulkDelete} loading={deleteMutation.isPending}>
            Delete selected ({selected.size})
          </Button>
        )}
        {total > 0 && selected.size === 0 && (
          <Button variant="ghost" size="sm" onClick={handleClearAll} loading={deleteAllMutation.isPending}>
            Clear all
          </Button>
        )}
      </div>

      <div className="flex-1 overflow-auto rounded-lg border border-border bg-surface-primary">
        {total === 0 ? (
          <TemplatesEmptyState onCreate={() => setModal({ kind: 'create', initial: {} })} />
        ) : visible === 0 ? (
          <EmptyState
            title="No matching templates"
            description={`No templates match «${query}». Try a different search.`}
          />
        ) : (
          <DataTable
            className="min-w-[760px]"
            columns={columns}
            items={filtered}
            rowKey={(t) => t.id}
            rowActions={(t) => (
              <>
                <RowActionButton
                  kind="edit"
                  onClick={() =>
                    setModal({
                      kind: 'edit',
                      id: t.id,
                      initial: { name: t.name, subject: t.subject, messageType: t.messageType, data: t.data, headers: t.headers, wildcards: t.wildcards },
                    })
                  }
                  label={`Edit template ${t.name}`}
                />
                <RowActionButton
                  kind="duplicate"
                  onClick={() =>
                    setModal({
                      kind: 'duplicate',
                      initial: {
                        name: `${t.name} (copy)`,
                        subject: t.subject,
                        messageType: t.messageType,
                        data: t.data,
                        headers: t.headers,
                        wildcards: t.wildcards,
                      },
                    })
                  }
                  label={`Duplicate template ${t.name}`}
                />
                <RowActionButton
                  kind="delete"
                  onClick={() => handleDelete(t.id, t.name)}
                  label={`Delete template ${t.name}`}
                />
              </>
            )}
          />
        )}
      </div>

      <TemplateEditModal
        isOpen={modal.kind !== 'closed'}
        mode={modal.kind === 'closed' ? 'create' : modal.kind}
        initial={modal.kind === 'closed' ? {} : modal.initial}
        onClose={closeModal}
        onSave={handleSave}
        isSaving={createMutation.isPending || updateMutation.isPending}
      />

      <ImportTemplatesModal
        isOpen={showImport}
        onClose={() => setShowImport(false)}
        onImport={handleImport}
      />

      <DestructiveConfirm
        isOpen={pendingDelete !== null}
        title={pendingDelete?.kind === 'all' ? 'Delete all templates' : 'Delete template'}
        description={pendingDeleteDescription}
        confirmLabel={pendingDelete?.kind === 'all' ? 'Delete all' : 'Delete'}
        isPending={deleteMutation.isPending || deleteAllMutation.isPending}
        onCancel={() => setPendingDelete(null)}
        onConfirm={performPendingDelete}
      />
    </SettingsPage>
  )
}

function TemplatesEmptyState({ onCreate }: { onCreate: () => void }) {
  return (
    <EmptyState
      icon={<DocumentIcon className="w-full h-full" />}
      title="No templates yet"
      description="Save reusable payloads while publishing, or create one here."
      action={<Button onClick={onCreate}>New template</Button>}
    />
  )
}
