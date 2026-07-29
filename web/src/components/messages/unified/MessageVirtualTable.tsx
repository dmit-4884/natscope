import { memo, useCallback, useEffect, useRef, useState } from 'react'
import { useVirtualizer } from '@tanstack/react-virtual'
import type { Message } from '@/types/nats'
import { formatBytes, formatTimeWithMs, formatTimestamp } from '@/utils/formatters'
import { getSubjectColor } from '@/utils/subjectColors'
import { CheckIcon } from '@/components/ui'
import { BookmarkButton } from '../Bookmarks'
import { getPayloadPreview, type LiveMessage, type ViewMode } from './messageListUtils'

type AnyMessage = Message | LiveMessage

interface RowProps {
  msg: AnyMessage
  index: number
  mode: ViewMode
  isSelected: boolean
  isNew: boolean
  isCompareHighlighted: boolean
  compareMode: boolean
  rowHeight: number
  cellPadding: string
  streamName: string
  connectionId: string | null
  timestampFormat: 'relative' | 'absolute' | 'iso'
  translateY: number
  /**
   * Roving tabindex: exactly one row has tabIndex=0 (Tab enters grid; arrows
   * move within).
   */
  isFocused: boolean
  canPullFocus: () => boolean
  onActivateIndex: (index: number) => void
  onFocusIndex: (index: number) => void
}

// React.memo'd so a row only re-renders on its own data/selection change — else
// the whole viewport repaints on every parent state change.
const MessageRow = memo(function MessageRow({
  msg,
  index,
  mode,
  isSelected,
  isNew,
  isCompareHighlighted,
  compareMode,
  rowHeight,
  cellPadding,
  streamName,
  connectionId,
  timestampFormat,
  translateY,
  isFocused,
  canPullFocus,
  onActivateIndex,
  onFocusIndex,
}: RowProps) {
  const rowRef = useRef<HTMLDivElement | null>(null)
  const isHistory = mode === 'history'
  const subjectColor = getSubjectColor(msg.subject)
  const showBookmark = Boolean(compareMode === false && isHistory && (msg as Message).sequence && connectionId)

  // Only decode the preview when the bookmark button renders — skips
  // base64+JSON for every other row.
  const payloadPreview = showBookmark ? getPayloadPreview(msg.data_base64) : ''

  useEffect(() => {
    if (
      isFocused &&
      rowRef.current &&
      document.activeElement !== rowRef.current &&
      canPullFocus()
    ) {
      rowRef.current.focus({ preventScroll: true })
    }
  }, [isFocused, canPullFocus])

  const handleKeyDown = (e: React.KeyboardEvent<HTMLDivElement>) => {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault()
      onActivateIndex(index)
    }
  }

  return (
    <div
      ref={rowRef}
      role="row"
      aria-selected={isSelected || isCompareHighlighted}
      tabIndex={isFocused ? 0 : -1}
      onFocus={() => onFocusIndex(index)}
      onKeyDown={handleKeyDown}
      className={`flex items-center cursor-pointer transition-all duration-300 border-b border-l-4 outline-none focus-visible:ring-2 focus-visible:ring-border-focus focus-visible:ring-inset ${
        isCompareHighlighted
          ? 'bg-indigo-100 border-l-indigo-500'
          : isSelected
            ? 'bg-accent-muted border-l-blue-500'
            : isNew
              ? 'bg-yellow-100/70 border-l-yellow-400'
              : subjectColor
                ? `${subjectColor.bg} ${subjectColor.border} hover:opacity-80`
                : 'bg-surface-primary border-l-transparent hover:bg-surface-secondary'
      }`}
      style={{
        position: 'absolute',
        top: 0,
        left: 0,
        width: '100%',
        height: `${rowHeight}px`,
        transform: `translateY(${translateY}px)`,
      }}
      onClick={() => onActivateIndex(index)}
    >
      <div className="w-8 flex items-center justify-center" role="gridcell">
        {compareMode && isHistory ? (
          <div
            className={`w-4 h-4 rounded border-2 flex items-center justify-center transition-colors ${
              isCompareHighlighted
                ? 'bg-indigo-500 border-indigo-500'
                : 'border-border-strong hover:border-indigo-400'
            }`}
            aria-hidden="true"
          >
            {isCompareHighlighted && (
              <CheckIcon className="w-3 h-3 text-content-inverse" />
            )}
          </div>
        ) : showBookmark ? (
          <BookmarkButton
            connectionId={connectionId!}
            streamName={streamName}
            sequence={(msg as Message).sequence}
            subject={msg.subject}
            dataPreview={payloadPreview}
          />
        ) : null}
      </div>
      {(isHistory || msg.sequence) && (
        <div className={`${cellPadding} text-sm font-mono text-content-primary w-20`} role="gridcell">
          {msg.sequence}
        </div>
      )}
      <div className={`${cellPadding} text-sm flex-1 truncate ${subjectColor?.text || 'text-content-primary'}`} role="gridcell">
        <span className="font-mono">{msg.subject}</span>
      </div>
      <div className={`${cellPadding} text-sm text-content-secondary w-28 font-mono`} role="gridcell">
        {isHistory
          ? formatTimestamp(msg.timestamp, timestampFormat)
          : formatTimeWithMs(msg.timestamp)}
      </div>
      <div className={`${cellPadding} text-sm text-content-secondary w-24 whitespace-nowrap text-right`} role="gridcell">
        {formatBytes(msg.data_size)}
      </div>
    </div>
  )
})

interface Props {
  messages: AnyMessage[]
  mode: ViewMode
  selectedMessageId?: string | null
  newMessageIds: Set<string>
  compareMode: boolean
  isCompareSelected: (msg: Message) => boolean
  rowHeight: number
  cellPadding: string
  streamName: string
  connectionId: string | null
  timestampFormat: 'relative' | 'absolute' | 'iso'
  autoScrollRef: React.MutableRefObject<boolean>
  onSelectHistory: (msg: Message) => void
  onSelectLive: (msg: LiveMessage) => void
  onCompareSelect: (msg: Message) => void
}

export function MessageVirtualTable({
  messages,
  mode,
  selectedMessageId,
  newMessageIds,
  compareMode,
  isCompareSelected,
  rowHeight,
  cellPadding,
  streamName,
  connectionId,
  timestampFormat,
  autoScrollRef,
  onSelectHistory,
  onSelectLive,
  onCompareSelect,
}: Props) {
  const parentRef = useRef<HTMLDivElement>(null!)
  const prevFirstIdRef = useRef<string | null>(null)
  const [showScrollTop, setShowScrollTop] = useState(false)
  const [focusedId, setFocusedId] = useState<string | null>(null)
  const isHistory = mode === 'history'

  const idOf = useCallback(
    (msg: AnyMessage) =>
      isHistory ? `history-${(msg as Message).sequence}` : (msg as LiveMessage).id,
    [isHistory],
  )

  // Scroll-top indicator + realtime auto-scroll pause
  useEffect(() => {
    const el = parentRef.current
    if (!el) return
    const handler = () => {
      setShowScrollTop(el.scrollTop > 200)
      if (mode === 'realtime') autoScrollRef.current = el.scrollTop < 50
    }
    el.addEventListener('scroll', handler)
    return () => el.removeEventListener('scroll', handler)
  }, [mode, autoScrollRef])

  useEffect(() => {
    if (mode !== 'realtime') return
    const firstId = messages.length > 0 ? idOf(messages[0]) : null
    const changed = firstId !== null && firstId !== prevFirstIdRef.current
    prevFirstIdRef.current = firstId
    if (changed && autoScrollRef.current && parentRef.current) {
      parentRef.current.scrollTo({ top: 0 })
    }
  }, [messages, mode, autoScrollRef, idOf])

  const rowVirtualizer = useVirtualizer({
    count: messages.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => rowHeight,
    overscan: 10,
  })

  const scrolledForIdRef = useRef<string | null | undefined>(null)
  useEffect(() => {
    if (!selectedMessageId || scrolledForIdRef.current === selectedMessageId) return
    const index = messages.findIndex((m) =>
      (isHistory ? `history-${(m as Message).sequence}` : (m as LiveMessage).id) === selectedMessageId,
    )
    if (index < 0) return // selection is past the loaded page — nothing to scroll to
    scrolledForIdRef.current = selectedMessageId
    rowVirtualizer.scrollToIndex(index, { align: 'auto' })
  }, [selectedMessageId, messages, isHistory, rowVirtualizer])

  const scrollToTop = () => parentRef.current?.scrollTo({ top: 0, behavior: 'smooth' })

  const activate = useCallback(
    (msg: AnyMessage) => {
      if (compareMode && isHistory) {
        onCompareSelect(msg as Message)
      } else if (isHistory) {
        onSelectHistory(msg as Message)
      } else {
        onSelectLive(msg as LiveMessage)
      }
    },
    [compareMode, isHistory, onCompareSelect, onSelectHistory, onSelectLive],
  )

  const messagesRef = useRef(messages)
  useEffect(() => {
    messagesRef.current = messages
  }, [messages])

  const canPullFocus = useCallback(
    () => parentRef.current?.contains(document.activeElement) ?? false,
    [],
  )

  const handleFocusIndex = useCallback(
    (index: number) => {
      const msg = messagesRef.current[index]
      if (msg) setFocusedId(idOf(msg))
    },
    [idOf],
  )

  const handleActivateIndex = useCallback(
    (index: number) => {
      const msg = messagesRef.current[index]
      if (!msg) return
      setFocusedId(idOf(msg))
      activate(msg)
    },
    [idOf, activate],
  )

  const focusedIndexRaw = focusedId ? messages.findIndex((m) => idOf(m) === focusedId) : -1
  const safeFocusedIndex = focusedIndexRaw >= 0 ? focusedIndexRaw : 0

  const moveFocus = useCallback(
    (next: number) => {
      const list = messagesRef.current
      if (list.length === 0) return
      const clamped = Math.max(0, Math.min(list.length - 1, next))
      setFocusedId(idOf(list[clamped]))
      // Scroll into view; the row's own useEffect calls .focus() once
      // virtualization mounts it.
      rowVirtualizer.scrollToIndex(clamped, { align: 'auto' })
    },
    [idOf, rowVirtualizer],
  )

  const handleGridKeyDown = (e: React.KeyboardEvent<HTMLDivElement>) => {
    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault()
        moveFocus(safeFocusedIndex + 1)
        break
      case 'ArrowUp':
        e.preventDefault()
        moveFocus(safeFocusedIndex - 1)
        break
      case 'PageDown':
        e.preventDefault()
        moveFocus(safeFocusedIndex + 10)
        break
      case 'PageUp':
        e.preventDefault()
        moveFocus(safeFocusedIndex - 10)
        break
      case 'Home':
        e.preventDefault()
        moveFocus(0)
        break
      case 'End':
        e.preventDefault()
        moveFocus(messages.length - 1)
        break
    }
  }

  return (
    <>
      <div className="bg-surface-secondary flex border-b text-xs font-medium text-content-secondary uppercase" role="row">
        <div className="w-8" />
        <div className="px-3 py-2 w-20" role="columnheader">Seq</div>
        <div className="px-3 py-2 flex-1" role="columnheader">Subject</div>
        <div className="px-3 py-2 w-28" role="columnheader">Received</div>
        <div className="px-3 py-2 w-24 text-right" role="columnheader">Size</div>
      </div>

      <div
        ref={parentRef}
        className="flex-1 min-h-0 overflow-auto relative focus:outline-none"
        role="grid"
        aria-label={`Messages in ${streamName || 'stream'}`}
        aria-rowcount={messages.length}
        onKeyDown={handleGridKeyDown}
      >
        <div style={{ height: `${rowVirtualizer.getTotalSize()}px`, width: '100%', position: 'relative' }}>
          {rowVirtualizer.getVirtualItems().map((virtualItem) => {
            const msg = messages[virtualItem.index]
            const id = isHistory ? `history-${(msg as Message).sequence}` : (msg as LiveMessage).id
            const isSelected = selectedMessageId === id
            const isNew = !isHistory && newMessageIds.has((msg as LiveMessage).id)
            const isCompareHighlighted = compareMode && isHistory && isCompareSelected(msg as Message)
            const isFocused = virtualItem.index === safeFocusedIndex

            return (
              <MessageRow
                key={id}
                msg={msg}
                index={virtualItem.index}
                mode={mode}
                isSelected={isSelected}
                isNew={isNew}
                isCompareHighlighted={isCompareHighlighted}
                compareMode={compareMode}
                rowHeight={virtualItem.size}
                cellPadding={cellPadding}
                streamName={streamName}
                connectionId={connectionId}
                timestampFormat={timestampFormat}
                translateY={virtualItem.start}
                isFocused={isFocused}
                canPullFocus={canPullFocus}
                onActivateIndex={handleActivateIndex}
                onFocusIndex={handleFocusIndex}
              />
            )
          })}
        </div>

        {showScrollTop && (
          <button
            onClick={scrollToTop}
            className="absolute bottom-4 right-4 p-2 bg-surface-primary border border-border-strong rounded-full shadow-lg hover:bg-surface-secondary transition-colors z-10"
            title="Scroll to top"
            aria-label="Scroll to top"
          >
            <svg className="w-5 h-5 text-content-secondary" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 10l7-7m0 0l7 7m-7-7v18" />
            </svg>
          </button>
        )}
      </div>
    </>
  )
}
