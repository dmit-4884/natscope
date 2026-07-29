import { LIVE_MESSAGES_STORAGE_PREFIX } from '@/utils/constants'
import { safeRemoveItem } from '@/utils/safeStorage'

/** Storage key for a connection. */
function getStorageKey(connectionId: string | null): string {
  return `${LIVE_MESSAGES_STORAGE_PREFIX}${connectionId || 'default'}`
}

/**
 * Remove all live-message localStorage keys except keepConnectionId's.
 * Call on disconnect/switch to bound localStorage growth.
 */
export function clearLiveMessagesStorage(keepConnectionId?: string | null) {
  try {
    const keysToRemove: string[] = []
    for (let i = 0; i < localStorage.length; i++) {
      const key = localStorage.key(i)
      if (key && key.startsWith(LIVE_MESSAGES_STORAGE_PREFIX)) {
        if (keepConnectionId && key === getStorageKey(keepConnectionId)) continue
        keysToRemove.push(key)
      }
    }
    keysToRemove.forEach((key) => safeRemoveItem(key))
  } catch {
    // Private mode / quota-exceeded — nothing to clean up.
  }
}
