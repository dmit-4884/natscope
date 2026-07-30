import { useState, useMemo, useEffect, useCallback } from 'react'
import { usePublishHistory, type PublishHistoryEntry } from '@/contexts/messages'
import JsonTreeViewer from '@/components/messages/JsonTreeViewer'
import { CopyIcon, SearchIcon } from '@/components/ui'
import Tooltip from '@/components/common/Tooltip'
import { formatBytes, formatMonthDay, formatTime } from '@/utils/formatters'
import { toast } from '@/utils/toast'
import { copyText } from '@/utils/clipboard'
import { setLastPattern, setPatternDraft } from '@/stores/streamTabState/publishDraftStore'
import { extractWildcardValues, countWildcards } from './publish/subjectPatternUtils'

// Fetched unfiltered (records span ALL streams) then narrowed client-side;
// large page so the per-stream filter has enough rows.
const HISTORY_PAGE_SIZE = 500

interface PublishHistoryProps {
  streamName: string | null
  connectionId: string | null
  connectionUrl: string | null
  /** Current stream's subject patterns — used to attribute failed publishes. */
  subjects?: string[]
}

function parsePayloadJson(payloadJson: string): Record<string, unknown> {
  try {
    return JSON.parse(payloadJson)
  } catch {
    return { _raw: payloadJson }
  }
}

function EntryDetails({
  entry,
  onLoadIntoForm,
}: {
  entry: PublishHistoryEntry
  onLoadIntoForm: ((entry: PublishHistoryEntry) => void) | null
}) {
  const handleCopy = async () => {
    let pretty = entry.payload_json
    try {
      pretty = JSON.stringify(JSON.parse(entry.payload_json), null, 2)
    } catch {
      /* keep raw */
    }
    await copyText(pretty, 'Payload copied to clipboard')
  }

  return (
    <div className="bg-slate-50 border-l-2 border-l-blue-500 p-2 space-y-2">
      {!entry.success && entry.error && (
        <div
          className="px-2 py-1.5 bg-status-error-bg border border-red-200 rounded text-xs text-red-700 break-words"
          data-testid="history-entry-error"
        >
          <span className="font-semibold">Failed:</span> {entry.error}
        </div>
      )}
      <div className="flex items-center gap-2">
        {onLoadIntoForm && (
          <Tooltip content="Fill the publish form with this entry's subject and payload">
            <button
              onClick={() => onLoadIntoForm(entry)}
              className="flex items-center gap-1 px-2 py-1 text-xs font-medium text-accent-text bg-accent-light hover:bg-accent-muted rounded transition-colors"
              data-testid="history-load-into-form"
            >
              <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 16l-4-4m0 0l4-4m-4 4h14m-5 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h7a3 3 0 013 3v1" />
              </svg>
              Load into form
            </button>
          </Tooltip>
        )}
        <button
          onClick={handleCopy}
          className="flex items-center gap-1 px-2 py-1 text-xs font-medium text-content-secondary bg-surface-tertiary hover:bg-surface-hover rounded transition-colors"
          data-testid="history-copy-payload"
        >
          <CopyIcon className="w-3.5 h-3.5" />
          Copy payload
        </button>
        {entry.success && entry.sequence != null && (
          <span className="ml-auto text-xs text-content-muted font-mono" title="Stream sequence">
            seq {entry.sequence}
          </span>
        )}
      </div>
      <JsonTreeViewer
        data={parsePayloadJson(entry.payload_json)}
        title=""
        defaultExpanded={true}
        searchable={false}
      />
    </div>
  )
}

export default function PublishHistory({ streamName, connectionId, connectionUrl, subjects = [] }: PublishHistoryProps) {
  const [selectedEntry, setSelectedEntry] = useState<PublishHistoryEntry | null>(null)
  const [searchTerm, setSearchTerm] = useState('')
  const [showAllStreams, setShowAllStreams] = useState(!streamName)

  useEffect(() => {
    if (!streamName) {
      setShowAllStreams(true)
    }
  }, [streamName])

  const { data: history = [], isLoading } = usePublishHistory(connectionId, connectionUrl || undefined, {
    pageSize: HISTORY_PAGE_SIZE,
  })

  const streamHistory = useMemo(() => {
    if (showAllStreams || !streamName) return history
    return history.filter(
      (entry) =>
        entry.stream === streamName ||
        (!entry.success && !entry.stream && !!entry.subject_pattern && subjects.includes(entry.subject_pattern)),
    )
  }, [history, showAllStreams, streamName, subjects])

  const filteredHistory = useMemo(() => {
    if (!searchTerm) return streamHistory
    const term = searchTerm.toLowerCase()
    return streamHistory.filter(
      (entry) =>
        entry.subject.toLowerCase().includes(term) ||
        (entry.message_type || '').toLowerCase().includes(term) ||
        entry.payload_json.toLowerCase().includes(term)
    )
  }, [streamHistory, searchTerm])

  // Writes into the publish draft store; only patterns from the open stream
  // persist — others get dropped by StreamView's stale-pattern cleanup.
  const loadIntoForm = useCallback(
    (entry: PublishHistoryEntry) => {
      if (!entry.subject_pattern || !streamName || !connectionUrl) return
      const scope = { connectionUrl, streamName }
      let body = entry.payload_json
      try {
        body = JSON.stringify(JSON.parse(entry.payload_json), null, 2)
      } catch {
        /* keep raw */
      }
      const wildcards = countWildcards(entry.subject_pattern)
        ? extractWildcardValues(entry.subject_pattern, entry.subject)
        : []
      setLastPattern(scope, entry.subject_pattern)
      setPatternDraft(scope, entry.subject_pattern, { messageJson: body, wildcards })
      toast.success('Loaded into publish form')
    },
    [streamName, connectionUrl],
  )

  // Mirror the list's attribution rule (failed publishes match by subject
  // pattern) so "Load into form" shows for them.
  const canLoadIntoForm = useCallback(
    (entry: PublishHistoryEntry) =>
      !!entry.subject_pattern &&
      !!streamName &&
      !!connectionUrl &&
      (entry.stream === streamName ||
        (!entry.success && !entry.stream && subjects.includes(entry.subject_pattern))),
    [streamName, connectionUrl, subjects],
  )

  return (
    <div className="h-full flex flex-col">
      {/* Header */}
      <div className="p-4 border-b bg-surface-primary shrink-0">
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-sm font-semibold text-content-primary">Publish History</h3>
        </div>

        {/* Stream filter toggle */}
        <div className="flex items-center gap-2 mb-3">
          <button
            onClick={() => setShowAllStreams(false)}
            disabled={!streamName}
            className={`px-2 py-1 text-xs rounded transition-colors ${
              !showAllStreams && streamName
                ? 'bg-accent-muted text-accent-text'
                : 'text-content-tertiary hover:text-gray-700'
            } ${!streamName ? 'opacity-50 cursor-not-allowed' : ''}`}
          >
            Current Stream
          </button>
          <button
            onClick={() => setShowAllStreams(true)}
            className={`px-2 py-1 text-xs rounded transition-colors ${
              showAllStreams
                ? 'bg-accent-muted text-accent-text'
                : 'text-content-tertiary hover:text-gray-700'
            }`}
          >
            All Streams
          </button>
        </div>

        {/* Search */}
        <div className="relative">
          <input
            type="text"
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            placeholder="Search history..."
            className="w-full px-3 py-1.5 text-sm border border-border-strong rounded-md focus:ring-border-focus focus:border-border-focus pl-8"
          />
          <SearchIcon className="w-4 h-4 absolute left-2.5 top-1/2 -translate-y-1/2 text-content-muted" />
        </div>
      </div>

      {/* History List */}
      <div className="flex-1 overflow-auto min-h-0">
        {isLoading ? (
          <div className="p-4 text-center text-content-tertiary text-sm">Loading...</div>
        ) : filteredHistory.length === 0 ? (
          <div className="p-4 text-center text-content-tertiary text-sm">
            {searchTerm ? 'No matching entries' : 'No publish history yet'}
          </div>
        ) : (
          <div className="divide-y">
            {filteredHistory.map((entry) => (
              <div key={entry.id}>
                {/* Entry row */}
                <button
                  type="button"
                  aria-expanded={selectedEntry?.id === entry.id}
                  onClick={() => setSelectedEntry(selectedEntry?.id === entry.id ? null : entry)}
                  className={`w-full text-left p-3 cursor-pointer hover:bg-surface-secondary transition-colors ${
                    selectedEntry?.id === entry.id ? 'bg-accent-light border-l-2 border-l-blue-500' : ''
                  }`}
                >
                  <div className="flex items-start justify-between gap-2">
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 mb-1">
                        <span
                          role="img"
                          aria-label={entry.success ? 'Success' : 'Failed'}
                          className={`w-2 h-2 rounded-full ${entry.success ? 'bg-green-400' : 'bg-red-400'}`}
                        />
                        <span className="text-xs text-content-tertiary">
                          {formatMonthDay(entry.created_at)} {formatTime(entry.created_at)}
                        </span>
                      </div>
                      <span
                        className="block text-sm font-medium text-content-primary truncate"
                        title={entry.subject}
                      >
                        {entry.subject}
                      </span>
                      <div className="flex items-center gap-2 text-xs text-content-tertiary">
                        {showAllStreams && (
                          <span className="px-1.5 py-0.5 bg-surface-tertiary rounded font-mono">
                            {entry.stream}
                          </span>
                        )}
                        <span className={`px-1.5 py-0.5 rounded text-2xs font-medium ${
                          entry.encoding_type === 'protobuf'
                            ? 'bg-accent-muted text-accent-text'
                            : 'bg-status-warning-light text-amber-700'
                        }`}>
                          {entry.encoding_type === 'protobuf' ? 'Protobuf' : 'JSON'}
                        </span>
                        {entry.message_type && (
                          <span className="truncate font-mono" title={entry.message_type}>
                            {entry.message_type}
                          </span>
                        )}
                      </div>
                    </div>
                    <span className="text-xs text-content-muted">{formatBytes(entry.payload_size)}</span>
                  </div>
                </button>

                {/* Inline details - shown under the selected message */}
                {selectedEntry?.id === entry.id && (
                  <EntryDetails entry={entry} onLoadIntoForm={canLoadIntoForm(entry) ? loadIntoForm : null} />
                )}
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
