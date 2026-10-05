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
  resume?: number
}

const IDLE: SearchState = { status: 'idle', messages: [], progress: null, done: null, error: null, carried: 0, range: null }

function applyEvent(prev: SearchState, event: SearchEvent): SearchState {
  switch (event.kind) {
    case 'progress':
      return { ...prev, progress: event.progress, range: { first: event.progress.range_first, last: event.progress.range_last } }
    case 'matches':
      return { ...prev, messages: [...prev.messages, ...event.messages] }
    case 'done':
      return { ...prev, status: 'done', done: event.done, range: { first: event.done.range_first, last: event.done.range_last } }
  }
}

function runScanned(state: SearchState): number {
  return state.done?.scanned ?? state.progress?.scanned ?? 0
}

function resumeAfter(progress: SearchProgress | null, direction: SearchQuery['direction']): number | undefined {
  if (!progress || progress.current_seq === 0) return undefined
  const next = direction === 'backward' ? progress.current_seq - 1 : progress.current_seq + 1
  return next >= progress.range_first && next <= progress.range_last ? next : undefined
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
    const direction = queryKey ? (JSON.parse(queryKey) as SearchQuery).direction : 'backward'
    setState((prev) => (prev.status === 'running' ? { ...prev, status: 'stopped', resume: resumeAfter(prev.progress, direction) } : prev))
  }, [queryKey])

  const cursor = state.status === 'done' ? state.done?.next_seq : state.status === 'stopped' ? state.resume : undefined

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
