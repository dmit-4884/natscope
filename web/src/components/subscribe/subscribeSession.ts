import { createContext, useContext } from 'react'
import type { SelectedMessage } from '@/types/messages'
import type { LiveSubscription } from '../messages/unified/useLiveSubscription'

export type SubscribeStatus = 'idle' | 'live' | 'connecting' | 'denied' | 'failed'

export interface SubscribeSession {
  running: boolean
  allDenied: boolean
  start: (subjects: string[]) => void
  stop: () => void
  live: LiveSubscription
  displayRate: number
  setDisplayRate: (rate: number) => void
  muted: string[]
  mute: (subject: string) => void
  unmute: (subject: string) => void
  query: string
  setQuery: (query: string) => void
  subjectFilter: string | null
  setSubjectFilter: (subject: string | null) => void
  selected: SelectedMessage | null
  setSelected: (message: SelectedMessage | null) => void
}

export const SubscribeSessionContext = createContext<SubscribeSession | null>(null)

export const SubscribeStatusContext = createContext<SubscribeStatus>('idle')

export function useSubscribeSession(): SubscribeSession {
  const session = useContext(SubscribeSessionContext)
  if (!session) throw new Error('useSubscribeSession must be used inside SubscribeSessionProvider')
  return session
}

export function useSubscribeStatus(): SubscribeStatus {
  return useContext(SubscribeStatusContext)
}

export const SUBSCRIBE_PAYLOAD_CAP = 64 * 1024
