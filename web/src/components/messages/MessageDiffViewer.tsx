import { useState, useMemo, useEffect, useCallback, useRef } from 'react'
import { diffJson, diffLines, type Change } from 'diff'
import { type Message } from '@/types/nats'
import { useStreamMessage } from '@/contexts/messages'
import { QueryErrorState, Spinner } from '@/components/ui'
import { useDialogA11y } from '@/hooks/useDialogA11y'
import { decodeBase64ToUtf8 } from '@/utils/base64'
import { formatTimestamp } from '@/utils/formatters'
import { usePreferencesStore } from '@/stores/preferencesStore'

interface MessageDiffViewerProps {
  messageA: Message | null
  messageB: Message | null
  connectionId?: string
  streamName?: string
  onClose: () => void
}

type DiffMode = 'unified' | 'side-by-side'

function getPayloadString(message: Message): string {
  if (message.decoded) {
    return JSON.stringify(message.decoded, null, 2)
  }

  if (message.data_base64) {
    try {
      const decoded = decodeBase64ToUtf8(message.data_base64)
      // Pretty-print if JSON
      try {
        return JSON.stringify(JSON.parse(decoded), null, 2)
      } catch {
        return decoded
      }
    } catch {
      return message.data_base64
    }
  }

  return ''
}

interface DiffLine {
  type: 'added' | 'removed' | 'unchanged'
  content: string
  oldLineNum?: number
  newLineNum?: number
}

function computeDiffLines(changes: Change[]): DiffLine[] {
  const lines: DiffLine[] = []
  let oldLine = 1
  let newLine = 1

  for (const change of changes) {
    const content = change.value.endsWith('\n')
      ? change.value.slice(0, -1)
      : change.value
    const splitLines = content.split('\n')

    for (const line of splitLines) {
      if (change.removed) {
        lines.push({ type: 'removed', content: line, oldLineNum: oldLine++ })
      } else if (change.added) {
        lines.push({ type: 'added', content: line, newLineNum: newLine++ })
      } else {
        lines.push({ type: 'unchanged', content: line, oldLineNum: oldLine++, newLineNum: newLine++ })
      }
    }
  }

  return lines
}

function computeSideBySideLines(changes: Change[]): { left: DiffLine[]; right: DiffLine[] } {
  const left: DiffLine[] = []
  const right: DiffLine[] = []
  let oldLine = 1
  let newLine = 1

  for (const change of changes) {
    const content = change.value.endsWith('\n')
      ? change.value.slice(0, -1)
      : change.value
    const splitLines = content.split('\n')

    if (change.removed) {
      for (const line of splitLines) {
        left.push({ type: 'removed', content: line, oldLineNum: oldLine++ })
        right.push({ type: 'removed', content: '', newLineNum: undefined })
      }
    } else if (change.added) {
      for (const line of splitLines) {
        left.push({ type: 'added', content: '', oldLineNum: undefined })
        right.push({ type: 'added', content: line, newLineNum: newLine++ })
      }
    } else {
      for (const line of splitLines) {
        left.push({ type: 'unchanged', content: line, oldLineNum: oldLine++ })
        right.push({ type: 'unchanged', content: line, newLineNum: newLine++ })
      }
    }
  }

  return { left, right }
}

function DiffBody({ changes, mode, labelA, labelB, emptyText }: { changes: Change[]; mode: DiffMode; labelA: string; labelB: string; emptyText: string }) {
  const unifiedLines = useMemo(() => computeDiffLines(changes), [changes])
  const sideBySide = useMemo(() => computeSideBySideLines(changes), [changes])
  const hasChanges = changes.some((c) => c.added || c.removed)

  return !hasChanges ? (
      <div className="text-center py-8 text-content-tertiary text-sm">
        {emptyText}
      </div>
    ) : mode === 'unified' ? (
      <div className="border rounded-lg overflow-hidden">
        <div className="max-h-[50vh] overflow-auto">
          {unifiedLines.map((line, idx) => (
            <div
              key={idx}
              className={`flex text-xs font-mono ${
                line.type === 'added'
                  ? 'bg-[#dcfce7]'
                  : line.type === 'removed'
                    ? 'bg-[#fee2e2]'
                    : ''
              }`}
            >
              <div className="w-10 flex-shrink-0 px-2 py-0.5 text-right select-none text-content-muted bg-surface-secondary border-r border-border">
                {line.oldLineNum ?? ''}
              </div>
              <div className="w-10 flex-shrink-0 px-2 py-0.5 text-right select-none text-content-muted bg-surface-secondary border-r border-border">
                {line.newLineNum ?? ''}
              </div>
              <div className="w-5 flex-shrink-0 text-center py-0.5 select-none text-content-tertiary">
                {line.type === 'added' ? '+' : line.type === 'removed' ? '-' : ' '}
              </div>
              <div className="px-2 py-0.5 whitespace-pre flex-1 overflow-x-auto">
                {line.content}
              </div>
            </div>
          ))}
        </div>
      </div>
    ) : (
      <div className="border rounded-lg overflow-hidden">
        <div className="flex max-h-[50vh] overflow-auto">
          {/* Left side (Message A) */}
          <div className="flex-1 border-r border-border overflow-x-auto">
            <div className="text-2xs font-medium text-content-muted px-3 py-1 bg-surface-secondary border-b border-border sticky top-0">
              {labelA}
            </div>
            <div className="min-w-fit">
              {sideBySide.left.map((line, idx) => (
                <div
                  key={idx}
                  className={`flex text-xs font-mono ${
                    line.type === 'added'
                      ? 'bg-status-success-bg/30'
                      : line.type === 'removed'
                        ? 'bg-[#fee2e2]'
                        : ''
                  }`}
                >
                  <div className="w-10 flex-shrink-0 px-2 py-0.5 text-right select-none text-content-muted bg-surface-secondary border-r border-border">
                    {line.oldLineNum ?? ''}
                  </div>
                  <div className="w-5 flex-shrink-0 text-center py-0.5 select-none text-content-tertiary">
                    {line.type === 'removed' ? '-' : ''}
                  </div>
                  <div className="px-2 py-0.5 whitespace-pre flex-1">
                    {line.content}
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* Right side (Message B) */}
          <div className="flex-1 overflow-x-auto">
            <div className="text-2xs font-medium text-content-muted px-3 py-1 bg-surface-secondary border-b border-border sticky top-0">
              {labelB}
            </div>
            <div className="min-w-fit">
              {sideBySide.right.map((line, idx) => (
                <div
                  key={idx}
                  className={`flex text-xs font-mono ${
                    line.type === 'added'
                      ? 'bg-[#dcfce7]'
                      : line.type === 'removed'
                        ? 'bg-status-error-bg/30'
                        : ''
                  }`}
                >
                  <div className="w-10 flex-shrink-0 px-2 py-0.5 text-right select-none text-content-muted bg-surface-secondary border-r border-border">
                    {line.newLineNum ?? ''}
                  </div>
                  <div className="w-5 flex-shrink-0 text-center py-0.5 select-none text-content-tertiary">
                    {line.type === 'added' ? '+' : ''}
                  </div>
                  <div className="px-2 py-0.5 whitespace-pre flex-1">
                    {line.content}
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    )
}

export default function MessageDiffViewer({
  messageA: listA,
  messageB: listB,
  connectionId,
  streamName,
  onClose,
}: MessageDiffViewerProps) {
  const [mode, setMode] = useState<DiffMode>('unified')
  const canFetch = !!connectionId && !!streamName
  const fullA = useStreamMessage(connectionId ?? null, streamName ?? null, listA?.truncated && canFetch ? listA.sequence : null)
  const fullB = useStreamMessage(connectionId ?? null, streamName ?? null, listB?.truncated && canFetch ? listB.sequence : null)
  const wantsFull = (list: Message | null) => !!list?.truncated && canFetch
  const messageA = wantsFull(listA) ? (fullA.data ?? null) : listA
  const messageB = wantsFull(listB) ? (fullB.data ?? null) : listB
  const loadError = (wantsFull(listA) && fullA.error) || (wantsFull(listB) && fullB.error) || null
  const loadingFull = !loadError && ((wantsFull(listA) && !fullA.data) || (wantsFull(listB) && !fullB.data))
  const stillClipped = !canFetch && (!!listA?.truncated || !!listB?.truncated)
  const retryFull = () => {
    if (fullA.error) void fullA.refetch()
    if (fullB.error) void fullB.refetch()
  }

  // Resizable dialog state — persisted in preferences
  const getPanelSize = usePreferencesStore((s) => s.getPanelSize)
  const setPanelSize = usePreferencesStore((s) => s.setPanelSize)

  const [dialogSize, setDialogSize] = useState(() => ({
    width: getPanelSize('diff-width', Math.min(1200, window.innerWidth - 80)),
    height: getPanelSize('diff-height', window.innerHeight - 80),
  }))
  const { dialogRef, onKeyDown } = useDialogA11y<HTMLDivElement>(true, onClose)
  const isResizing = useRef(false)
  const resizeEdge = useRef<string>('')
  // Track the active resize listeners so they're removed if the dialog unmounts
  // mid-drag (handleMouseUp only fires on a completed drag).
  const activeResizeListeners = useRef<{ move: (e: MouseEvent) => void; up: () => void } | null>(null)
  useEffect(
    () => () => {
      if (activeResizeListeners.current) {
        document.removeEventListener('mousemove', activeResizeListeners.current.move)
        document.removeEventListener('mouseup', activeResizeListeners.current.up)
      }
    },
    [],
  )

  const handleResizeStart = useCallback((e: React.MouseEvent, edge: string) => {
    e.preventDefault()
    e.stopPropagation()
    isResizing.current = true
    resizeEdge.current = edge
    const startX = e.clientX
    const startY = e.clientY
    const startW = dialogSize.width
    const startH = dialogSize.height
    let lastSize = { width: startW, height: startH }

    const handleMouseMove = (ev: MouseEvent) => {
      const dx = ev.clientX - startX
      const dy = ev.clientY - startY
      const newSize = { ...dialogSize }

      if (edge.includes('e')) newSize.width = Math.max(500, Math.min(startW + dx, window.innerWidth - 40))
      if (edge.includes('w')) newSize.width = Math.max(500, Math.min(startW - dx, window.innerWidth - 40))
      if (edge.includes('s')) newSize.height = Math.max(300, Math.min(startH + dy, window.innerHeight - 40))
      if (edge.includes('n')) newSize.height = Math.max(300, Math.min(startH - dy, window.innerHeight - 40))

      lastSize = newSize
      setDialogSize(newSize)
    }

    const handleMouseUp = () => {
      isResizing.current = false
      document.removeEventListener('mousemove', handleMouseMove)
      document.removeEventListener('mouseup', handleMouseUp)
      activeResizeListeners.current = null
      setPanelSize('diff-width', lastSize.width)
      setPanelSize('diff-height', lastSize.height)
    }

    activeResizeListeners.current = { move: handleMouseMove, up: handleMouseUp }
    document.addEventListener('mousemove', handleMouseMove)
    document.addEventListener('mouseup', handleMouseUp)
  }, [dialogSize, setPanelSize])

  const metadataDiff = useMemo(() => {
    if (!listA || !listB) return null

    return {
      sequence: { a: listA.sequence, b: listB.sequence },
      subject: { a: listA.subject, b: listB.subject },
      timestamp: { a: formatTimestamp(listA.timestamp, 'absolute'), b: formatTimestamp(listB.timestamp, 'absolute') },
    }
  }, [listA, listB])

  const hasHeaders = Object.keys(messageA?.headers ?? {}).length + Object.keys(messageB?.headers ?? {}).length > 0
  const headerChanges = useMemo(() => {
    if (!messageA || !messageB) return null
    return diffJson(messageA.headers ?? {}, messageB.headers ?? {})
  }, [messageA, messageB])

  const payloadChanges = useMemo(() => {
    if (!messageA || !messageB) return null

    const payloadA = getPayloadString(messageA)
    const payloadB = getPayloadString(messageB)

    // diffJson when both parse as JSON, else diffLines
    try {
      JSON.parse(payloadA)
      JSON.parse(payloadB)
      return diffJson(JSON.parse(payloadA), JSON.parse(payloadB))
    } catch {
      return diffLines(payloadA, payloadB)
    }
  }, [messageA, messageB])

  if (!listA || !listB) {
    return (
      <div className="fixed inset-0 z-50">
        <div className="fixed inset-0 bg-black/50" onClick={onClose} />
        <div
          ref={dialogRef}
          role="dialog"
          aria-modal="true"
          aria-label="Message diff"
          tabIndex={-1}
          onKeyDown={onKeyDown}
          className="fixed top-[10%] left-1/2 -translate-x-1/2 w-full max-w-2xl bg-surface-primary rounded-xl shadow-xl border border-border overflow-hidden focus:outline-none"
        >
          <div className="flex items-center justify-between px-6 py-4 border-b border-border">
            <h2 className="text-lg font-semibold text-content-primary">Message Diff</h2>
            <button
              onClick={onClose}
              aria-label="Close diff"
              className="text-content-muted hover:text-content-secondary text-xl leading-none"
            >
              &times;
            </button>
          </div>
          <div className="px-6 py-12 text-center text-content-tertiary text-sm">
            Select two messages to compare
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="fixed inset-0 z-50">
      {/* Overlay */}
      <div className="fixed inset-0 bg-black/50" onClick={onClose} />

      {/* Dialog — resizable */}
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-label="Message diff"
        tabIndex={-1}
        onKeyDown={onKeyDown}
        className="fixed bg-surface-primary rounded-xl shadow-xl border border-border overflow-hidden flex flex-col focus:outline-none"
        style={{
          width: `${dialogSize.width}px`,
          height: `${dialogSize.height}px`,
          top: '50%',
          left: '50%',
          transform: 'translate(-50%, -50%)',
        }}
      >
        {/* Resize handles */}
        <div className="absolute top-0 left-0 right-0 h-1.5 cursor-n-resize z-10" onMouseDown={(e) => handleResizeStart(e, 'n')} />
        <div className="absolute bottom-0 left-0 right-0 h-1.5 cursor-s-resize z-10" onMouseDown={(e) => handleResizeStart(e, 's')} />
        <div className="absolute top-0 bottom-0 left-0 w-1.5 cursor-w-resize z-10" onMouseDown={(e) => handleResizeStart(e, 'w')} />
        <div className="absolute top-0 bottom-0 right-0 w-1.5 cursor-e-resize z-10" onMouseDown={(e) => handleResizeStart(e, 'e')} />
        <div className="absolute top-0 left-0 w-3 h-3 cursor-nw-resize z-10" onMouseDown={(e) => handleResizeStart(e, 'nw')} />
        <div className="absolute top-0 right-0 w-3 h-3 cursor-ne-resize z-10" onMouseDown={(e) => handleResizeStart(e, 'ne')} />
        <div className="absolute bottom-0 left-0 w-3 h-3 cursor-sw-resize z-10" onMouseDown={(e) => handleResizeStart(e, 'sw')} />
        <div className="absolute bottom-0 right-0 w-3 h-3 cursor-se-resize z-10" onMouseDown={(e) => handleResizeStart(e, 'se')} />
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-border flex-shrink-0">
          <h2 className="text-lg font-semibold text-content-primary">Message Diff</h2>
          <div className="flex items-center gap-3">
            <div className="flex rounded-lg border border-border overflow-hidden text-xs">
              <button
                onClick={() => setMode('unified')}
                className={`px-3 py-1.5 ${mode === 'unified' ? 'bg-surface-tertiary text-content-primary font-medium' : 'text-content-tertiary hover:bg-surface-secondary'}`}
              >
                Unified
              </button>
              <button
                onClick={() => setMode('side-by-side')}
                className={`px-3 py-1.5 border-l border-border ${mode === 'side-by-side' ? 'bg-surface-tertiary text-content-primary font-medium' : 'text-content-tertiary hover:bg-surface-secondary'}`}
              >
                Side by Side
              </button>
            </div>
            <button
              onClick={onClose}
              aria-label="Close diff"
              className="text-content-muted hover:text-content-secondary text-xl leading-none"
            >
              &times;
            </button>
          </div>
        </div>

        {/* Content */}
        <div className="overflow-y-auto flex-1">
          {/* Metadata comparison */}
          {metadataDiff && (
            <div className="px-6 py-4 border-b border-border">
              <h3 className="text-xs font-medium text-content-tertiary uppercase tracking-wide mb-3">
                Metadata
              </h3>
              <div className="grid grid-cols-[auto_1fr_auto_1fr] gap-x-4 gap-y-2 text-sm items-center">
                {/* Header row */}
                <div className="text-xs text-content-muted"></div>
                <div className="text-xs text-content-muted font-medium">Message A</div>
                <div className="text-xs text-content-muted"></div>
                <div className="text-xs text-content-muted font-medium">Message B</div>

                {/* Sequence */}
                <div className="text-xs text-content-tertiary font-medium">Seq</div>
                <div className="font-mono text-gray-700">{metadataDiff.sequence.a}</div>
                <div className="text-gray-300">&rarr;</div>
                <div className={`font-mono ${metadataDiff.sequence.a !== metadataDiff.sequence.b ? 'text-green-700 font-semibold' : 'text-gray-700'}`}>
                  {metadataDiff.sequence.b}
                </div>

                {/* Subject */}
                <div className="text-xs text-content-tertiary font-medium">Subject</div>
                <div className="font-mono text-xs text-gray-700 truncate" title={metadataDiff.subject.a}>{metadataDiff.subject.a}</div>
                <div className="text-gray-300">&rarr;</div>
                <div className={`font-mono text-xs truncate ${metadataDiff.subject.a !== metadataDiff.subject.b ? 'text-green-700 font-semibold' : 'text-gray-700'}`} title={metadataDiff.subject.b}>
                  {metadataDiff.subject.b}
                </div>

                {/* Timestamp */}
                <div className="text-xs text-content-tertiary font-medium">Time</div>
                <div className="text-xs text-gray-700">{metadataDiff.timestamp.a}</div>
                <div className="text-gray-300">&rarr;</div>
                <div className={`text-xs ${metadataDiff.timestamp.a !== metadataDiff.timestamp.b ? 'text-green-700 font-semibold' : 'text-gray-700'}`}>
                  {metadataDiff.timestamp.b}
                </div>
              </div>
            </div>
          )}

          {loadError ? (
            <QueryErrorState error={loadError} title="Failed to load the full messages" onRetry={retryFull} />
          ) : loadingFull ? (
            <div className="flex items-center justify-center gap-2 py-10 text-sm text-content-tertiary">
              <Spinner />
              <span>Loading the full messages…</span>
            </div>
          ) : (
            <>
              {stillClipped && (
                <p className="px-6 pt-4 text-xs text-status-warning-text">
                  The list shows clipped payloads; this diff covers only the part that was loaded.
                </p>
              )}
              {hasHeaders && (
              <div className="px-6 py-4 border-b border-border">
                <h3 className="text-xs font-medium text-content-tertiary uppercase tracking-wide mb-3">Headers</h3>
                <DiffBody
                  changes={headerChanges ?? []}
                  mode={mode}
                  labelA={`Message A (seq ${listA.sequence})`}
                  labelB={`Message B (seq ${listB.sequence})`}
                  emptyText="No header differences"
                />
              </div>
              )}
              <div className="px-6 py-4">
                <h3 className="text-xs font-medium text-content-tertiary uppercase tracking-wide mb-3">Payload</h3>
                <DiffBody
                  changes={payloadChanges ?? []}
                  mode={mode}
                  labelA={`Message A (seq ${listA.sequence})`}
                  labelB={`Message B (seq ${listB.sequence})`}
                  emptyText="No payload differences"
                />
              </div>
            </>
          )}
        </div>

        {/* Footer */}
        <div className="border-t border-border px-6 py-3 flex items-center justify-between text-xs text-content-muted flex-shrink-0">
          <span>
            Comparing seq {listA.sequence} with seq {listB.sequence}
          </span>
          <kbd className="px-1.5 py-0.5 bg-surface-tertiary rounded text-2xs font-mono">
            Esc to close
          </kbd>
        </div>
      </div>
    </div>
  )
}
