import { useEffect, useMemo, useRef, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { Alert, Badge, Button, Input, Toggle } from '@/components/ui'
import { getErrorMessage } from '@/api/errors'
import {
  useCreateProtoSource,
  useUpdateProtoSource,
  useValidateLocalPath,
  useProtoSource,
  useCompileLocal,
  useCompileFiles,
} from '@/contexts/proto'
import type {
  CompileDiagnostic,
  CreateProtoSourceRequest,
  ProtoSourceType,
  UpdateProtoSourceRequest,
} from '@/api/protoSources'
import { CompileDiagnosticsList } from '@/components/proto/CompileDiagnosticsList'

interface CompileOutcome {
  messageTypes: number
  fileDescriptors: number
  valid: boolean
  diagnostics: CompileDiagnostic[]
}

// Splits a textarea value into a clean list of paths. Accepts both newlines
// and commas as separators (often pasted from CLI args, JSON arrays, etc.).
function parsePathList(text: string): string[] {
  return text
    .split(/[\n,]/)
    .map((s) => s.trim())
    .filter(Boolean)
}

interface Props {
  mode: 'create' | 'edit'
}

// Automatic (git/local): heuristic directory/repo walk, flagged experimental
// in the UI. Manual: explicit .proto path list, strict resolution only.
type SourceMode = 'automatic' | 'manual'

function modeOf(t: ProtoSourceType): SourceMode {
  return t === 'files' ? 'manual' : 'automatic'
}

const TYPE_OPTIONS: { value: ProtoSourceType; label: string; hint: string }[] = [
  { value: 'git', label: 'Git Repository', hint: 'Auto-walk · from tag' },
  { value: 'local', label: 'Local Directory', hint: 'Auto-walk · watched folder' },
  { value: 'files', label: 'Manual Files', hint: 'Strict · explicit path list' },
]

const TYPE_BADGE: Record<ProtoSourceType, 'primary' | 'warning' | 'success'> = {
  git: 'primary',
  local: 'warning',
  files: 'success',
}

export default function ProtoSourceEditPage({ mode }: Props) {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const isEdit = mode === 'edit'
  const sourceQuery = useProtoSource(isEdit && id ? id : null)
  const existing = sourceQuery.data

  const [sourceType, setSourceType] = useState<ProtoSourceType>('git')
  const [name, setName] = useState('')
  const [repository, setRepository] = useState('')
  const [token, setToken] = useState('')
  const [localPath, setLocalPath] = useState('')
  const [watcherEnabled, setWatcherEnabled] = useState(true)
  // Manual Files & Include Directories are bulk-first: the user pastes paths
  // separated by newlines or commas. Parsed only when validating/saving.
  const [filesText, setFilesText] = useState('')
  const [includeDirsText, setIncludeDirsText] = useState('')
  // Import Roots, when set, disables auto-detection entirely. Exclude Prefixes
  // drop whole subtrees before resolution, fixing duplicate vendored files.
  const [importRootsText, setImportRootsText] = useState('')
  const [excludePrefixesText, setExcludePrefixesText] = useState('')

  const [error, setError] = useState<string | null>(null)
  const [localValidate, setLocalValidate] = useState<{ valid: boolean; protoFileCount: number; error?: string } | null>(null)
  // Inline compile result so the user can "Save & Compile" right here and see
  // the outcome (counts + diagnostics) without bouncing back to the card.
  const [compileOut, setCompileOut] = useState<CompileOutcome | null>(null)
  const [compileErr, setCompileErr] = useState<string | null>(null)

  const createMutation = useCreateProtoSource()
  const updateMutation = useUpdateProtoSource()
  const validateLocal = useValidateLocalPath()
  const compileLocalMutation = useCompileLocal()
  const compileFilesMutation = useCompileFiles()
  const isCompiling = compileLocalMutation.isPending || compileFilesMutation.isPending

  const lastHydratedFor = useRef<string | null>(null)
  useEffect(() => {
    if (!isEdit || !existing) return
    if (lastHydratedFor.current === existing.id) return
    lastHydratedFor.current = existing.id
    setSourceType(existing.sourceType)
    setName(existing.name)
    setRepository(existing.repository)
    setToken('') // never returned from API
    setLocalPath(existing.localPath || '')
    setWatcherEnabled(existing.watcherEnabled)
    setFilesText(existing.files.join('\n'))
    setIncludeDirsText(existing.includeDirs.join('\n'))
    setImportRootsText((existing.importRoots ?? []).join('\n'))
    setExcludePrefixesText((existing.excludePrefixes ?? []).join('\n'))
  }, [isEdit, existing])

  const cleanedFiles = useMemo(() => parsePathList(filesText), [filesText])
  const cleanedIncludeDirs = useMemo(() => parsePathList(includeDirsText), [includeDirsText])
  const cleanedImportRoots = useMemo(() => parsePathList(importRootsText), [importRootsText])
  const cleanedExcludePrefixes = useMemo(() => parsePathList(excludePrefixesText), [excludePrefixesText])

  const isLoading = createMutation.isPending || updateMutation.isPending
  const saveDisabled =
    isLoading ||
    !name.trim() ||
    (sourceType === 'git' && !repository.trim()) ||
    (sourceType === 'local' && !localPath.trim()) ||
    (sourceType === 'files' && cleanedFiles.length === 0)

  const handleValidateLocalPath = async () => {
    if (!localPath.trim()) return
    setLocalValidate(null)
    try {
      const r = await validateLocal.mutateAsync({ path: localPath.trim() })
      setLocalValidate(r)
    } catch (err) {
      setLocalValidate({ valid: false, protoFileCount: 0, error: getErrorMessage(err) })
    }
  }

  const createdIdRef = useRef<string | null>(null)

  // Saves without navigating away; shared by "Save" (navigates after) and
  // "Save & Compile" (stays to render the result).
  const persist = async (): Promise<string | undefined> => {
    const existingId = isEdit && existing ? existing.id : createdIdRef.current
    if (existingId) {
      const data: UpdateProtoSourceRequest = {}
      if (isEdit && existing) {
        data.name = name !== existing.name ? name : undefined
        if (sourceType === 'git') {
          data.repository = repository !== existing.repository ? repository : undefined
          if (token) data.token = token
        }
        if (sourceType === 'local') {
          data.localPath = localPath !== existing.localPath ? localPath : undefined
        }
      } else {
        data.name = name
        if (sourceType === 'git') {
          data.repository = repository
          if (token) data.token = token
        }
        if (sourceType === 'local') {
          data.localPath = localPath
        }
      }
      if (modeOf(sourceType) === 'automatic') {
        data.importRoots = cleanedImportRoots
        data.excludePrefixes = cleanedExcludePrefixes
      }
      if (sourceType === 'files') {
        data.files = cleanedFiles
        data.includeDirs = cleanedIncludeDirs
      }
      await updateMutation.mutateAsync({ id: existingId, data })
      return existingId
    }
    const data: CreateProtoSourceRequest = { name, sourceType }
    if (sourceType === 'git') {
      data.repository = repository
      data.token = token || undefined
    }
    if (sourceType === 'local') {
      data.localPath = localPath
      data.watcherEnabled = watcherEnabled
    }
    if (modeOf(sourceType) === 'automatic') {
      if (cleanedImportRoots.length) data.importRoots = cleanedImportRoots
      if (cleanedExcludePrefixes.length) data.excludePrefixes = cleanedExcludePrefixes
    }
    if (sourceType === 'files') {
      data.files = cleanedFiles
      data.includeDirs = cleanedIncludeDirs
    }
    const created = await createMutation.mutateAsync(data)
    createdIdRef.current = created.id
    return created.id
  }

  const handleSave = async () => {
    setError(null)
    try {
      await persist()
      navigate('/settings/proto')
    } catch (err) {
      setError(getErrorMessage(err) || 'Failed to save proto source')
    }
  }

  // Save, then compile in place. Compilation errors are diagnostics, not a save
  // failure — they render in the result panel below, the source still saved.
  const handleSaveAndCompile = async () => {
    setError(null)
    setCompileErr(null)
    setCompileOut(null)
    try {
      const sourceId = await persist()
      if (!sourceId) return
      if (sourceType === 'local') {
        setCompileOut(await compileLocalMutation.mutateAsync({ sourceId }))
      } else if (sourceType === 'files') {
        setCompileOut(await compileFilesMutation.mutateAsync({ sourceId }))
      }
    } catch (err) {
      setCompileErr(getErrorMessage(err) || 'Failed to compile')
    }
  }

  if (isEdit && sourceQuery.isLoading) {
    return (
      <div className="flex-1 flex items-center justify-center text-sm text-content-tertiary">Loading…</div>
    )
  }
  if (isEdit && !existing) {
    return (
      <div className="flex-1 flex items-center justify-center text-sm text-content-tertiary">
        Source not found.{' '}
        <Link to="/settings/proto" className="ml-1 text-accent hover:underline">Back to Proto Files</Link>
      </div>
    )
  }

  // Type-specific main content
  const showSplitForFiles = sourceType === 'files'

  const formColumn = (
    <div className="space-y-5">
      {/* Source Type — disabled in edit mode */}
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-2">Source Type</label>
        {isEdit ? (
          <Badge variant={TYPE_BADGE[sourceType]} size="sm">{sourceType.toUpperCase()}</Badge>
        ) : (
          <>
            <div className="grid grid-cols-3 gap-2">
              {TYPE_OPTIONS.map((opt) => {
                const selected = sourceType === opt.value
                return (
                  <button
                    key={opt.value}
                    type="button"
                    onClick={() => setSourceType(opt.value)}
                    className={`px-3 py-2 rounded-md border text-left transition-all ${
                      selected
                        ? 'border-border-focus bg-accent-light/50'
                        : 'border-border hover:border-border-strong bg-surface-primary'
                    }`}
                  >
                    <div className={`text-sm font-medium ${selected ? 'text-accent-text' : 'text-gray-800'}`}>
                      {opt.label}
                    </div>
                    <div className="text-xs text-content-tertiary mt-0.5">{opt.hint}</div>
                  </button>
                )
              })}
            </div>
            {modeOf(sourceType) === 'automatic' && (
              <p className="mt-2 text-xs text-amber-700">
                Auto-detects <code>buf.yaml</code>/<code>buf.work.yaml</code> module roots; otherwise infers
                import roots from the import graph. Well-known types (<code>google/protobuf/*</code>) resolve automatically.
              </p>
            )}
          </>
        )}
      </div>

      <div>
        <label htmlFor="name" className="block text-sm font-medium text-gray-700 mb-2">Name *</label>
        <Input
          id="name"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="e.g., Core Proto"
          disabled={isLoading}
        />
      </div>

      {sourceType === 'git' && (
        <>
          <div>
            <label htmlFor="repository" className="block text-sm font-medium text-gray-700 mb-2">Repository URL *</label>
            <Input
              id="repository"
              value={repository}
              onChange={(e) => setRepository(e.target.value)}
              placeholder="https://gitlab.com/org/proto.git"
              disabled={isLoading}
            />
          </div>
          <div>
            <label htmlFor="token" className="block text-sm font-medium text-gray-700 mb-2">
              Access Token {isEdit && <span className="text-content-muted font-normal">(leave empty to keep current)</span>}
            </label>
            <Input
              id="token"
              type="password"
              value={token}
              onChange={(e) => setToken(e.target.value)}
              placeholder={isEdit ? '********' : 'Personal access token'}
              disabled={isLoading}
            />
          </div>
        </>
      )}

      {sourceType === 'local' && (
        <>
          <div>
            <label htmlFor="localPath" className="block text-sm font-medium text-gray-700 mb-2">Directory Path *</label>
            <div className="flex gap-2">
              <div className="flex-1">
                <Input
                  id="localPath"
                  value={localPath}
                  onChange={(e) => {
                    setLocalPath(e.target.value)
                    setLocalValidate(null)
                  }}
                  placeholder="/path/to/proto/files"
                  disabled={isLoading}
                />
              </div>
              <Button
                type="button"
                variant="secondary"
                size="sm"
                onClick={handleValidateLocalPath}
                loading={validateLocal.isPending}
                disabled={!localPath.trim() || isLoading}
              >
                Validate
              </Button>
            </div>
            {localValidate && (
              <div className={`mt-2 text-xs px-3 py-2 rounded ${
                localValidate.valid ? 'bg-status-success-bg text-green-700' : 'bg-status-error-bg text-red-700'
              }`}>
                {localValidate.valid
                  ? `Valid path - ${localValidate.protoFileCount} .proto file(s) found`
                  : localValidate.error || 'Invalid path'}
              </div>
            )}
          </div>
          {!isEdit && (
            <div className="flex items-center justify-between">
              <div>
                <label className="text-sm font-medium text-gray-700">Enable File Watcher</label>
                <p className="text-xs text-content-muted mt-0.5">Automatically recompile on file changes</p>
              </div>
              <Toggle checked={watcherEnabled} onChange={setWatcherEnabled} label="Enable file watcher" />
            </div>
          )}
        </>
      )}

      {/* Advanced overrides — automatic (git/local) sources only */}
      {modeOf(sourceType) === 'automatic' && (
        <div className="rounded-md border border-border p-3 space-y-4">
          <div className="text-xs font-semibold text-content-secondary uppercase tracking-wide">
            Advanced (optional)
          </div>
          <div>
            <div className="flex items-center justify-between mb-2">
              <label htmlFor="excludePrefixesText" className="block text-sm font-medium text-gray-700">
                Exclude Prefixes
              </label>
              <span className="text-xs text-content-tertiary">{cleanedExcludePrefixes.length}</span>
            </div>
            <textarea
              id="excludePrefixesText"
              value={excludePrefixesText}
              onChange={(e) => setExcludePrefixesText(e.target.value)}
              spellCheck={false}
              className="w-full px-3 py-2 font-mono text-sm text-gray-800 bg-surface-primary border border-border-strong rounded-md focus:border-border-focus focus:outline-none resize-y min-h-[60px]"
              disabled={isLoading}
            />
            <p className="mt-1.5 text-xs text-content-tertiary leading-relaxed">
              Folders to skip during compilation, written as path prefixes relative to the source
              root — one per line, or comma-separated. Use this to drop generated or vendored copies
              that duplicate real <code>.proto</code> files and break the build. For example, typing{' '}
              <code>pb</code> ignores everything under <code>pb/</code>. Leave empty to compile the
              whole tree.
            </p>
          </div>
          <div>
            <div className="flex items-center justify-between mb-2">
              <label htmlFor="importRootsText" className="block text-sm font-medium text-gray-700">
                Import Roots
              </label>
              <span className="text-xs text-content-tertiary">{cleanedImportRoots.length}</span>
            </div>
            <textarea
              id="importRootsText"
              value={importRootsText}
              onChange={(e) => setImportRootsText(e.target.value)}
              spellCheck={false}
              className="w-full px-3 py-2 font-mono text-sm text-gray-800 bg-surface-primary border border-border-strong rounded-md focus:border-border-focus focus:outline-none resize-y min-h-[60px]"
              disabled={isLoading}
            />
            <p className="mt-1.5 text-xs text-content-tertiary leading-relaxed">
              Advanced — usually leave this empty. These are the base directories that your{' '}
              <code>import &quot;...&quot;</code> paths are written relative to. When empty, they are
              detected automatically from the import graph (and any <code>buf.yaml</code>). Fill this
              in only to override detection when the wrong root is picked — doing so turns
              auto-detection off completely.
            </p>
          </div>
        </div>
      )}

      {sourceType === 'files' && (
        <>
          <div>
            <div className="flex items-center justify-between mb-2">
              <label htmlFor="filesText" className="block text-sm font-medium text-gray-700">
                Proto Files *
              </label>
              <span className="text-xs text-content-tertiary">{cleanedFiles.length} paths</span>
            </div>
            <textarea
              id="filesText"
              value={filesText}
              onChange={(e) => setFilesText(e.target.value)}
              spellCheck={false}
              className="w-full px-3 py-2 font-mono text-sm text-gray-800 bg-surface-primary border border-border-strong rounded-md focus:border-border-focus focus:outline-none resize-y min-h-[140px]"
              placeholder={'/Users/you/proto/myapi/v1/myapi.proto\n/Users/you/proto/myapi/v1/events.proto'}
              disabled={isLoading}
            />
            <p className="mt-1.5 text-xs text-content-tertiary">
              Absolute paths, one per line or comma-separated. Strict — only these files are
              compiled.
            </p>
          </div>

          <div>
            <div className="flex items-center justify-between mb-2">
              <label htmlFor="includeDirsText" className="block text-sm font-medium text-gray-700">
                Include Directories
              </label>
              <span className="text-xs text-content-tertiary">{cleanedIncludeDirs.length} paths</span>
            </div>
            <textarea
              id="includeDirsText"
              value={includeDirsText}
              onChange={(e) => setIncludeDirsText(e.target.value)}
              spellCheck={false}
              className="w-full px-3 py-2 font-mono text-sm text-gray-800 bg-surface-primary border border-border-strong rounded-md focus:border-border-focus focus:outline-none resize-y min-h-[80px]"
              placeholder={'/Users/you/proto/third_party'}
              disabled={isLoading}
            />
            <p className="mt-1.5 text-xs text-content-tertiary">
              Resolves <code>import "..."</code> statements (like <code>protoc -I</code>). Files
              inside are not compiled, only made available as dependencies. Newline or comma
              separated.
            </p>
          </div>

        </>
      )}
    </div>
  )

  // Files-type right column shows only saved counts; no Validate button since
  // it duplicated Compile now. Diagnostics surface from Compile now after saving.
  const validationColumn = sourceType === 'files' ? (
    <div className="space-y-3 lg:sticky lg:top-6">
      <div className="rounded-lg border border-border bg-surface-primary p-4">
        <div className="text-sm font-semibold text-gray-700 mb-2">Summary</div>
        <dl className="text-sm text-content-secondary space-y-1">
          <div className="flex justify-between">
            <dt>Files</dt>
            <dd className="font-mono">{cleanedFiles.length}</dd>
          </div>
          <div className="flex justify-between">
            <dt>Include directories</dt>
            <dd className="font-mono">{cleanedIncludeDirs.length}</dd>
          </div>
        </dl>
        <p className="mt-3 text-xs text-content-tertiary">
          Compile is run automatically after saving from the source card.
        </p>
      </div>
    </div>
  ) : null

  return (
    <div className="flex-1 overflow-y-auto">
      <header className="bg-surface-primary border-b border-border sticky top-0 z-10">
        <div className="px-4 sm:px-6 lg:px-8 py-4 flex items-center justify-between gap-4">
          <div className="flex items-center gap-3 min-w-0">
            <Link
              to="/settings/proto"
              className="text-sm text-content-tertiary hover:text-content-primary flex items-center gap-1"
            >
              ← Back
            </Link>
            <div className="h-5 w-px bg-surface-hover" />
            <h1 className="text-lg font-semibold text-content-primary truncate">
              {isEdit ? 'Edit Proto Source' : 'Add Proto Source'}
            </h1>
          </div>
        </div>
      </header>

      <div className="px-4 sm:px-6 lg:px-8 py-6">
        {error && (
          <div className="mb-4">
            <Alert variant="error">{error}</Alert>
          </div>
        )}
        {/* Left column keeps a fixed width across modes so switching source type
            doesn't shift the form; right column stays in layout even when empty. */}
        <div className="grid grid-cols-1 lg:grid-cols-[minmax(0,640px)_minmax(0,1fr)] gap-6 max-w-7xl">
          <div className="space-y-6">
            {formColumn}
            {/* Action bar pinned to the end of the form column so the user
                doesn't have to chase Save buttons across a wide monitor. */}
            <div className="flex items-center gap-3 pt-4 border-t border-border">
              <Button onClick={handleSave} loading={isLoading} disabled={saveDisabled || isCompiling}>
                {isEdit ? 'Save Changes' : 'Add Source'}
              </Button>
              {(sourceType === 'local' || sourceType === 'files') && (
                <Button
                  variant="secondary"
                  onClick={handleSaveAndCompile}
                  loading={isCompiling}
                  disabled={saveDisabled || isLoading}
                >
                  Save &amp; Compile
                </Button>
              )}
              <Link
                to="/settings/proto"
                className="px-3 h-9 inline-flex items-center text-sm text-gray-700 hover:bg-surface-tertiary rounded-md"
              >
                Cancel
              </Link>
            </div>

            {/* Inline compile result so the user can iterate on Exclude Prefixes /
                Import Roots without leaving the form. */}
            {(compileOut || compileErr) && (
              <div className="space-y-2">
                {compileErr && <Alert variant="error">{compileErr}</Alert>}
                {compileOut && (
                  <>
                    <div
                      className={`text-xs px-3 py-2 rounded ${
                        compileOut.valid ? 'bg-status-success-bg text-green-700' : 'bg-surface-secondary text-content-secondary'
                      }`}
                    >
                      Compiled: {compileOut.fileDescriptors} file descriptors,{' '}
                      {compileOut.messageTypes} message types
                    </div>
                    {compileOut.diagnostics.length > 0 && (
                      <CompileDiagnosticsList diagnostics={compileOut.diagnostics} />
                    )}
                  </>
                )}
              </div>
            )}
          </div>
          <div>{showSplitForFiles ? validationColumn : null}</div>
        </div>
      </div>

    </div>
  )
}
