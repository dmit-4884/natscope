import type { SelectedMessage } from '@/types/messages'
import { createStreamScopedStore, type StreamScope } from './index'

/** Query context the message list is currently showing — published by
 *  UnifiedMessageList so arrow navigation fetches with identical filters. */
export interface NavQuery {
  subjectFilter?: string
  contentFilter?: string
  direction: 'backward' | 'forward'
}

/**
 * Messages-tab state, persisted to sessionStorage (not local) — selection is
 * per-tab; surviving restarts would accumulate stale base64 payloads.
 */
interface MessagesViewEntry {
  selectedMessage: SelectedMessage | null
  /** Optional: entries persisted before this field existed return undefined. */
  navQuery?: NavQuery
}

const DEFAULTS: MessagesViewEntry = {
  selectedMessage: null,
}

const messagesViewStore = createStreamScopedStore<MessagesViewEntry>({
  name: 'messages-view',
  defaults: DEFAULTS,
  version: 1,
  storage: typeof window !== 'undefined' ? window.sessionStorage : undefined,
  // Selections can be large (base64); tighter TTL prunes stale entries.
  ttlMs: 6 * 60 * 60 * 1000, // 6h
})

export function useMessagesViewEntry(scope: StreamScope) {
  return messagesViewStore.useEntry(scope)
}

export function setNavQuery(scope: StreamScope, navQuery: NavQuery): void {
  messagesViewStore.set(scope, { navQuery })
}

/** Drop every entry (e.g. on logout). */
export function clearAllMessagesView(): void {
  messagesViewStore.clearAll()
}
