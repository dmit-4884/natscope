import { safeGetItem, safeSetItem, safeRemoveItem } from '@/utils/safeStorage'

const ACTIVE_CONNECTION_KEY = 'nats_active_connection_id'
export const ACTIVE_CONNECTION_INFO_KEY = 'nats_active_connection_info'

const CHANGE_EVENT = 'natscope:active-connection-changed'

export function getActiveConnectionId(): string | null {
  return safeGetItem(ACTIVE_CONNECTION_KEY)
}

export function setActiveConnectionId(id: string): void {
  safeSetItem(ACTIVE_CONNECTION_KEY, id)
  window.dispatchEvent(new Event(CHANGE_EVENT))
}

export function clearActiveConnection(): void {
  safeRemoveItem(ACTIVE_CONNECTION_KEY)
  safeRemoveItem(ACTIVE_CONNECTION_INFO_KEY)
  window.dispatchEvent(new Event(CHANGE_EVENT))
}

export function subscribeActiveConnection(callback: () => void): () => void {
  window.addEventListener(CHANGE_EVENT, callback)
  window.addEventListener('storage', callback)
  return () => {
    window.removeEventListener(CHANGE_EVENT, callback)
    window.removeEventListener('storage', callback)
  }
}
