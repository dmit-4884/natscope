import { useState, useMemo, useRef, useEffect } from 'react'
import type { Message } from '@/types/nats'
import { Modal, DownloadIcon } from '@/components/ui'
import { decodeBase64ToUtf8 } from '@/utils/base64'
import { formatCount } from '@/utils/formatters'
import { getErrorMessage } from '@/api/errors'
import { getMessages } from '@/api/messages'
import { useMessagesPolicy } from '@/contexts/settings'
import { toast } from '@/utils/toast'
import { collectRange } from './exportRange'

type ExportFormat = 'json' | 'ndjson' | 'csv'
type ExportScope = 'filtered' | 'all' | 'range'

// Range-walk page size; matches backend DefaultMaxMessageLimit (500) — higher
// is clamped server-side.
const RANGE_PAGE_SIZE = 500

interface ExportOptions {
  format: ExportFormat
  scope: ExportScope
  includeMetadata: boolean
  includeHeaders: boolean
  includeDecoded: boolean
  limit?: number
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
}: ExportDialogProps) {
  const msgPolicy = useMessagesPolicy()
  const [options, setOptions] = useState<ExportOptions>(() => ({
    format: msgPolicy.defaultExportFormat,
    scope: 'filtered',
    includeMetadata: true,
    includeHeaders: false,
    includeDecoded: true,
    limit: 1000,
  }))
  const [isExporting, setIsExporting] = useState(false)
  // Range-export progress: number fetched so far, plus a cancel handle.
  const [rangeProgress, setRangeProgress] = useState<number | null>(null)
  const abortRef = useRef<AbortController | null>(null)

  // Abort any in-flight range export on unmount.
  useEffect(() => () => abortRef.current?.abort(), [])

  const rangeLimit = options.scope === 'range' ? (options.limit || msgPolicy.exportRangeLimit) : options.limit

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

  const formatMessage = (msg: Message, includeMetadata: boolean, includeHeaders: boolean, includeDecoded: boolean) => {
    if (!includeMetadata) {
      // Payload only — prefer decoded.
      if (includeDecoded && msg.decoded) {
        return msg.decoded
      }
      try {
        return JSON.parse(decodeBase64ToUtf8(msg.data_base64))
      } catch {
        return msg.data_base64
      }
    }

    const formatted: Record<string, unknown> = {
      sequence: msg.sequence,
      subject: msg.subject,
      timestamp: msg.timestamp,
    }

    if (includeHeaders && msg.headers) {
      formatted.headers = msg.headers
    }

    if (includeDecoded && msg.decoded) {
      formatted.data = msg.decoded
      formatted.decoded_type = msg.decoded_type
    } else {
      try {
        formatted.data = JSON.parse(decodeBase64ToUtf8(msg.data_base64))
      } catch {
        formatted.data = msg.data_base64
      }
    }

    return formatted
  }

  const serializeMessages = (msgs: Message[]) => {
    const formatted = msgs.map((msg) =>
      formatMessage(msg, options.includeMetadata, options.includeHeaders, options.includeDecoded)
    )

    switch (options.format) {
      case 'json':
        return JSON.stringify(formatted, null, 2)

      case 'ndjson':
        return formatted.map((msg) => JSON.stringify(msg)).join('\n')

      case 'csv': {
        if (formatted.length === 0) return ''

        // Build CSV headers
        const headers: string[] = []

        if (options.includeMetadata) {
          headers.push('sequence', 'subject', 'timestamp')
          if (options.includeHeaders) {
            headers.push('headers')
          }
        }
        headers.push('data')

        // Build rows
        const rows = formatted.map((msg) => {
          const row: string[] = []

          if (options.includeMetadata) {
            const typedMsg = msg as Record<string, unknown>
            row.push(String(typedMsg.sequence || ''))
            row.push(escapeCsvValue(String(typedMsg.subject || '')))
            row.push(String(typedMsg.timestamp || ''))
            if (options.includeHeaders) {
              row.push(escapeCsvValue(JSON.stringify(typedMsg.headers || {})))
            }
            row.push(escapeCsvValue(JSON.stringify(typedMsg.data)))
          } else {
            row.push(escapeCsvValue(JSON.stringify(msg)))
          }

          return row.join(',')
        })

        return [headers.join(','), ...rows].join('\n')
      }

      default:
        return ''
    }
  }

  const escapeCsvValue = (value: string): string => {
    // Guard against CSV formula injection: a leading =, +, - or @ makes
    // spreadsheet apps evaluate the cell as a formula. Prefix with a single
    // quote to force text.
    const safe = /^[=+\-@]/.test(value) ? `'${value}` : value
    if (safe.includes(',') || safe.includes('"') || safe.includes('\n')) {
      return `"${safe.replace(/"/g, '""')}"`
    }
    return safe
  }

  const download = (content: string) => {
    const mimeTypes: Record<ExportFormat, string> = {
      json: 'application/json',
      ndjson: 'application/x-ndjson',
      csv: 'text/csv',
    }
    const blob = new Blob([content], { type: mimeTypes[options.format] })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${streamName}-messages-${Date.now()}.${options.format === 'ndjson' ? 'ndjson' : options.format}`
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
    const limit = options.limit || msgPolicy.exportRangeLimit
    const controller = new AbortController()
    abortRef.current = controller
    setIsExporting(true)
    setRangeProgress(0)
    try {
      const result = await collectRange(
        (startSeq, signal) =>
          getMessages(
            streamName,
            {
              connection_id: connectionId,
              start_seq: startSeq,
              limit: RANGE_PAGE_SIZE,
              direction: 'forward',
              // Full payloads — preview truncation would corrupt the export.
              max_payload_bytes: 0,
            },
            signal,
          ),
        { startSeq: streamFirstSeq && streamFirstSeq > 0 ? streamFirstSeq : 1, limit, signal: controller.signal, onProgress: setRangeProgress },
      )

      if (result.aborted && result.messages.length === 0) {
        toast.info('Export cancelled')
        return
      }
      download(serializeMessages(result.messages))
      if (result.aborted) {
        // Cancelled mid-export with partial data — warn so the user knows the
        // file is incomplete.
        toast.warning(
          `Export cancelled — ${formatCount(result.messages.length)} of ${formatCount(limit)} messages saved`,
        )
      } else if (result.truncated) {
        toast.warning(`Exported first ${formatCount(result.messages.length)} messages (limit reached)`) // explicit truncation
      } else {
        toast.success(`Exported ${formatCount(result.messages.length)} messages`)
      }
      onClose()
    } catch (error) {
      if (controller.signal.aborted) {
        toast.info('Export cancelled')
      } else {
        toast.error(`Export failed: ${getErrorMessage(error)}`)
      }
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
      download(serializeMessages(messagesToExport.slice(0, limit)))
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
                  onClick={() => setOptions({ ...options, format })}
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
                  onChange={() => setOptions({ ...options, scope: 'filtered' })}
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
                  onChange={() => setOptions({ ...options, scope: 'all' })}
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
                  onChange={() => setOptions({ ...options, scope: 'range', limit: msgPolicy.exportRangeLimit })}
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
              <p className="mt-1.5 text-xs text-content-tertiary">
                Walks the stream server-side with full payloads, up to the limit below. Larger
                exports may take a while.
              </p>
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
                  onChange={(e) => setOptions({ ...options, includeMetadata: e.target.checked })}
                  className="text-accent focus:ring-border-focus rounded"
                />
                <span className="text-sm text-gray-700">Include metadata (sequence, subject, timestamp)</span>
              </label>
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={options.includeHeaders}
                  onChange={(e) => setOptions({ ...options, includeHeaders: e.target.checked })}
                  className="text-accent focus:ring-border-focus rounded"
                />
                <span className="text-sm text-gray-700">Include message headers</span>
              </label>
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={options.includeDecoded}
                  onChange={(e) => setOptions({ ...options, includeDecoded: e.target.checked })}
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
                value={(options.scope === 'range' ? rangeLimit : options.limit) || ''}
                onChange={(e) => {
                  if (!e.target.value) {
                    setOptions({ ...options, limit: undefined })
                    return
                  }
                  // Clamp to [1, 10_000_000] — zero/negative would silently
                  // drop data in slice/collectRange.
                  const parsed = parseInt(e.target.value, 10)
                  const clamped = Number.isFinite(parsed) ? Math.max(1, Math.min(parsed, 10_000_000)) : undefined
                  setOptions({ ...options, limit: clamped })
                }}
                placeholder={options.scope === 'range' ? String(msgPolicy.exportRangeLimit) : 'No limit'}
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
                  Up to <strong>{formatCount(rangeLimit || msgPolicy.exportRangeLimit)}</strong> messages
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
