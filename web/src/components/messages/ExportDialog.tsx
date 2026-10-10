import { useState, useMemo, useRef, useEffect } from 'react'
import type { Message } from '@/types/nats'
import { Modal, DownloadIcon } from '@/components/ui'
import { formatCount } from '@/utils/formatters'
import { getErrorMessage } from '@/api/errors'
import { getMessages } from '@/api/messages'
import { useMessagesPolicy } from '@/contexts/settings'
import { toast } from '@/utils/toast'
import { createExportSerializer, type ExportFormat } from './exportFormat'
import { walkRange } from './exportRange'

type ExportScope = 'filtered' | 'all' | 'range'

// Range-walk page size; matches backend DefaultMaxMessageLimit (500) — higher
// is clamped server-side.
const RANGE_PAGE_SIZE = 500

// The file is assembled in browser memory, so the range is capped well below the
// 10M the settings allow.
const MAX_RANGE_LIMIT = 1_000_000

interface ExportOptions {
  format: ExportFormat
  scope: ExportScope
  includeMetadata: boolean
  includeHeaders: boolean
  includeDecoded: boolean
  limit?: number
  rangeLimit?: number
}

interface ExportDialogProps {
  isOpen: boolean
  onClose: () => void
  messages: Message[]
  filteredMessages?: Message[]
  streamName: string
  totalCount?: number
  /** Connection id — required for the server-side "range" export walk. */
  connectionId?: string | null
  /** Stream's first sequence — where the range walk starts. */
  streamFirstSeq?: number
  /** Subject filter active in the toolbar; the server-side range walk honours it. */
  subjectFilter?: string
}

export default function ExportDialog({
  isOpen,
  onClose,
  messages,
  filteredMessages,
  streamName,
  totalCount,
  connectionId,
  streamFirstSeq,
  subjectFilter,
}: ExportDialogProps) {
  const msgPolicy = useMessagesPolicy()
  const [options, setOptions] = useState<ExportOptions>(() => ({
    format: msgPolicy.defaultExportFormat,
    scope: 'filtered',
    includeMetadata: true,
    includeHeaders: false,
    includeDecoded: true,
    limit: 1000,
    rangeLimit: Math.min(msgPolicy.exportRangeLimit, MAX_RANGE_LIMIT),
  }))
  const patchOptions = (patch: Partial<ExportOptions>) => setOptions((prev) => ({ ...prev, ...patch }))
  const [isExporting, setIsExporting] = useState(false)
  // Range-export progress: number fetched so far, plus a cancel handle.
  const [rangeProgress, setRangeProgress] = useState<number | null>(null)
  const abortRef = useRef<AbortController | null>(null)

  // Abort any in-flight range export on unmount.
  useEffect(() => () => abortRef.current?.abort(), [])

  const rangeLimit = options.rangeLimit || Math.min(msgPolicy.exportRangeLimit, MAX_RANGE_LIMIT)

  const messagesToExport = useMemo(() => {
    switch (options.scope) {
      case 'filtered':
        return filteredMessages || messages
      case 'all':
        return messages
      default:
        return messages
    }
  }, [options.scope, messages, filteredMessages])

  const exportCount = useMemo(() => {
    const count = messagesToExport.length
    if (options.limit && count > options.limit) {
      return options.limit
    }
    return count
  }, [messagesToExport, options.limit])

  const serializerOptions = () => ({
    format: options.format,
    includeMetadata: options.includeMetadata,
    includeHeaders: options.includeHeaders,
    includeDecoded: options.includeDecoded,
  })

  const download = (parts: BlobPart[], mime: string, extension: string) => {
    const blob = new Blob(parts, { type: mime })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${streamName}-messages-${Date.now()}.${extension}`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  }

  // Server-side full-range export: page forward from first seq, full payloads
  // (max_payload_bytes: 0), up to limit.
  const handleRangeExport = async () => {
    if (!connectionId) {
      toast.error('Range export needs an active connection')
      return
    }
    const limit = rangeLimit
    const serializer = createExportSerializer(serializerOptions())
    const parts: BlobPart[] = [serializer.begin()]
    const controller = new AbortController()
    abortRef.current = controller
    setIsExporting(true)
    setRangeProgress(0)
    try {
      const result = await walkRange(
        (startSeq, signal) =>
          getMessages(
            streamName,
            {
              connection_id: connectionId,
              start_seq: startSeq,
              limit: RANGE_PAGE_SIZE,
              direction: 'forward',
              subject_filter: subjectFilter || undefined,
              // Full payloads — preview truncation would corrupt the export.
              max_payload_bytes: 0,
            },
            signal,
          ),
        {
          startSeq: streamFirstSeq && streamFirstSeq > 0 ? streamFirstSeq : 1,
          limit,
          signal: controller.signal,
          onProgress: setRangeProgress,
          onPage: (messages) => {
            parts.push(new Blob([serializer.chunk(messages)]))
          },
        },
      )

      if (result.aborted && result.count === 0) {
        toast.info('Export cancelled')
        return
      }
      parts.push(serializer.end())
      download(parts, serializer.mime, serializer.extension)
      if (result.aborted) {
        // Cancelled mid-export with partial data — warn so the user knows the
        // file is incomplete.
        toast.warning(`Export cancelled — ${formatCount(result.count)} of ${formatCount(limit)} messages saved`)
      } else if (result.truncated) {
        toast.warning(`Exported first ${formatCount(result.count)} messages (limit reached)`) // explicit truncation
      } else {
        toast.success(`Exported ${formatCount(result.count)} messages`)
      }
      onClose()
    } catch (error) {
      toast.error(`Export failed: ${getErrorMessage(error)}`)
    } finally {
      setIsExporting(false)
      setRangeProgress(null)
      abortRef.current = null
    }
  }

  const handleExport = async () => {
    if (options.scope === 'range') {
      await handleRangeExport()
      return
    }
    setIsExporting(true)
    try {
      const limit = options.limit || messagesToExport.length
      const serializer = createExportSerializer(serializerOptions())
      download(
        [serializer.begin(), serializer.chunk(messagesToExport.slice(0, limit)), serializer.end()],
        serializer.mime,
        serializer.extension,
      )
      onClose()
    } catch (error) {
      toast.error(`Export failed: ${getErrorMessage(error)}`)
    } finally {
      setIsExporting(false)
    }
  }

  const cancelRange = () => abortRef.current?.abort()

  return (
    <Modal isOpen={isOpen} onClose={onClose} title="Export Messages">
      <>
        {/* Content */}
        <div className="px-6 py-4 space-y-5 overflow-y-auto">
          {/* Format Selection */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">Format</label>
            <div className="grid grid-cols-3 gap-2">
              {(['json', 'ndjson', 'csv'] as ExportFormat[]).map((format) => (
                <button
                  key={format}
                  onClick={() => patchOptions({ format })}
                  className={`px-3 py-2 text-sm font-medium rounded-md border transition-colors ${
                    options.format === format
                      ? 'border-border-focus bg-accent-light text-accent-text'
                      : 'border-border-strong text-gray-700 hover:bg-surface-secondary'
                  }`}
                >
                  {format.toUpperCase()}
                </button>
              ))}
            </div>
            <p className="mt-1 text-xs text-content-tertiary">
              {options.format === 'json' && 'Pretty-printed JSON array'}
              {options.format === 'ndjson' && 'Newline-delimited JSON (one object per line)'}
              {options.format === 'csv' && 'Comma-separated values'}
            </p>
          </div>

          {/* Scope Selection */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">Export Scope</label>
            <div className="space-y-2">
              <label className="flex items-center gap-2">
                <input
                  type="radio"
                  name="scope"
                  value="filtered"
                  checked={options.scope === 'filtered'}
                  onChange={() => patchOptions({ scope: 'filtered' })}
                  className="text-accent focus:ring-border-focus"
                />
                <span className="text-sm text-gray-700">
                  Filtered messages ({(filteredMessages || messages).length})
                </span>
              </label>
              <label className="flex items-center gap-2">
                <input
                  type="radio"
                  name="scope"
                  value="all"
                  checked={options.scope === 'all'}
                  onChange={() => patchOptions({ scope: 'all' })}
                  className="text-accent focus:ring-border-focus"
                />
                <span className="text-sm text-gray-700">
                  All loaded messages ({messages.length})
                  {totalCount && totalCount > messages.length && (
                    <span className="text-content-muted ml-1">
                      of {totalCount} in stream
                    </span>
                  )}
                </span>
              </label>
              <label className={`flex items-center gap-2 ${connectionId ? '' : 'opacity-50'}`}>
                <input
                  type="radio"
                  name="scope"
                  value="range"
                  checked={options.scope === 'range'}
                  disabled={!connectionId}
                  onChange={() => patchOptions({ scope: 'range' })}
                  className="text-accent focus:ring-border-focus"
                  data-testid="export-scope-range"
                />
                <span className="text-sm text-gray-700">
                  Full range from server{' '}
                  {totalCount != null && <span className="text-content-muted">(~{formatCount(totalCount)} in stream)</span>}
                </span>
              </label>
            </div>
            {options.scope === 'range' && (
              <div className="mt-1.5 space-y-1 text-xs text-content-tertiary">
                <p>
                  Walks the stream server-side with full payloads, up to the limit below.{' '}
                  {subjectFilter ? (
                    <>
                      Only subjects matching the toolbar filter <span className="font-mono">{subjectFilter}</span> are exported.
                    </>
                  ) : (
                    'No subject filter is set, so every message in the stream is exported.'
                  )}
                </p>
                <p>
                  The file is built in the browser, so very large exports need a lot of memory (up to{' '}
                  {formatCount(MAX_RANGE_LIMIT)} messages at most). Cancel keeps what was fetched so far.
                </p>
              </div>
            )}
          </div>

          {/* Options */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">Options</label>
            <div className="space-y-2">
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={options.includeMetadata}
                  onChange={(e) => patchOptions({ includeMetadata: e.target.checked })}
                  className="text-accent focus:ring-border-focus rounded"
                />
                <span className="text-sm text-gray-700">Include metadata (sequence, subject, timestamp)</span>
              </label>
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={options.includeHeaders}
                  onChange={(e) => patchOptions({ includeHeaders: e.target.checked })}
                  className="text-accent focus:ring-border-focus rounded"
                />
                <span className="text-sm text-gray-700">Include message headers</span>
              </label>
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={options.includeDecoded}
                  onChange={(e) => patchOptions({ includeDecoded: e.target.checked })}
                  className="text-accent focus:ring-border-focus rounded"
                />
                <span className="text-sm text-gray-700">Include decoded protobuf data</span>
              </label>
            </div>
          </div>

          {/* Limit */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              {options.scope === 'range' ? 'Range Limit (hard cap)' : 'Message Limit'}
            </label>
            <div className="flex items-center gap-2">
              <input
                type="number"
                value={(options.scope === 'range' ? options.rangeLimit : options.limit) || ''}
                onChange={(e) => {
                  const key = options.scope === 'range' ? 'rangeLimit' : 'limit'
                  if (!e.target.value) {
                    patchOptions({ [key]: undefined })
                    return
                  }
                  // Clamp to [1, MAX_RANGE_LIMIT] — zero/negative would silently
                  // drop data, and the file is assembled in browser memory.
                  const parsed = parseInt(e.target.value, 10)
                  const clamped = Number.isFinite(parsed) ? Math.max(1, Math.min(parsed, MAX_RANGE_LIMIT)) : undefined
                  patchOptions({ [key]: clamped })
                }}
                placeholder={options.scope === 'range' ? String(rangeLimit) : 'No limit'}
                className="w-32 px-3 py-1.5 text-sm border border-border-strong rounded-md focus:ring-border-focus focus:border-border-focus"
                min={1}
                data-testid="export-limit"
              />
              <span className="text-sm text-content-tertiary">messages</span>
            </div>
          </div>
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between px-6 py-4 border-t bg-surface-secondary rounded-b-lg">
          <div className="text-sm text-content-secondary" data-testid="export-status">
            {options.scope === 'range' ? (
              rangeProgress != null ? (
                <span>
                  Fetched <strong>{formatCount(rangeProgress)}</strong> messages…
                </span>
              ) : (
                <span>
                  Up to <strong>{formatCount(rangeLimit)}</strong> messages
                </span>
              )
            ) : (
              <span>
                Exporting <strong>{exportCount}</strong> messages
              </span>
            )}
          </div>
          <div className="flex items-center gap-2">
            {isExporting && options.scope === 'range' ? (
              <button
                onClick={cancelRange}
                data-testid="export-cancel"
                className="px-4 py-2 text-sm font-medium text-gray-700 hover:bg-surface-tertiary rounded-md"
              >
                Cancel
              </button>
            ) : (
              <button
                onClick={onClose}
                className="px-4 py-2 text-sm font-medium text-gray-700 hover:bg-surface-tertiary rounded-md"
              >
                Cancel
              </button>
            )}
            <button
              onClick={handleExport}
              disabled={isExporting || (options.scope !== 'range' && exportCount === 0)}
              data-testid="export-confirm"
              className="px-4 py-2 text-sm font-medium text-content-inverse bg-accent hover:bg-accent-hover rounded-md disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
            >
              {isExporting ? (
                <>
                  <svg className="animate-spin h-4 w-4" fill="none" viewBox="0 0 24 24" aria-hidden="true">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                  </svg>
                  Exporting...
                </>
              ) : (
                <>
                  <DownloadIcon className="w-4 h-4" />
                  Export
                </>
              )}
            </button>
          </div>
        </div>
      </>
    </Modal>
  )
}
