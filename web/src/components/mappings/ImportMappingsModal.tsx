import { useMemo, useRef, useState } from 'react'
import { useProtoSources } from '@/contexts/proto'
import { SourcePicker } from '@/components/proto/SourcePicker'
import { Modal, Button, Badge, WarningIcon, TrashIcon, UploadIcon } from '@/components/ui'
import Tooltip from '@/components/common/Tooltip'
import { cn } from '@/utils/cn'
import type { ImportDraftRow, ParseError, ParseResult } from './useMappingImportExport'

interface Props {
  onClose: () => void
  parseText: (text: string) => ParseResult | ParseError
  parseFile: (file: File) => Promise<ParseResult | ParseError>
  commit: (
    rows: Array<{ pattern: string; messageType: string; sourceId: string }>,
  ) => Promise<boolean>
  isCommitting: boolean
  error: string
}

interface ReviewRow extends ImportDraftRow {
  rid: string
  sourceId: string
}

type Stage = 'pick' | 'review'

export function ImportMappingsModal({
  onClose,
  parseText,
  parseFile,
  commit,
  isCommitting,
  error,
}: Props) {
  const [stage, setStage] = useState<Stage>('pick')
  const [rows, setRows] = useState<ReviewRow[]>([])
  const [parseError, setParseError] = useState('')
  const [bulkSourceId, setBulkSourceId] = useState('')
  const [importJson, setImportJson] = useState('')
  const fileInputRef = useRef<HTMLInputElement>(null)

  const { data: sources = [] } = useProtoSources()
  const enabledSources = useMemo(() => sources.filter((s) => s.enabled), [sources])

  const handleFileChange = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0]
    event.target.value = ''
    if (!file) return
    handleParseResult(await parseFile(file))
  }

  const handlePasteContinue = () => {
    handleParseResult(parseText(importJson))
  }

  const handleParseResult = (result: ParseResult | ParseError) => {
    if (!result.ok) {
      setParseError(result.error)
      return
    }
    setParseError('')
    setRows(
      result.rows.map((r, i) => ({
        rid: `${i}-${r.pattern}`,
        pattern: r.pattern,
        messageType: r.messageType,
        sourceId: '',
      })),
    )
    setStage('review')
  }

  const setRowSource = (rid: string, sourceId: string) => {
    setRows((prev) => prev.map((r) => (r.rid === rid ? { ...r, sourceId } : r)))
  }

  const removeRow = (rid: string) => {
    setRows((prev) => prev.filter((r) => r.rid !== rid))
  }

  const applyBulkToAll = () => {
    if (!bulkSourceId) return
    setRows((prev) => prev.map((r) => ({ ...r, sourceId: bulkSourceId })))
  }

  const applyBulkToEmpty = () => {
    if (!bulkSourceId) return
    setRows((prev) => prev.map((r) => (r.sourceId ? r : { ...r, sourceId: bulkSourceId })))
  }

  const duplicateKeys = useMemo(() => {
    const counts = new Map<string, number>()
    for (const r of rows) {
      if (!r.sourceId) continue
      const key = `${r.pattern}\x00${r.sourceId}`
      counts.set(key, (counts.get(key) ?? 0) + 1)
    }
    const dups = new Set<string>()
    for (const [key, n] of counts) if (n > 1) dups.add(key)
    return dups
  }, [rows])

  const emptyCount = rows.filter((r) => !r.sourceId).length
  const dupCount = duplicateKeys.size
  const readyCount = rows.length - emptyCount
  const ready = rows.length > 0 && emptyCount === 0 && dupCount === 0

  const submit = async (skipEmpty: boolean) => {
    const filtered = skipEmpty ? rows.filter((r) => r.sourceId) : rows
    if (filtered.length === 0) return
    const payload = filtered.map((r) => ({
      pattern: r.pattern,
      messageType: r.messageType,
      sourceId: r.sourceId,
    }))
    const ok = await commit(payload)
    if (ok) onClose()
  }

  const footer =
    stage === 'pick' ? (
      <>
        <Button variant="secondary" onClick={onClose}>
          Cancel
        </Button>
        <Button onClick={handlePasteContinue} disabled={!importJson.trim()}>
          Continue
        </Button>
      </>
    ) : (
      <>
        <div className="mr-auto text-xs text-content-tertiary">
          Bound to <span className="font-medium text-content-secondary">v3</span> export format.
        </div>
        <Button variant="secondary" onClick={() => setStage('pick')}>
          Back
        </Button>
        <Button variant="secondary" onClick={onClose}>
          Cancel
        </Button>
        {emptyCount > 0 && rows.length > emptyCount && (
          <Button
            variant="secondary"
            onClick={() => submit(true)}
            disabled={isCommitting || dupCount > 0}
            className="!text-amber-700 !border-amber-300 hover:!bg-status-warning-bg"
          >
            Skip {emptyCount} & Import {readyCount}
          </Button>
        )}
        <Button onClick={() => submit(false)} disabled={!ready} loading={isCommitting}>
          Import {rows.length}
        </Button>
      </>
    )

  return (
    <Modal
      isOpen
      onClose={onClose}
      title={stage === 'pick' ? 'Import Mappings' : 'Import Mappings — Review'}
      size={stage === 'pick' ? 'md' : 'xl'}
      footer={footer}
    >
      {stage === 'pick' ? (
        <PickStage
          importJson={importJson}
          setImportJson={setImportJson}
          fileInputRef={fileInputRef}
          onFileChange={handleFileChange}
          onContinue={handlePasteContinue}
          error={parseError || error}
        />
      ) : (
        <ReviewStage
          rows={rows}
          duplicateKeys={duplicateKeys}
          enabledSources={enabledSources}
          totalCount={rows.length}
          readyCount={readyCount}
          emptyCount={emptyCount}
          dupCount={dupCount}
          bulkSourceId={bulkSourceId}
          setBulkSourceId={setBulkSourceId}
          applyBulkToAll={applyBulkToAll}
          applyBulkToEmpty={applyBulkToEmpty}
          setRowSource={setRowSource}
          removeRow={removeRow}
          error={error}
        />
      )}
    </Modal>
  )
}

// ───────────────────────────────────── Stage 1 ──────────────────────────────────

function PickStage({
  importJson,
  setImportJson,
  fileInputRef,
  onFileChange,
  onContinue,
  error,
}: {
  importJson: string
  setImportJson: (v: string) => void
  fileInputRef: React.RefObject<HTMLInputElement>
  onFileChange: (e: React.ChangeEvent<HTMLInputElement>) => void
  onContinue: () => void
  error: string
}) {
  return (
    <div className="p-6 space-y-5 overflow-y-auto">
      <button
        onClick={() => fileInputRef.current?.click()}
        className="group w-full px-6 py-8 border-2 border-dashed border-border-strong rounded-card hover:border-accent hover:bg-accent-light transition-colors flex flex-col items-center gap-3"
      >
        <div className="w-12 h-12 rounded-full bg-surface-tertiary group-hover:bg-surface-primary flex items-center justify-center transition-colors">
          <UploadIcon className="w-6 h-6 text-content-tertiary group-hover:text-accent" />
        </div>
        <div className="text-center">
          <div className="text-sm font-medium text-content-primary">Drop a JSON file or click to browse</div>
          <div className="mt-1 text-xs text-content-tertiary">v3 mappings export</div>
        </div>
      </button>
      <input ref={fileInputRef} type="file" accept=".json" onChange={onFileChange} className="hidden" />

      <div className="flex items-center gap-3">
        <div className="flex-1 h-px bg-border" />
        <span className="text-xs text-content-muted uppercase tracking-wide">or paste</span>
        <div className="flex-1 h-px bg-border" />
      </div>

      <div className="space-y-2">
        <textarea
          value={importJson}
          onChange={(e) => setImportJson(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
              e.preventDefault()
              onContinue()
            }
          }}
          spellCheck={false}
          placeholder={`{\n  "version": 3,\n  "mappings": [\n    { "pattern": "ORDERS.*", "messageType": "api.v1.OrderEvent" }\n  ]\n}`}
          className="w-full h-44 px-4 py-3 text-xs font-mono leading-relaxed bg-surface-secondary border border-border rounded-card focus:bg-surface-primary focus:ring-2 focus:ring-border-focus focus:border-transparent resize-none placeholder:text-content-muted"
        />
        {error ? (
          <div data-testid="import-error" className="flex items-start gap-2 px-3 py-2 rounded-button bg-status-error-bg border border-red-200 text-xs text-red-700">
            <WarningIcon className="w-4 h-4 flex-shrink-0 mt-0.5" />
            <span>{error}</span>
          </div>
        ) : (
          <p className="text-xs text-content-tertiary">
            Source IDs are environment-specific and aren't part of the export. Pick a target source per row in the next step.
          </p>
        )}
      </div>
    </div>
  )
}

// ───────────────────────────────────── Stage 2 ──────────────────────────────────

function ReviewStage({
  rows,
  duplicateKeys,
  enabledSources,
  totalCount,
  readyCount,
  emptyCount,
  dupCount,
  bulkSourceId,
  setBulkSourceId,
  applyBulkToAll,
  applyBulkToEmpty,
  setRowSource,
  removeRow,
  error,
}: {
  rows: ReviewRow[]
  duplicateKeys: Set<string>
  enabledSources: ReturnType<typeof useProtoSources>['data'] extends infer T
    ? T extends Array<infer U>
      ? U[]
      : never
    : never
  totalCount: number
  readyCount: number
  emptyCount: number
  dupCount: number
  bulkSourceId: string
  setBulkSourceId: (v: string) => void
  applyBulkToAll: () => void
  applyBulkToEmpty: () => void
  setRowSource: (rid: string, sourceId: string) => void
  removeRow: (rid: string) => void
  error: string
}) {
  return (
    <div className="flex-1 flex flex-col overflow-hidden">
      {/* Sticky stats + bulk panel */}
      <div className="px-6 pt-5 pb-4 border-b border-gray-100 bg-surface-secondary space-y-4">
        <div className="flex items-center gap-2 flex-wrap">
          <StatPill label="Total" value={totalCount} tone="default" />
          <StatPill label="Ready" value={readyCount} tone={readyCount === totalCount && totalCount > 0 ? 'success' : 'default'} />
          <StatPill label="Missing source" value={emptyCount} tone={emptyCount > 0 ? 'warning' : 'default'} />
          <StatPill label="Duplicates" value={dupCount} tone={dupCount > 0 ? 'error' : 'default'} />
        </div>

        <div className="flex items-end gap-3">
          <div className="flex-1 min-w-0">
            <label className="block text-2xs font-medium text-content-tertiary uppercase tracking-wide mb-1.5">
              Bulk assign
            </label>
            <SourcePicker
              value={bulkSourceId}
              onChange={setBulkSourceId}
              placeholder="Pick a source to apply…"
            />
          </div>
          <Button
            size="md"
            variant="primary"
            onClick={applyBulkToAll}
            disabled={!bulkSourceId || rows.length === 0}
          >
            Apply to all
          </Button>
          <Button
            size="md"
            variant="secondary"
            onClick={applyBulkToEmpty}
            disabled={!bulkSourceId || emptyCount === 0}
          >
            Apply to empty
          </Button>
        </div>
      </div>

      {/* Body */}
      <div className="flex-1 overflow-auto px-6 py-4 bg-surface-primary" data-testid="review-table">
        {enabledSources.length === 0 ? (
          <EmptyHint
            tone="warning"
            title="No enabled proto sources"
            body="Add or enable a proto source in Settings → Proto Files before importing."
          />
        ) : rows.length === 0 ? (
          <EmptyHint
            tone="default"
            title="Nothing to import"
            body="All rows were removed. Go back to upload again."
          />
        ) : (
          <ul className="space-y-2">
            {rows.map((r, idx) => {
              const dupKey = `${r.pattern}\x00${r.sourceId}`
              const isDup = !!r.sourceId && duplicateKeys.has(dupKey)
              const empty = !r.sourceId
              const status: 'ready' | 'empty' | 'dup' = isDup ? 'dup' : empty ? 'empty' : 'ready'
              return <ReviewCard key={r.rid} idx={idx} row={r} status={status} setRowSource={setRowSource} removeRow={removeRow} />
            })}
          </ul>
        )}
      </div>

      {error && (
        <div className="px-6 py-2.5 border-t border-red-200 bg-status-error-bg text-xs text-red-700 flex items-center gap-2" data-testid="import-error">
          <WarningIcon className="w-4 h-4 flex-shrink-0" />
          <span>{error}</span>
        </div>
      )}
    </div>
  )
}

// ───────────────────────────────────── Atoms ──────────────────────────────────

function StatPill({
  label,
  value,
  tone,
}: {
  label: string
  value: number
  tone: 'default' | 'success' | 'warning' | 'error'
}) {
  const styles = {
    default: 'bg-surface-primary border-border text-content-secondary',
    success: 'bg-status-success-bg border-green-200 text-green-700',
    warning: 'bg-status-warning-bg border-amber-200 text-amber-700',
    error: 'bg-status-error-bg border-red-200 text-red-700',
  }[tone]
  return (
    <div className={cn('inline-flex items-center gap-2 px-2.5 py-1 rounded-full border text-xs', styles)}>
      <span className="font-medium">{label}</span>
      <span className="font-mono tabular-nums font-semibold">{value}</span>
    </div>
  )
}

function ReviewCard({
  idx,
  row,
  status,
  setRowSource,
  removeRow,
}: {
  idx: number
  row: ReviewRow
  status: 'ready' | 'empty' | 'dup'
  setRowSource: (rid: string, sourceId: string) => void
  removeRow: (rid: string) => void
}) {
  const railColor = {
    ready: 'border-l-green-500',
    empty: 'border-l-amber-400',
    dup: 'border-l-red-500',
  }[status]

  return (
    <li
      className={cn(
        'group rounded-card border border-border border-l-4 bg-surface-primary hover:shadow-card transition-shadow',
        railColor,
      )}
      data-testid={`review-row-${idx}`}
    >
      <div className="p-4 grid grid-cols-12 gap-4 items-center min-w-0">
        <div className="col-span-1 text-xs text-content-muted font-mono tabular-nums">
          {(idx + 1).toString().padStart(2, '0')}
        </div>

        <div className="col-span-7 min-w-0 space-y-1">
          <div className="flex items-center gap-2">
            <code className="px-2 py-0.5 rounded bg-surface-tertiary text-content-primary font-mono text-xs truncate">
              {row.pattern}
            </code>
            {status === 'dup' && (
              <Badge variant="error" size="sm" shape="pill">
                duplicate
              </Badge>
            )}
            {status === 'empty' && (
              <Badge variant="warning" size="sm" shape="pill">
                no source
              </Badge>
            )}
          </div>
          <div className="text-xs text-content-tertiary font-mono truncate" title={row.messageType}>
            → {row.messageType}
          </div>
        </div>

        <div className="col-span-3">
          <SourcePicker
            value={row.sourceId}
            onChange={(v) => setRowSource(row.rid, v)}
            placeholder="— pick source —"
          />
          {status === 'dup' && (
            <p className="mt-1 text-2xs text-status-error-text">Same (pattern, source) appears twice in the import set.</p>
          )}
        </div>

        <div className="col-span-1 flex justify-end">
          <Tooltip content="Remove from import">
            <button
              onClick={() => removeRow(row.rid)}
              className="p-1.5 rounded-button text-content-muted hover:text-status-error-text hover:bg-status-error-bg transition-colors"
              aria-label={`Remove row ${idx + 1}`}
              data-testid={`review-remove-${idx}`}
            >
              <TrashIcon className="w-4 h-4" />
            </button>
          </Tooltip>
        </div>
      </div>
    </li>
  )
}

function EmptyHint({
  tone,
  title,
  body,
}: {
  tone: 'default' | 'warning'
  title: string
  body: string
}) {
  const styles =
    tone === 'warning'
      ? 'border-amber-200 bg-status-warning-bg text-amber-900'
      : 'border-border bg-surface-secondary text-content-secondary'
  return (
    <div className={cn('rounded-card border p-6 text-center', styles)}>
      <div className="text-sm font-medium">{title}</div>
      <div className="mt-1 text-xs">{body}</div>
    </div>
  )
}
