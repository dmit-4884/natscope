import { useCallback, useEffect, useRef, useState } from 'react'
import { searchMessages, type SearchDone, type SearchEvent, type SearchProgress } from '@/api/messages'
import type { Message } from '@/types/nats'

export interface SearchQuery {
  direction: 'forward' | 'backward'
  subject_filter?: string
  from_seq?: number
  to_seq?: number
  from_time?: number
  to_time?: number
  text?: string
  regex?: boolean
  header_name?: string
  header_value?: string
}

type SearchStatus = 'idle' | 'running' | 'done' | 'stopped' | 'error'

export interface MessageSearch {
  status: SearchStatus
  messages: Message[]
  progress: SearchProgress | null
  done: SearchDone | null
  error: unknown
  scanned: number
  range: { first: number; last: number } | null
  canContinue: boolean
  more: () => void
  stop: () => void
  restart: () => void
}

interface SearchState {
  status: SearchStatus
  messages: Message[]
  progress: SearchProgress | null
  done: SearchDone | null
  error: unknown
  carried: number
  range: { first: number; last: number } | null
  runCursor?: number
  resume?: number
}

const IDLE: SearchState = { status: 'idle', messages: [], progress: null, done: null, error: null, carried: 0, range: null }

function applyEvent(prev: SearchState, event: SearchEvent): SearchState {
  switch (event.kind) {
    case 'progress':
      return { ...prev, progress: event.progress, range: rangeOf(event.progress.range_first, event.progress.range_last) }
    case 'matches': {
      const seen = new Set(prev.messages.map((m) => m.sequence))
      const fresh = event.messages.filter((m) => !seen.has(m.sequence))
      return fresh.length === 0 ? prev : { ...prev, messages: [...prev.messages, ...fresh] }
    }
    case 'done':
      return { ...prev, status: 'done', done: event.done, range: rangeOf(event.done.range_first, event.done.range_last) }
  }
}

function runScanned(state: SearchState): number {
  return state.done?.scanned ?? state.progress?.scanned ?? 0
}

function rangeOf(first: number, last: number): SearchState['range'] {
  return first > 0 ? { first, last } : null
}

function resumeAfter(state: SearchState): number | undefined {
  return state.progress ? state.progress.resume_seq : state.runCursor
}

export function useMessageSearch(connectionId: string | null, streamName: string | null, query: SearchQuery | null): MessageSearch {
  const [state, setState] = useState<SearchState>(IDLE)
  const controllerRef = useRef<AbortController | null>(null)
  const queryKey = query ? JSON.stringify(query) : null

  const run = useCallback(
    (cursor: number | undefined, append: boolean) => {
      if (!connectionId || !streamName || !queryKey) return
      const current = JSON.parse(queryKey) as SearchQuery
      controllerRef.current?.abort()
      const controller = new AbortController()
      controllerRef.current = controller
      setState((prev) => ({
        status: 'running',
        messages: append ? prev.messages : [],
        progress: null,
        done: null,
        error: null,
        carried: append ? prev.carried + runScanned(prev) : 0,
        range: append ? prev.range : null,
        runCursor: cursor,
      }))
      void (async () => {
        try {
          for await (const event of searchMessages(streamName, { connection_id: connectionId, ...current, cursor_seq: cursor }, controller.signal)) {
            if (controller.signal.aborted) return
            setState((prev) => applyEvent(prev, event))
          }
        } catch (error) {
          if (!controller.signal.aborted) setState((prev) => ({ ...prev, status: 'error', error }))
        }
      })()
    },
    [connectionId, streamName, queryKey],
  )

  useEffect(() => {
    if (!queryKey) {
      controllerRef.current?.abort()
      setState(IDLE)
      return
    }
    run(undefined, false)
    return () => controllerRef.current?.abort()
  }, [queryKey, run])

  const stop = useCallback(() => {
    controllerRef.current?.abort()
    setState((prev) => (prev.status === 'running' ? { ...prev, status: 'stopped', resume: resumeAfter(prev) } : prev))
  }, [])

  const cursor =
    state.status === 'done'
      ? state.done?.next_seq
      : state.status === 'stopped'
        ? state.resume
        : state.status === 'error'
          ? resumeAfter(state)
          : undefined

  const more = useCallback(() => {
    if (cursor != null) run(cursor, true)
  }, [cursor, run])

  const restart = useCallback(() => run(undefined, false), [run])

  return {
    status: state.status,
    messages: state.messages,
    progress: state.progress,
    done: state.done,
    error: state.error,
    scanned: state.carried + runScanned(state),
    range: state.range,
    canContinue: cursor != null,
    more,
    stop,
    restart,
  }
}
