import { useEffect, useRef, useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { getErrorMessage } from '@/api/errors'
import { toast } from '@/utils/toast'
import { DestructiveConfirm, ChevronDownIcon, PencilIcon, TrashIcon, ClipboardIcon } from '@/components/ui'
import Tooltip from '@/components/common/Tooltip'
import {
  useSearchTemplates,
  useCreateTemplate,
  useDeleteTemplate,
  useTemplates,
  type MessageTemplate,
} from '@/contexts/templates'
import { rememberSettingsReturn } from '@/components/settings/page/settingsNav'
import { TemplateEditModal, type TemplateValues } from '@/components/settings/templates/TemplateEditModal'

interface Props {
  /** Current subject pattern (used as prefill + contextual sorting). */
  subjectPattern: string
  /** Current resolved message type (used as prefill + contextual sorting). */
  messageType: string
  /** Current JSON payload — saved with the template. */
  messageJson: string
  /** Current headers map — saved with the template. */
  headers: Record<string, string>
  /** Current wildcard slot values — saved with the template. */
  wildcards: string[]
  /**
   * Callback when a template is loaded. Replaces editor's subject, payload,
   * headers, and wildcards.
   */
  onLoad: (values: {
    name: string
    subject: string
    data: string
    headers: Record<string, string>
    wildcards: string[]
  }) => void
}

/**
 * Quick save/load picker for publish payload templates (backend-stored); full
 * management at /settings/templates.
 */
export function TemplateMenu({ subjectPattern, messageType, messageJson, headers, wildcards, onLoad }: Props) {
  const navigate = useNavigate()
  const location = useLocation()
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState('')
  const [showSaveModal, setShowSaveModal] = useState(false)
  const [pendingDelete, setPendingDelete] = useState<MessageTemplate | null>(null)
  const containerRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    const onMouseDown = (e: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false)
        setQuery('')
      }
    }
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        setOpen(false)
        setQuery('')
      }
    }
    document.addEventListener('mousedown', onMouseDown)
    document.addEventListener('keydown', onKeyDown)
    return () => {
      document.removeEventListener('mousedown', onMouseDown)
      document.removeEventListener('keydown', onKeyDown)
    }
  }, [open])

  const { data: templates = [] } = useTemplates()
  const result = useSearchTemplates(query, {
    subject: subjectPattern || undefined,
    messageType: messageType || undefined,
  })
  const createMutation = useCreateTemplate()
  const deleteMutation = useDeleteTemplate()

  const total = templates.length
  const totalMatching = result.contextual.length + result.rest.length

  const handleSaveSubmit = (values: TemplateValues) => {
    createMutation.mutate(values, {
      onSuccess: () => {
        setShowSaveModal(false)
        toast.success(`Template "${values.name}" saved`)
      },
      onError: (e) => toast.error(`Failed to save: ${getErrorMessage(e)}`),
    })
  }

  const handleLoad = (template: MessageTemplate) => {
    // Form-state owner shows the success toast (with Undo); only it can
    // snapshot/restore the draft.
    onLoad({
      name: template.name,
      subject: template.subject,
      data: template.data,
      headers: template.headers ?? {},
      wildcards: template.wildcards ?? [],
    })
    setOpen(false)
    setQuery('')
  }

  const handleDelete = (e: React.MouseEvent, t: MessageTemplate) => {
    e.stopPropagation()
    setPendingDelete(t)
  }

  const performDelete = () => {
    const t = pendingDelete
    if (!t) return
    deleteMutation.mutate(t.id, {
      onSuccess: () => toast.success(`Deleted template "${t.name}"`),
      onError: (err) => toast.error(`Failed to delete: ${getErrorMessage(err)}`),
    })
    setPendingDelete(null)
  }

  const handleEdit = (e: React.MouseEvent, t: MessageTemplate) => {
    e.stopPropagation()
    setOpen(false)
    rememberSettingsReturn(location.pathname + location.search)
    navigate(`/settings/templates?edit=${encodeURIComponent(t.id)}`)
  }

  const goToManage = () => {
    setOpen(false)
    rememberSettingsReturn(location.pathname + location.search)
    navigate('/settings/templates')
  }

  const headersCount = Object.keys(headers).length

  return (
    <div ref={containerRef} className="relative flex items-center gap-1.5">
      <Tooltip content="Load a saved template into the form">
        <button
          onClick={() => {
            setOpen((v) => !v)
            if (open) setQuery('')
          }}
          className="flex items-center gap-1.5 px-2.5 py-1 text-xs font-medium text-content-secondary bg-surface-primary border border-border-strong rounded-md hover:bg-surface-secondary hover:text-content-primary transition-colors"
          data-testid="templates-trigger"
        >
          <ClipboardIcon className="w-3.5 h-3.5" />
          Templates{total > 0 ? ` (${total})` : ''}
          <ChevronDownIcon className={`w-3 h-3 transition-transform ${open ? 'rotate-180' : ''}`} />
        </button>
      </Tooltip>
      <Tooltip content="Save current payload as a reusable template">
        <button
          onClick={() => setShowSaveModal(true)}
          className="px-2.5 py-1 text-xs font-medium text-content-secondary bg-surface-primary border border-border-strong rounded-md hover:bg-surface-secondary hover:text-content-primary transition-colors"
          data-testid="save-template"
        >
          Save as template
        </button>
      </Tooltip>

      {open && (
        <div className="absolute top-full mt-1 right-0 w-80 bg-surface-primary border border-border rounded-lg shadow-lg z-30 max-h-96 overflow-hidden flex flex-col">
          <div className="p-2 border-b border-gray-100 bg-surface-secondary">
            <input
              type="text"
              autoFocus
              placeholder="Search by name, subject or type…"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              className="w-full px-2 py-1 text-xs border border-border-strong rounded focus:ring-border-focus focus:border-border-focus"
            />
          </div>

          <div className="flex-1 overflow-auto">
            {total === 0 && (
              <div className="px-3 py-6 text-xs text-content-muted text-center">
                No templates yet. Hit &ldquo;Save as template&rdquo; to create one.
              </div>
            )}

            {total > 0 && totalMatching === 0 && (
              <div className="px-3 py-6 text-xs text-content-muted text-center">
                No templates match &ldquo;{query}&rdquo;.
              </div>
            )}

            {result.contextual.length > 0 && <SectionHeader>Suggested for current context</SectionHeader>}
            {result.contextual.map((t) => (
              <TemplateRow key={t.id} t={t} onLoad={handleLoad} onEdit={handleEdit} onDelete={handleDelete} />
            ))}

            {result.rest.length > 0 && result.contextual.length > 0 && <SectionHeader>All templates</SectionHeader>}
            {result.rest.map((t) => (
              <TemplateRow key={t.id} t={t} onLoad={handleLoad} onEdit={handleEdit} onDelete={handleDelete} />
            ))}
          </div>

          <button
            onClick={goToManage}
            className="border-t border-gray-100 px-3 py-2 text-xs text-accent hover:bg-accent-light text-left"
          >
            Manage all in Settings →
          </button>
        </div>
      )}

      <TemplateEditModal
        isOpen={showSaveModal}
        mode="create"
        initial={{
          name: '',
          subject: subjectPattern,
          messageType,
          data: messageJson,
          headers: headersCount > 0 ? headers : undefined,
          wildcards: wildcards.length > 0 ? wildcards : undefined,
        }}
        onClose={() => setShowSaveModal(false)}
        onSave={handleSaveSubmit}
      />

      <DestructiveConfirm
        isOpen={pendingDelete !== null}
        title="Delete template"
        description={
          pendingDelete ? (
            <span>Delete template <strong>{pendingDelete.name}</strong>?</span>
          ) : null
        }
        confirmLabel="Delete"
        isPending={deleteMutation.isPending}
        onCancel={() => setPendingDelete(null)}
        onConfirm={performDelete}
      />
    </div>
  )
}

function SectionHeader({ children }: { children: React.ReactNode }) {
  return (
    <div className="px-3 py-1.5 text-2xs uppercase tracking-wide text-content-muted bg-surface-secondary border-b border-gray-100 sticky top-0">
      {children}
    </div>
  )
}

function TemplateRow({
  t,
  onLoad,
  onEdit,
  onDelete,
}: {
  t: MessageTemplate
  onLoad: (t: MessageTemplate) => void
  onEdit: (e: React.MouseEvent, t: MessageTemplate) => void
  onDelete: (e: React.MouseEvent, t: MessageTemplate) => void
}) {
  return (
    <div className="group flex items-start gap-2 px-3 py-2 hover:bg-accent-light border-b border-gray-100 last:border-b-0">
      <button
        type="button"
        className="flex-1 min-w-0 text-left cursor-pointer"
        onClick={() => onLoad(t)}
      >
        <div className="font-medium text-sm text-gray-700 truncate">{t.name}</div>
        <div className="text-xs text-content-tertiary font-mono truncate">{t.subject || '—'}</div>
        {t.messageType && <div className="text-2xs text-accent font-mono truncate">{t.messageType}</div>}
      </button>
      <div className="flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
        <Tooltip content="Edit in Settings">
          <button
            onClick={(e) => onEdit(e, t)}
            className="text-content-muted hover:text-accent p-1"
            aria-label={`Edit ${t.name}`}
          >
            <PencilIcon className="w-3.5 h-3.5" />
          </button>
        </Tooltip>
        <Tooltip content="Delete template">
          <button
            onClick={(e) => onDelete(e, t)}
            className="text-content-muted hover:text-status-error-text p-1"
            aria-label={`Delete ${t.name}`}
          >
            <TrashIcon className="w-3.5 h-3.5" />
          </button>
        </Tooltip>
      </div>
    </div>
  )
}
