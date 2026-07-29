import { useCallback, useMemo, useRef, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { downloadBlob } from '@/utils/download'
import { toast } from '@/utils/toast'
import { plural } from '@/utils/plural'
import { getErrorMessage } from '@/api/errors'
import { Button, DownloadIcon, UploadIcon } from '@/components/ui'
import { SettingsSection } from '@/components/settings/SettingsSection'
import {
  listSections,
  exportWorkspace,
  validateWorkspace,
  importWorkspace,
  type SectionReport,
  type WorkspaceStrategy,
} from '@/api/workspace'
import { SettingsPage } from './SettingsPage'

const SECTIONS_QUERY_KEY = ['workspace', 'sections'] as const
const REPLACE_CONFIRM_WORD = 'REPLACE'

function toggle(set: Set<string>, key: string): Set<string> {
  const next = new Set(set)
  if (next.has(key)) next.delete(key)
  else next.add(key)
  return next
}

export default function WorkspacePage() {
  const queryClient = useQueryClient()

  // --- Sections (server state via React Query: handles StrictMode double-mount,
  // loading, and error without a hand-rolled useEffect) ---
  const sectionsQuery = useQuery({ queryKey: SECTIONS_QUERY_KEY, queryFn: listSections })
  const sections = useMemo(() => sectionsQuery.data ?? [], [sectionsQuery.data])

  // --- Export selection (defaults to all once sections load) ---
  const [exportSel, setExportSel] = useState<Set<string> | null>(null)
  const effectiveExportSel = exportSel ?? new Set(sections.map((s) => s.key))

  // --- Import state ---
  const fileRef = useRef<HTMLInputElement>(null)
  const [payload, setPayload] = useState<Uint8Array<ArrayBuffer> | null>(null)
  const [fileName, setFileName] = useState<string | null>(null)
  const [strategy, setStrategy] = useState<WorkspaceStrategy>('merge')
  const [reports, setReports] = useState<SectionReport[] | null>(null)
  const [importSel, setImportSel] = useState<Set<string>>(new Set())
  const [confirmText, setConfirmText] = useState('')
  const [confirmOpen, setConfirmOpen] = useState(false)

  // Monotonic guard so a slow validate (e.g. for a strategy the user already
  // toggled away from) never overwrites the result of a newer one.
  const validateSeq = useRef(0)

  const exportMut = useMutation({
    mutationFn: (keys: string[]) => exportWorkspace(keys),
    onSuccess: (bytes) => {
      downloadBlob(new Blob([bytes], { type: 'application/json' }), `natscope-workspace-${new Date().toISOString().split('T')[0]}.json`)
      toast.success('Workspace exported — secrets (credentials, tokens, TLS keys) were not included')
    },
    onError: (e) => toast.error(`Export failed: ${getErrorMessage(e)}`),
  })

  const validateMut = useMutation({
    mutationFn: (vars: { bytes: Uint8Array<ArrayBuffer>; strat: WorkspaceStrategy }) =>
      validateWorkspace(vars.bytes, [], vars.strat),
  })
  const { mutate: runValidateMutate } = validateMut

  // `fresh` is true only for a brand-new file — the one time we (re)seed the
  // section selection; otherwise we keep the user's current selection intersected.
  const runValidate = useCallback(
    (bytes: Uint8Array<ArrayBuffer>, strat: WorkspaceStrategy, fresh: boolean) => {
      const seq = ++validateSeq.current
      runValidateMutate(
        { bytes, strat },
        {
          onSuccess: (r) => {
            if (validateSeq.current !== seq) return // a newer validate superseded this one
            setReports(r)
            const supported = new Set(r.filter((x) => !x.unknown).map((x) => x.key))
            if (fresh) {
              setImportSel(supported)
            } else {
              setImportSel((prev) => new Set([...prev].filter((k) => supported.has(k))))
            }
          },
          onError: (e) => {
            if (validateSeq.current !== seq) return
            setReports(null)
            toast.error(`Invalid workspace file: ${getErrorMessage(e)}`)
          },
        },
      )
    },
    [runValidateMutate],
  )

  const importMut = useMutation({
    mutationFn: (vars: { bytes: Uint8Array<ArrayBuffer>; keys: string[]; strat: WorkspaceStrategy }) =>
      importWorkspace(vars.bytes, vars.keys, vars.strat),
    onSuccess: (results, vars) => {
      const total = results.reduce((acc, r) => acc + r.created + r.updated + r.deleted, 0)
      toast.success(`Import applied: ${plural(total, 'change')}`)
      const warnings = results.flatMap((r) => r.warnings)
      warnings.forEach((w) => toast.warning(w))
      setConfirmOpen(false)
      setConfirmText('')
      // Refresh counts and re-run the dry-run, preserving the user's selection.
      void queryClient.invalidateQueries({ queryKey: SECTIONS_QUERY_KEY })
      runValidate(vars.bytes, vars.strat, false)
    },
    onError: (e) => toast.error(`Import failed: ${getErrorMessage(e)}`),
  })
  const { mutate: runImportMutate } = importMut

  const handleFile = useCallback(
    async (file: File) => {
      const bytes = new Uint8Array(await file.arrayBuffer())
      setPayload(bytes)
      setFileName(file.name)
      setConfirmOpen(false)
      setConfirmText('')
      runValidate(bytes, strategy, true)
    },
    [runValidate, strategy],
  )

  const handleStrategyChange = useCallback(
    (s: WorkspaceStrategy) => {
      setStrategy(s)
      setConfirmOpen(false)
      setConfirmText('')
      if (payload) runValidate(payload, s, false)
    },
    [payload, runValidate],
  )

  // Predicted deletions for the CURRENTLY-SELECTED sections — drives the
  // destructive-import guard.
  const selectedDeletions = useMemo(() => {
    if (!reports) return 0
    return reports
      .filter((r) => importSel.has(r.key) && !r.unknown)
      .reduce((acc, r) => acc + r.deleted, 0)
  }, [reports, importSel])

  const needsConfirm = strategy === 'replace' && selectedDeletions > 0

  const doImport = useCallback(() => {
    if (!payload) return
    runImportMutate({ bytes: payload, keys: [...importSel], strat: strategy })
  }, [payload, importSel, strategy, runImportMutate])

  const handleApply = useCallback(() => {
    if (!payload || importSel.size === 0) return
    // Replace with deletions is irreversible (removes saved connections and
    // proto sources along with their credentials/tokens) — require type-to-confirm.
    if (needsConfirm) {
      setConfirmOpen(true)
      return
    }
    doImport()
  }, [payload, importSel, needsConfirm, doImport])

  const validating = validateMut.isPending
  const importing = importMut.isPending

  return (
    <SettingsPage
      title="Workspace"
      description="Export and import connections, proto sources, mappings, templates, and settings."
      meta="Secrets (credentials, tokens, TLS keys) are never exported"
    >
      <div data-testid="workspace-page" className="space-y-6">
        <SettingsSection
          title="Export workspace"
          description="Select what to include. Secrets (credentials, tokens, TLS keys) are never exported."
          icon={<DownloadIcon />}
        >
          {sectionsQuery.isLoading ? (
            <p className="text-sm text-content-muted">Loading sections…</p>
          ) : sectionsQuery.isError ? (
            <p className="text-sm text-status-error-text">Failed to load sections: {getErrorMessage(sectionsQuery.error)}</p>
          ) : sections.length === 0 ? (
            <p className="text-sm text-content-muted">No sections available.</p>
          ) : (
            <div className="space-y-2" data-testid="export-sections">
              {sections.map((s) => (
                <label key={s.key} className="flex items-center gap-2 text-sm text-gray-700">
                  <input
                    type="checkbox"
                    className="rounded text-accent focus:ring-border-focus"
                    checked={effectiveExportSel.has(s.key)}
                    onChange={() => setExportSel(toggle(effectiveExportSel, s.key))}
                    data-testid={`export-section-${s.key}`}
                  />
                  <span>
                    {s.title} <span className="text-content-muted">({s.count})</span>
                  </span>
                </label>
              ))}
            </div>
          )}
          <div>
            <Button
              onClick={() => exportMut.mutate([...effectiveExportSel])}
              disabled={effectiveExportSel.size === 0}
              loading={exportMut.isPending}
              data-testid="workspace-export-btn"
            >
              Download workspace
            </Button>
          </div>
        </SettingsSection>

        <SettingsSection
          title="Import workspace"
          description="Upload a workspace file to preview changes before applying."
          icon={<UploadIcon />}
        >
          <div className="flex flex-wrap items-center gap-3">
            <input
              ref={fileRef}
              type="file"
              accept="application/json,.json"
              className="hidden"
              data-testid="workspace-file-input"
              onChange={(e) => {
                const f = e.target.files?.[0]
                if (f) void handleFile(f)
                e.target.value = ''
              }}
            />
            <Button variant="secondary" onClick={() => fileRef.current?.click()}>
              Choose file…
            </Button>
            {fileName && <span className="text-xs text-content-tertiary">{fileName}</span>}
          </div>

          {payload && (
            <div className="flex flex-wrap items-center gap-4">
              <span className="text-sm font-medium text-gray-700">Strategy</span>
              {(['merge', 'replace'] as WorkspaceStrategy[]).map((s) => (
                <label key={s} className="flex items-center gap-1.5 text-sm text-gray-700">
                  <input
                    type="radio"
                    name="strategy"
                    checked={strategy === s}
                    onChange={() => handleStrategyChange(s)}
                    data-testid={`strategy-${s}`}
                  />
                  <span className="capitalize">{s}</span>
                </label>
              ))}
            </div>
          )}

          {validating && <p className="text-sm text-content-muted">Validating…</p>}

          {reports && (
            <div className="space-y-2" data-testid="import-reports">
              {reports.length === 0 && <p className="text-sm text-content-muted">File has no sections.</p>}
              {reports.map((r) => (
                <div
                  key={r.key}
                  data-testid={`import-report-${r.key}`}
                  className={`flex items-start gap-2 p-2 rounded-md border ${r.unknown ? 'border-amber-200 bg-status-warning-bg' : 'border-border'}`}
                >
                  <input
                    type="checkbox"
                    className="mt-0.5 rounded text-accent focus:ring-border-focus"
                    disabled={r.unknown}
                    checked={importSel.has(r.key)}
                    onChange={() => setImportSel((prev) => toggle(prev, r.key))}
                    data-testid={`import-section-${r.key}`}
                  />
                  <div className="text-sm">
                    <div className="font-medium text-gray-800">{r.key}</div>
                    {r.unknown ? (
                      <div className="text-xs text-amber-700">Unknown section — not supported, will be skipped.</div>
                    ) : (
                      <div className="text-xs text-content-tertiary">
                        <span className="text-green-700">{r.created} new</span>
                        {' · '}
                        <span className="text-accent-text">{r.updated} updated</span>
                        {r.deleted > 0 && (
                          <>
                            {' · '}
                            <span className="text-red-700" data-testid={`import-report-${r.key}-deleted`}>
                              {r.deleted} deleted
                            </span>
                          </>
                        )}
                        {r.conflicts.length > 0 && (
                          <>
                            {' · '}
                            <span className="text-amber-700">{plural(r.conflicts.length, 'conflict')}</span>
                          </>
                        )}
                      </div>
                    )}
                    {r.warnings.map((w, i) => (
                      <div key={i} className="text-xs text-status-warning-text mt-0.5">
                        ⚠ {w}
                      </div>
                    ))}
                  </div>
                </div>
              ))}

              {confirmOpen ? (
                <div
                  className="mt-2 p-3 rounded-md border border-red-300 bg-status-error-bg space-y-2"
                  data-testid="replace-confirm"
                >
                  <p className="text-sm text-red-800">
                    This <strong>replace</strong> import will permanently delete{' '}
                    <strong>{selectedDeletions}</strong> existing item{selectedDeletions === 1 ? '' : 's'} (including
                    saved connections and their credentials, and proto sources and their git tokens). This cannot be
                    undone.
                  </p>
                  <p className="text-xs text-red-700">
                    Type <code className="font-mono font-semibold">{REPLACE_CONFIRM_WORD}</code> to confirm.
                  </p>
                  <input
                    type="text"
                    autoFocus
                    value={confirmText}
                    onChange={(e) => setConfirmText(e.target.value)}
                    placeholder={REPLACE_CONFIRM_WORD}
                    data-testid="replace-confirm-input"
                    className="w-40 px-3 py-1.5 text-sm border border-red-300 rounded-md focus:ring-status-error-border focus:border-status-error-border"
                  />
                  <div className="flex items-center gap-2">
                    <Button
                      variant="danger"
                      onClick={doImport}
                      disabled={importing || confirmText !== REPLACE_CONFIRM_WORD}
                      data-testid="replace-confirm-btn"
                    >
                      {importing ? 'Applying…' : `Delete ${selectedDeletions} & apply`}
                    </Button>
                    <Button
                      variant="secondary"
                      onClick={() => {
                        setConfirmOpen(false)
                        setConfirmText('')
                      }}
                    >
                      Cancel
                    </Button>
                  </div>
                </div>
              ) : (
                <div className="mt-2">
                  <Button
                    variant={needsConfirm ? 'danger' : 'primary'}
                    onClick={handleApply}
                    disabled={importing || importSel.size === 0}
                    data-testid="workspace-import-btn"
                  >
                    {importing ? 'Applying…' : `Apply ${strategy} import`}
                  </Button>
                </div>
              )}
            </div>
          )}
        </SettingsSection>
      </div>
    </SettingsPage>
  )
}
