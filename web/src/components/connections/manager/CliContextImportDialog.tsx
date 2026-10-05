import { useEffect, useMemo, useRef, useState } from 'react'
import { Alert, Button, EmptyState, Modal, QueryErrorState, SkeletonRows, UploadIcon, WarningIcon } from '@/components/ui'
import { getErrorMessage, getErrorReason } from '@/api/errors'
import type { CliContextFile, CliContextSummary } from '@/api/connections'
import { useCliContexts, useImportCliContexts } from '@/contexts/connection'
import { toast } from '@/utils/toast'
import { plural } from '@/utils/plural'
import { AUTH_LABELS } from './connectionFormData'

interface Props {
  isOpen: boolean
  onClose: () => void
}

const HOST_DISABLED = 'CLI_CONTEXTS_HOST_DISABLED'

const selectable = (c: CliContextSummary) => c.importable && !c.exists

async function readFiles(list: FileList): Promise<CliContextFile[]> {
  return Promise.all(Array.from(list, async (f) => ({ name: f.name, content: new Uint8Array(await f.arrayBuffer()) })))
}

export function CliContextImportDialog({ isOpen, onClose }: Props) {
  const [upload, setUpload] = useState<{ id: number; files: CliContextFile[] }>({ id: 0, files: [] })
  const [chosen, setChosen] = useState<Set<string>>(new Set())
  const fileInputRef = useRef<HTMLInputElement>(null)
  const { data, isLoading, error, refetch } = useCliContexts(upload, isOpen)
  const importMutation = useImportCliContexts()

  const contexts = useMemo(() => data?.contexts ?? [], [data])
  useEffect(() => {
    setChosen(new Set(contexts.filter(selectable).map((c) => c.name)))
  }, [contexts])

  const toggle = (name: string) =>
    setChosen((prev) => {
      const next = new Set(prev)
      if (next.has(name)) next.delete(name)
      else next.add(name)
      return next
    })

  const handleFiles = async (list: FileList | null) => {
    if (!list || list.length === 0) return
    const files = await readFiles(list)
    setUpload((prev) => ({ id: prev.id + 1, files }))
  }

  const handleImport = async () => {
    const names = contexts.filter((c) => chosen.has(c.name)).map((c) => c.name)
    try {
      const res = await importMutation.mutateAsync({ names, files: upload.files })
      if (res.created.length > 0) toast.success(`Imported ${plural(res.created.length, 'connection')}`)
      for (const s of res.skipped) toast.warning(`Skipped ${s.name}: ${s.reason}`)
      onClose()
    } catch {
      /* toasted by the mutation hooks */
    }
  }

  const uploaded = upload.files.length > 0

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title="Import from nats CLI"
      size="lg"
      footer={
        <div className="flex w-full items-center justify-between gap-2">
          <Button variant="ghost" onClick={() => fileInputRef.current?.click()} icon={<UploadIcon />}>
            Upload context files
          </Button>
          <div className="flex gap-2">
            <Button variant="secondary" onClick={onClose}>
              Cancel
            </Button>
            <Button onClick={() => void handleImport()} disabled={chosen.size === 0} loading={importMutation.isPending}>
              {`Import ${plural(chosen.size, 'connection')}`}
            </Button>
          </div>
        </div>
      }
    >
      <input
        ref={fileInputRef}
        type="file"
        accept=".json,application/json"
        multiple
        aria-label="Upload context files"
        className="hidden"
        onChange={(e) => {
          void handleFiles(e.target.files)
          e.target.value = ''
        }}
      />
      <div className="space-y-3 text-sm">
        {uploaded ? (
          <p className="text-content-secondary">{`Read from ${plural(upload.files.length, 'uploaded file')}.`}</p>
        ) : (
          data && (
            <p className="text-content-secondary">
              Contexts in <code className="text-xs">{data.directory}</code> on the machine running Natscope. Natscope runs
              in Docker or on another host? Upload the <code className="text-xs">.json</code> files from that folder on your
              machine instead.
            </p>
          )
        )}
        {uploaded && (
          <p className="text-xs text-content-tertiary">
            Credentials, NKey and certificate files the contexts point at stay on your machine: add them to each connection
            after the import.
          </p>
        )}

        {isLoading ? (
          <SkeletonRows count={3} />
        ) : error && getErrorReason(error) === HOST_DISABLED ? (
          <Alert variant="info">{getErrorMessage(error)}</Alert>
        ) : error ? (
          <QueryErrorState error={error} onRetry={() => void refetch()} />
        ) : contexts.length === 0 ? (
          <EmptyState
            size="sm"
            title={uploaded ? 'No contexts in these files' : `No nats CLI contexts in ${data?.directory ?? 'the context folder'}`}
            description="Create one with `nats context add`, or upload context files."
          />
        ) : (
          <ul className="divide-y divide-border rounded-lg border border-border">
            {contexts.map((c) => (
              <li key={c.name} className="flex items-start gap-3 px-3 py-2.5">
                <input
                  type="checkbox"
                  aria-label={`Import ${c.name}`}
                  checked={chosen.has(c.name)}
                  disabled={!selectable(c)}
                  onChange={() => toggle(c.name)}
                  className="mt-1 rounded border-border-strong text-accent focus:ring-border-focus disabled:opacity-50"
                />
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="font-medium text-content-primary">{c.name}</span>
                    {c.selected && (
                      <span className="rounded-full bg-accent-light px-1.5 py-0.5 text-2xs font-medium text-accent-text">current</span>
                    )}
                    {c.exists && (
                      <span className="rounded-full bg-surface-tertiary px-1.5 py-0.5 text-2xs font-medium text-content-secondary">
                        already saved
                      </span>
                    )}
                  </div>
                  {c.description && <p className="text-xs text-content-tertiary">{c.description}</p>}
                  {c.urls.length > 0 && <p className="truncate font-mono text-xs text-content-secondary">{c.urls.join(', ')}</p>}
                  {c.importable && (
                    <p className="mt-0.5 flex flex-wrap gap-x-3 text-2xs text-content-tertiary">
                      <span>{c.authMethod === 'none' ? 'No auth' : AUTH_LABELS[c.authMethod]}</span>
                      {c.tls && <span>TLS</span>}
                      {c.jetstreamDomain && <span>{`domain ${c.jetstreamDomain}`}</span>}
                      {c.jetstreamApiPrefix && <span>{`API prefix ${c.jetstreamApiPrefix}`}</span>}
                      {c.inboxPrefix && <span>{`inbox ${c.inboxPrefix}`}</span>}
                    </p>
                  )}
                  {c.warnings.length > 0 && (
                    <ul className="mt-1 space-y-0.5">
                      {c.warnings.map((w) => (
                        <li key={w} className="flex items-start gap-1 text-xs text-status-warning-text">
                          <WarningIcon className="mt-0.5 h-3 w-3 shrink-0" />
                          <span>{w}</span>
                        </li>
                      ))}
                    </ul>
                  )}
                </div>
              </li>
            ))}
          </ul>
        )}
      </div>
    </Modal>
  )
}
