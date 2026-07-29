import { useCallback, useEffect, useRef, useState } from 'react'
import { getMessages } from '@/api/messages'
import type { SelectedMessage } from '@/types/messages'
import type { NavQuery } from '@/stores/streamTabState/messagesViewStore'
import { toast } from '@/utils/toast'
import { getErrorMessage } from '@/api/errors'
import { resolveNavTarget, type VisualDirection } from './navTarget'
import { toSelectedHistoryMessage } from './selectedMessage'

export interface UseMessageNavigationOptions {
  streamName: string | null
  connectionId: string | null
  selectedMessage: SelectedMessage | null
  /** Query context published by UnifiedMessageList; undefined until the
   *  list has mounted once for this scope — fall back to backward/unfiltered. */
  navQuery: NavQuery | undefined
  onSelectMessage: (msg: SelectedMessage | null) => void
  /** Attach the document-level ArrowUp/ArrowDown listener (viewer visible). */
  keyboardEnabled: boolean
}

export interface MessageNavigation {
  goPrev: () => void
  goNext: () => void
  /** False when the selected message has no stream sequence (live core_nats). */
  canNavigate: boolean
  prevDisabled: boolean
  nextDisabled: boolean
  prevLoading: boolean
  nextLoading: boolean
}

/**
 * Pure-API prev/next stepping from the detail panel: one
 * listMessages(limit=1) per press, mapped from visual direction to sequence
 * by the list's direction. The loaded list is never consulted, so
 * navigation works past page boundaries and identically for both fetch
 * policies (direct/consumer — a server-side concern).
 */
export function useMessageNavigation({
  streamName,
  connectionId,
  selectedMessage,
  navQuery,
  onSelectMessage,
  keyboardEnabled,
}: UseMessageNavigationOptions): MessageNavigation {
  const [loadingDir, setLoadingDir] = useState<VisualDirection | null>(null)
  const [edges, setEdges] = useState<Record<VisualDirection, boolean>>({ up: false, down: false })

  const sequence = selectedMessage?.sequence
  const canNavigate = !!streamName && !!connectionId && sequence != null && sequence > 0

  // Edge knowledge belongs to the message it was learned on.
  const selectedId = selectedMessage?.id ?? null
  useEffect(() => {
    setEdges({ up: false, down: false })
  }, [selectedId])

  // Fresh selected id for the stale-response guard: a row click during an
  // in-flight nav request must win over the request's result.
  const selectedIdRef = useRef(selectedId)
  selectedIdRef.current = selectedId

  // Ref (not state) so rapid presses in the same tick are still blocked.
  const inFlightRef = useRef(false)

  const step = useCallback(
    async (visual: VisualDirection) => {
      if (!canNavigate || inFlightRef.current) return
      const target = resolveNavTarget(visual, navQuery?.direction ?? 'backward', sequence!)
      if (target === 'edge') {
        setEdges((prev) => ({ ...prev, [visual]: true }))
        return
      }
      const idAtCall = selectedIdRef.current
      inFlightRef.current = true
      setLoadingDir(visual)
      try {
        const res = await getMessages(streamName!, {
          connection_id: connectionId!,
          start_seq: target.startSeq,
          direction: target.apiDirection,
          limit: 1,
          subject_filter: navQuery?.subjectFilter,
          content_filter: navQuery?.contentFilter,
        })
        if (selectedIdRef.current !== idAtCall) return // selection moved on
        if (res.messages.length === 0) {
          setEdges((prev) => ({ ...prev, [visual]: true }))
          return
        }
        onSelectMessage(toSelectedHistoryMessage(res.messages[0]))
      } catch (err) {
        if (selectedIdRef.current === idAtCall) toast.error(getErrorMessage(err))
      } finally {
        inFlightRef.current = false
        setLoadingDir(null)
      }
    },
    [canNavigate, navQuery, sequence, streamName, connectionId, onSelectMessage],
  )

  const goPrev = useCallback(() => void step('up'), [step])
  const goNext = useCallback(() => void step('down'), [step])

  useEffect(() => {
    if (!keyboardEnabled || !canNavigate) return
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key !== 'ArrowUp' && e.key !== 'ArrowDown') return
      if (e.metaKey || e.ctrlKey || e.altKey || e.shiftKey) return
      // Editable elements keep their native behaviour; the message grid keeps
      // its own roving-focus arrows; dialogs keep arrows to themselves.
      const target = e.target instanceof HTMLElement ? e.target : null
      if (target?.closest('input, textarea, select, [contenteditable="true"], [role="grid"], [role="dialog"]')) {
        return
      }
      e.preventDefault()
      void step(e.key === 'ArrowUp' ? 'up' : 'down')
    }
    document.addEventListener('keydown', onKeyDown)
    return () => document.removeEventListener('keydown', onKeyDown)
  }, [keyboardEnabled, canNavigate, step])

  return {
    goPrev,
    goNext,
    canNavigate,
    prevDisabled: !canNavigate || edges.up || loadingDir !== null,
    nextDisabled: !canNavigate || edges.down || loadingDir !== null,
    prevLoading: loadingDir === 'up',
    nextLoading: loadingDir === 'down',
  }
}
