import { useCallback, useEffect, useRef, useState } from 'react'
import {
  LiveStreamClient,
  useLiveStatsStore,
  useProtoReloadInvalidation,
  type WSBatchPayload,
  type WSMessagePayload,
  type WSStatsPayload,
} from '@/contexts/live'
import { matchesPattern } from '@/contexts/messages'
import type { LiveMessage, LiveMessageLimit, WsStatus } from './messageListUtils'

interface Options {
  connectionId: string | null
  streamName: string | null
  enabled: boolean
  maxDisplayRate?: number
  initialLimit: LiveMessageLimit
  subjectFilter?: string
}

interface Result {
  liveMessages: LiveMessage[]
  liveLimit: LiveMessageLimit
  setLiveLimit: (limit: LiveMessageLimit) => void
  wsStatus: WsStatus
  wsError: string | null
  isPaused: boolean
  togglePause: () => void
  newMessageIds: Set<string>
  clearMessages: () => void
}

function toLiveMessage(msg: WSMessagePayload): LiveMessage {
  const id = `${Date.now()}-${Math.random().toString(36).slice(2, 11)}`
  return {
    ...msg,
    id,
    timestamp: msg.timestamp,
    decoded: msg.decoded ?? undefined,
    decodedType: msg.decoded_type ?? undefined,
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
  enabled,
  maxDisplayRate,
  initialLimit,
  subjectFilter,
}: Options): Result {
  const [liveMessages, setLiveMessages] = useState<LiveMessage[]>([])
  const [liveLimit, setLiveLimit] = useState<LiveMessageLimit>(initialLimit)
  const [wsStatus, setWsStatus] = useState<WsStatus>('disconnected')
  const [wsError, setWsError] = useState<string | null>(null)
  const [isPaused, setIsPaused] = useState(false)
  const [newMessageIds, setNewMessageIds] = useState<Set<string>>(new Set())

  const [ws, setWs] = useState<LiveStreamClient | null>(null)
  const liveLimitRef = useRef(liveLimit)
  const streamNameRef = useRef(streamName)
  const subjectFilterRef = useRef(subjectFilter)
  const setGlobalStats = useLiveStatsStore((s) => s.setStats)

  // Optional display-rate throttle: coalesce incoming messages and flush them
  // to state at most `maxDisplayRate` times per second so a high-throughput
  // stream doesn't thrash React. 0/undefined means flush every batch.
  const maxDisplayRateRef = useRef(maxDisplayRate)
  const pendingRef = useRef<LiveMessage[]>([])
  const flushTimerRef = useRef<ReturnType<typeof setTimeout>>()
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

  const processBatch = useCallback(
    (batch: WSBatchPayload) => {
      const pattern = subjectFilterRef.current
      const relevant = batch.messages.filter((msg) => {
        if (msg.stream_name !== streamNameRef.current) return false
        if (!pattern) return true
        if (pattern.includes('*') || pattern.includes('>')) return matchesPattern(msg.subject, pattern)
        return msg.subject.toLowerCase().includes(pattern.toLowerCase())
      })
      if (relevant.length === 0) return

      const converted = relevant.map(toLiveMessage)
      const rate = maxDisplayRateRef.current

      if (!rate || rate <= 0) {
        flush(converted)
        return
      }

      // Buffer and flush at most `rate` times per second (newest-first).
      pendingRef.current = [...converted, ...pendingRef.current].slice(0, liveLimitRef.current)
      if (flushTimerRef.current) return
      flushTimerRef.current = setTimeout(() => {
        flushTimerRef.current = undefined
        const buffered = pendingRef.current
        pendingRef.current = []
        flush(buffered)
      }, 1000 / rate)
    },
    [flush],
  )

  useEffect(() => {
    if (!connectionId || !enabled) return

    setWsStatus('connecting')
    setIsPaused(false)
    const socket = new LiveStreamClient(connectionId)

    socket.onConnected = () => {
      setWsStatus('connected')
      setWsError(null)
      if (streamNameRef.current) socket.subscribe(streamNameRef.current)
    }
    socket.onSubscribed = () => {
      setWsStatus('connected')
      setWsError(null)
    }
    socket.onBatch = processBatch
    socket.onStats = (payload: WSStatsPayload) => {
      setGlobalStats({
        messagesReceived: payload.messages_received,
        messagesDropped: payload.messages_dropped,
        msgPerSecond: payload.msg_per_second,
        isConnected: true,
      })
    }
    socket.onError = (payload) => setWsError(payload.message)
    socket.onDisconnect = () => {
      setWsStatus('disconnected')
      setGlobalStats(null)
    }
    socket.onReconnecting = () => setWsStatus('reconnecting')

    setWs(socket)
    socket.connect()

    return () => {
      socket.disconnect()
      setWs(null)
      setIsPaused(false)
      clearTimeout(flushTimerRef.current)
      flushTimerRef.current = undefined
      pendingRef.current = []
      clearHighlightTimers()
    }

  }, [connectionId, enabled, processBatch, setGlobalStats, clearHighlightTimers])

  // Subscribe / unsubscribe when stream changes on an open connection.
  useEffect(() => {
    if (!ws || wsStatus !== 'connected') return
    if (streamName) ws.subscribe(streamName)
    else ws.unsubscribe()
  }, [ws, streamName, wsStatus])

  // Clear on stream change
  useEffect(() => {
    setLiveMessages([])
  }, [streamName])

  // Trim on limit change
  useEffect(() => {
    setLiveMessages((prev) => prev.slice(0, liveLimit))
  }, [liveLimit])

  const togglePause = useCallback(() => {
    if (!ws) return
    if (isPaused) {
      ws.resume()
      setIsPaused(false)
    } else {
      ws.pause()
      setIsPaused(true)
    }
  }, [ws, isPaused])

  const clearMessages = useCallback(() => setLiveMessages([]), [])

  return {
    liveMessages,
    liveLimit,
    setLiveLimit,
    wsStatus,
    wsError,
    isPaused,
    togglePause,
    newMessageIds,
    clearMessages,
  }
}
