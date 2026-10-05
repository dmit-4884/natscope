import { useCallback, useMemo, useState, type ReactNode } from 'react'
import { useLivePolicy } from '@/contexts/settings'
import { useSubscribeDraft, withRecentSubjects } from '@/stores/subscribeDraftStore'
import type { SelectedMessage } from '@/types/messages'
import { useLiveSubscription } from '../messages/unified/useLiveSubscription'
import { SubscribeRunningContext, SubscribeSessionContext, type SubscribeSession } from './subscribeSession'

interface Props {
  connectionId: string
  children: ReactNode
}

export function SubscribeSessionProvider({ connectionId, children }: Props) {
  const [draft, updateDraft] = useSubscribeDraft(connectionId)
  const [running, setRunning] = useState(false)
  const [query, setQuery] = useState('')
  const [subjectFilter, setSubjectFilter] = useState<string | null>(null)
  const [selected, setSelected] = useState<SelectedMessage | null>(null)
  const liveSettings = useLivePolicy()

  const live = useLiveSubscription({
    connectionId,
    streamName: null,
    subjects: running ? draft.subjects : undefined,
    enabled: running,
    maxDisplayRate: liveSettings.maxDisplayRate,
    initialLimit: 100,
    globalStats: false,
  })

  const recentSubjects = draft.recentSubjects
  const start = useCallback(
    (subjects: string[]) => {
      updateDraft({ recentSubjects: withRecentSubjects(recentSubjects, subjects) })
      setRunning(true)
    },
    [recentSubjects, updateDraft],
  )
  const stop = useCallback(() => setRunning(false), [])

  const session = useMemo<SubscribeSession>(
    () => ({ running, start, stop, live, query, setQuery, subjectFilter, setSubjectFilter, selected, setSelected }),
    [running, start, stop, live, query, subjectFilter, selected],
  )

  return (
    <SubscribeRunningContext.Provider value={running}>
      <SubscribeSessionContext.Provider value={session}>{children}</SubscribeSessionContext.Provider>
    </SubscribeRunningContext.Provider>
  )
}
