import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react'
import { useLivePolicy } from '@/contexts/settings'
import { useSubscribeDraft, withRecentSubjects } from '@/stores/subscribeDraftStore'
import type { SelectedMessage } from '@/types/messages'
import { useLiveSubscription } from '../messages/unified/useLiveSubscription'
import {
  SUBSCRIBE_PAYLOAD_CAP,
  SubscribeSessionContext,
  SubscribeStatusContext,
  type SubscribeSession,
  type SubscribeStatus,
} from './subscribeSession'

interface Props {
  connectionId: string
  children: ReactNode
}

function sessionStatus(running: boolean, allDenied: boolean, connected: boolean): SubscribeStatus {
  if (!running) return 'idle'
  if (allDenied) return 'denied'
  return connected ? 'live' : 'connecting'
}

export function SubscribeSessionProvider({ connectionId, children }: Props) {
  const [draft, updateDraft] = useSubscribeDraft(connectionId)
  const [running, setRunning] = useState(false)
  const [query, setQuery] = useState('')
  const [subjectFilter, setSubjectFilter] = useState<string | null>(null)
  const [selected, setSelected] = useState<SelectedMessage | null>(null)
  const [muted, setMuted] = useState<string[]>([])
  const [rateOverride, setDisplayRate] = useState<number | null>(null)
  const liveSettings = useLivePolicy()
  const displayRate = rateOverride ?? liveSettings.maxDisplayRate ?? 0

  const subjectLimits = useMemo(
    () => ({ maxPayloadBytes: SUBSCRIBE_PAYLOAD_CAP, maxDisplayRate: displayRate, exclude: muted }),
    [displayRate, muted],
  )

  const live = useLiveSubscription({
    connectionId,
    streamName: null,
    subjects: running ? draft.subjects : undefined,
    enabled: running,
    initialLimit: 100,
    globalStats: false,
    subjectLimits,
    exclude: muted,
  })

  const { clearMessages } = live
  useEffect(() => {
    setRunning(false)
    setQuery('')
    setSubjectFilter(null)
    setSelected(null)
    setMuted([])
    setDisplayRate(null)
    clearMessages()
  }, [connectionId, clearMessages])

  const recentSubjects = draft.recentSubjects
  const start = useCallback(
    (subjects: string[]) => {
      updateDraft({ recentSubjects: withRecentSubjects(recentSubjects, subjects) })
      setRunning(true)
    },
    [recentSubjects, updateDraft],
  )
  const stop = useCallback(() => setRunning(false), [])
  const mute = useCallback((subject: string) => {
    setMuted((prev) => (prev.includes(subject) ? prev : [...prev, subject]))
    setSubjectFilter((current) => (current === subject ? null : current))
  }, [])
  const unmute = useCallback((subject: string) => setMuted((prev) => prev.filter((s) => s !== subject)), [])

  const allDenied =
    running && draft.subjects.length > 0 && draft.subjects.every((s) => live.deniedSubjects.includes(s))

  const status = sessionStatus(running, allDenied, live.wsStatus === 'connected')

  const session = useMemo<SubscribeSession>(
    () => ({
      running,
      allDenied,
      start,
      stop,
      live,
      displayRate,
      setDisplayRate,
      muted,
      mute,
      unmute,
      query,
      setQuery,
      subjectFilter,
      setSubjectFilter,
      selected,
      setSelected,
    }),
    [running, allDenied, start, stop, live, displayRate, muted, mute, unmute, query, subjectFilter, selected],
  )

  return (
    <SubscribeStatusContext.Provider value={status}>
      <SubscribeSessionContext.Provider value={session}>{children}</SubscribeSessionContext.Provider>
    </SubscribeStatusContext.Provider>
  )
}
