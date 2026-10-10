import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import {
  LiveStreamClient,
  useLiveStatsStore,
  useProtoReloadInvalidation,
  type SubjectSessionLimits,
  type WSBatchPayload,
  type WSMessagePayload,
  type WSStatsPayload,
} from '@/contexts/live'
import { usePageVisible } from '@/hooks/usePageVisible'
import { matchSubject } from '@/shared/domain/subjectMatch'
import type { LiveMessage, LiveMessageLimit, WsStatus } from './messageListUtils'

interface Options {
  connectionId: string | null
  streamName: string | null
  subjects?: string[]
  enabled: boolean
  maxDisplayRate?: number
  initialLimit: LiveMessageLimit
  subjectFilter?: string
  globalStats?: boolean
  subjectLimits?: SubjectSessionLimits
  exclude?: string[]
}

export interface LiveSubscription {
  liveMessages: LiveMessage[]
  liveLimit: LiveMessageLimit
  setLiveLimit: (limit: LiveMessageLimit) => void
  wsStatus: WsStatus
  wsError: string | null
  isPaused: boolean
  togglePause: () => void
  newMessageIds: Set<string>
  clearMessages: () => void
  subjectCounts: Record<string, number>
  deniedSubjects: string[]
  msgPerSecond: number | undefined
  messagesDropped: number | undefined
  messagesReceived: number | undefined
  pausedCount: number
}

const MAX_COUNTED_SUBJECTS = 1000

interface Tally {
  carried: number
  session: number
  cleared: number
}

type Tallies = Record<'dropped' | 'received', Tally>

const freshTallies = (): Tallies => ({
  dropped: { carried: 0, session: 0, cleared: 0 },
  received: { carried: 0, session: 0, cleared: 0 },
})

const tallyTotal = (t: Tally) => t.carried + t.session - t.cleared

function carryTallies(t: Tallies): Tallies {
  return {
    dropped: { carried: tallyTotal(t.dropped), session: 0, cleared: 0 },
    received: { carried: tallyTotal(t.received), session: 0, cleared: 0 },
  }
}

function clearTallies(t: Tallies): Tallies {
  return {
    dropped: { carried: 0, session: t.dropped.session, cleared: t.dropped.session },
    received: { carried: 0, session: t.received.session, cleared: t.received.session },
  }
}

function toLiveMessage(msg: WSMessagePayload): LiveMessage {
  const id = `${Date.now()}-${Math.random().toString(36).slice(2, 11)}`
  return {
    ...msg,
    id,
    timestamp: msg.timestamp,
    decoded: msg.decoded ?? undefined,
    decodedType: msg.decoded_type ?? undefined,
    decodedAuto: msg.decoded_auto,
    decodedSourceId: msg.decoded_source_id,
    decodeError: msg.decode_error ?? undefined,
  }
}

/**
 * Live WS lifecycle for a stream subscription: connection, batch processing,
 * FIFO cap, stats, pause/resume.
 */
export function useLiveSubscription({
  connectionId,
  streamName,
  subjects,
  enabled,
  maxDisplayRate,
  initialLimit,
  subjectFilter,
  globalStats = true,
  subjectLimits,
  exclude,
}: Options): LiveSubscription {
  const [liveMessages, setLiveMessages] = useState<LiveMessage[]>([])
  const [liveLimit, setLiveLimit] = useState<LiveMessageLimit>(initialLimit)
  const [wsStatus, setWsStatus] = useState<WsStatus>('disconnected')
  const [natsDown, setNatsDown] = useState(false)
  const [wsError, setWsError] = useState<string | null>(null)
  const [isPaused, setIsPaused] = useState(false)
  const [newMessageIds, setNewMessageIds] = useState<Set<string>>(new Set())
  const [subjectCounts, setSubjectCounts] = useState<Record<string, number>>({})
  const [deniedSubjects, setDeniedSubjects] = useState<string[]>([])
  const [msgPerSecond, setMsgPerSecond] = useState<number | undefined>(undefined)
  const [messagesDropped, setMessagesDropped] = useState<number | undefined>(undefined)
  const [messagesReceived, setMessagesReceived] = useState<number | undefined>(undefined)
  const talliesRef = useRef(freshTallies())
  const [pausedCount, setPausedCount] = useState(0)
  const [feedFor, setFeedFor] = useState(connectionId)
  if (feedFor !== connectionId) {
    setFeedFor(connectionId)
    setLiveMessages([])
    setSubjectCounts({})
    setNewMessageIds(new Set())
    setDeniedSubjects([])
  }

  const visible = usePageVisible()
  const active = enabled && visible
  const [ws, setWs] = useState<LiveStreamClient | null>(null)
  const wsRef = useRef<LiveStreamClient | null>(null)
  useEffect(() => {
    wsRef.current = ws
  }, [ws])
  const liveLimitRef = useRef(liveLimit)
  const streamNameRef = useRef(streamName)
  const subjectsKey = subjects && subjects.length > 0 ? subjects.join('\n') : null
  const subjectsKeyRef = useRef(subjectsKey)
  const subjectFilterRef = useRef(subjectFilter)
  const limitsKey = subjectLimits
    ? `${subjectLimits.maxPayloadBytes ?? ''}|${subjectLimits.maxDisplayRate ?? ''}|${(subjectLimits.exclude ?? []).join('\n')}`
    : ''
  const limitsRef = useRef(subjectLimits)
  const excludeRef = useRef(exclude)
  useEffect(() => {
    limitsRef.current = subjectLimits
  }, [subjectLimits])
  useEffect(() => {
    excludeRef.current = exclude
  }, [exclude])
  const setGlobalStats = useLiveStatsStore((s) => s.setStats)

  const maxDisplayRateRef = useRef(maxDisplayRate)
  const queueRef = useRef<LiveMessage[]>([])
  const dripTimerRef = useRef<ReturnType<typeof setTimeout>>()
  const resumingRef = useRef(false)
  useEffect(() => {
    maxDisplayRateRef.current = maxDisplayRate
  }, [maxDisplayRate])

  // `ws` in state (not ref) so proto-reload re-subscribes once the socket
  // exists; a ref read null forever.
  useProtoReloadInvalidation(ws)

  useEffect(() => {
    liveLimitRef.current = liveLimit
  }, [liveLimit])
  useEffect(() => {
    streamNameRef.current = streamName
  }, [streamName])
  useEffect(() => {
    subjectsKeyRef.current = subjectsKey
    setDeniedSubjects([])
  }, [subjectsKey])
  useEffect(() => {
    subjectFilterRef.current = subjectFilter
  }, [subjectFilter])

  const highlightTimersRef = useRef<Set<ReturnType<typeof setTimeout>>>(new Set())
  const clearHighlightTimers = useCallback(() => {
    highlightTimersRef.current.forEach(clearTimeout)
    highlightTimersRef.current.clear()
  }, [])

  const flush = useCallback((converted: LiveMessage[]) => {
    if (converted.length === 0) return
    const newIds = converted.map((m) => m.id)

    setNewMessageIds((prev) => {
      const next = new Set(prev)
      newIds.forEach((id) => next.add(id))
      return next
    })

    setLiveMessages((prev) => {
      const updated = [...converted, ...prev]
      return updated.slice(0, liveLimitRef.current)
    })

    // Expire only this batch's ids after the highlight window, so the set can't
    // grow without bound under sustained traffic (a debounced clear-all never
    // fires while messages keep arriving).
    const timer = setTimeout(() => {
      highlightTimersRef.current.delete(timer)
      setNewMessageIds((prev) => {
        if (prev.size === 0) return prev
        const next = new Set(prev)
        newIds.forEach((id) => next.delete(id))
        return next
      })
    }, 800)
    highlightTimersRef.current.add(timer)
  }, [])

  const stopDrip = useCallback(() => {
    clearTimeout(dripTimerRef.current)
    dripTimerRef.current = undefined
    queueRef.current = []
  }, [])

  const flushQueue = useCallback(() => {
    const queued = queueRef.current
    stopDrip()
    flush(queued.reverse())
  }, [flush, stopDrip])

  const drip = useCallback(function next() {
    const rate = maxDisplayRateRef.current
    if (!rate || rate <= 0) {
      flushQueue()
      return
    }
    const message = queueRef.current.shift()
    if (!message) {
      dripTimerRef.current = undefined
      return
    }
    flush([message])
    dripTimerRef.current = setTimeout(next, 1000 / rate)
  }, [flush, flushQueue])

  const processBatch = useCallback(
    (batch: WSBatchPayload) => {
      const pattern = subjectFilterRef.current
      const subjectMode = subjectsKeyRef.current !== null
      const muted = excludeRef.current ?? []
      const relevant = batch.messages.filter((msg) => {
        if (!subjectMode && msg.stream_name !== streamNameRef.current) return false
        if (muted.some((m) => matchSubject(msg.subject, m))) return false
        if (!pattern) return true
        if (pattern.includes('*') || pattern.includes('>')) return matchSubject(msg.subject, pattern)
        return msg.subject.toLowerCase().includes(pattern.toLowerCase())
      })
      if (relevant.length === 0) return

      setSubjectCounts((prev) => {
        const next = { ...prev }
        for (const msg of relevant) {
          if (msg.subject in next || Object.keys(next).length < MAX_COUNTED_SUBJECTS) {
            next[msg.subject] = (next[msg.subject] ?? 0) + 1
          }
        }
        return next
      })

      const converted = relevant.map(toLiveMessage)
      const rate = maxDisplayRateRef.current

      if (!rate || rate <= 0 || resumingRef.current || subjectMode) {
        flush(converted.reverse())
        return
      }

      const queue = [...queueRef.current, ...converted]
      queueRef.current = queue.slice(Math.max(0, queue.length - rate))
      if (!dripTimerRef.current) drip()
    },
    [flush, drip],
  )

  useEffect(() => {
    if (!connectionId || !active) return

    setWsStatus('connecting')
    setIsPaused(false)
    const socket = new LiveStreamClient(connectionId)

    socket.onConnected = () => {
      setWsError(null)
      if (subjectsKeyRef.current) socket.subscribeSubjects(subjectsKeyRef.current.split('\n'), limitsRef.current)
      else if (streamNameRef.current) socket.subscribe(streamNameRef.current)
    }
    socket.onSubscribed = () => {
      setWsStatus('connected')
      setNatsDown(false)
      setWsError(null)
    }
    socket.onBatch = processBatch
    socket.onStats = (payload: WSStatsPayload) => {
      setMsgPerSecond(payload.msg_per_second)
      const tallies = talliesRef.current
      tallies.dropped.session = payload.messages_dropped
      tallies.received.session = payload.messages_received
      setMessagesDropped(tallyTotal(tallies.dropped))
      setMessagesReceived(tallyTotal(tallies.received))
      if (!globalStats) return
      setGlobalStats({
        messagesReceived: payload.messages_received,
        messagesDropped: payload.messages_dropped,
        msgPerSecond: payload.msg_per_second,
        isConnected: true,
      })
    }
    socket.onError = (payload) => {
      const denied = payload.access?.status === 'denied' ? payload.access.subject : null
      if (denied) {
        setDeniedSubjects((prev) => (prev.includes(denied) ? prev : [...prev, denied]))
        if (subjectsKeyRef.current) return
      }
      setWsError(payload.message)
    }
    socket.onDisconnect = () => {
      setWsStatus('disconnected')
      setNatsDown(false)
      setMsgPerSecond(undefined)
      if (globalStats) setGlobalStats(null)
    }
    socket.onReconnecting = () => setWsStatus('reconnecting')
    socket.onBuffered = setPausedCount
    socket.onLink = (connected) => setNatsDown(!connected)

    setWs(socket)
    socket.connect()

    return () => {
      socket.disconnect()
      setWs(null)
      setMsgPerSecond(undefined)
      talliesRef.current = carryTallies(talliesRef.current)
      setPausedCount(0)
      setIsPaused(false)
      stopDrip()
      clearHighlightTimers()
    }

  }, [connectionId, active, processBatch, globalStats, setGlobalStats, clearHighlightTimers, stopDrip])

  // Subscribe / unsubscribe when stream changes on an open connection.
  useEffect(() => {
    if (!ws || wsStatus === 'disconnected') return
    talliesRef.current = carryTallies(talliesRef.current)
    if (subjectsKey) ws.subscribeSubjects(subjectsKey.split('\n'), limitsRef.current)
    else if (streamName) ws.subscribe(streamName)
    else ws.unsubscribe()
  }, [ws, streamName, subjectsKey, limitsKey, wsStatus])

  // Clear on stream change
  useEffect(() => {
    stopDrip()
    setLiveMessages([])
  }, [streamName, stopDrip])

  // Trim on limit change
  useEffect(() => {
    setLiveMessages((prev) => prev.slice(0, liveLimit))
  }, [liveLimit])

  const togglePause = useCallback(() => {
    if (!ws) return
    if (isPaused) {
      resumingRef.current = true
      ws.resume()
      resumingRef.current = false
      setIsPaused(false)
    } else {
      ws.pause()
      flushQueue()
      setIsPaused(true)
    }
  }, [ws, isPaused, flushQueue])

  const clearMessages = useCallback(() => {
    stopDrip()
    wsRef.current?.discardPaused()
    setPausedCount(0)
    setLiveMessages([])
    setSubjectCounts({})
    talliesRef.current = clearTallies(talliesRef.current)
    setMessagesDropped((prev) => (prev === undefined ? prev : 0))
    setMessagesReceived((prev) => (prev === undefined ? prev : 0))
  }, [stopDrip])

  const opening = !!connectionId && active && ws === null && wsStatus === 'disconnected'
  const shownStatus = opening ? 'connecting' : natsDown && wsStatus === 'connected' ? 'reconnecting' : wsStatus

  return useMemo(
    () => ({
      liveMessages,
      liveLimit,
      setLiveLimit,
      wsStatus: shownStatus,
      wsError,
      isPaused,
      togglePause,
      newMessageIds,
      clearMessages,
      subjectCounts,
      deniedSubjects,
      msgPerSecond,
      messagesDropped,
      messagesReceived,
      pausedCount,
    }),
    [
      liveMessages,
      liveLimit,
      shownStatus,
      wsError,
      isPaused,
      togglePause,
      newMessageIds,
      clearMessages,
      subjectCounts,
      deniedSubjects,
      msgPerSecond,
      messagesDropped,
      messagesReceived,
      pausedCount,
    ],
  )
}
