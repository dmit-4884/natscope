import { createContext, useContext } from 'react'
import type { SelectedMessage } from '@/types/messages'
import type { LiveSubscription } from '../messages/unified/useLiveSubscription'

export interface SubscribeSession {
  running: boolean
  start: (subjects: string[]) => void
  stop: () => void
  live: LiveSubscription
  query: string
  setQuery: (query: string) => void
  subjectFilter: string | null
  setSubjectFilter: (subject: string | null) => void
  selected: SelectedMessage | null
  setSelected: (message: SelectedMessage | null) => void
}

export const SubscribeSessionContext = createContext<SubscribeSession | null>(null)

export const SubscribeRunningContext = createContext(false)

export function useSubscribeSession(): SubscribeSession {
  const session = useContext(SubscribeSessionContext)
  if (!session) throw new Error('useSubscribeSession must be used inside SubscribeSessionProvider')
  return session
}

export function useSubscribeRunning(): boolean {
  return useContext(SubscribeRunningContext)
}
