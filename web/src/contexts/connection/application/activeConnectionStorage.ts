import { z } from 'zod'
import type { ConnectionLabel } from '@/api/connections'
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

const storedConnectionInfoSchema = z.object({
  id: z.string(),
  name: z.string(),
  urls: z.array(z.string()),
  readOnly: z.boolean().optional(),
  label: z.object({ text: z.string(), color: z.enum(['gray', 'blue', 'green', 'amber', 'red']) }).nullable().optional(),
})

export type StoredConnectionInfo = z.infer<typeof storedConnectionInfoSchema>

export function getStoredConnectionInfo(): StoredConnectionInfo | null {
  const raw = safeGetItem(ACTIVE_CONNECTION_INFO_KEY)
  if (!raw) return null
  try {
    const parsed = storedConnectionInfoSchema.safeParse(JSON.parse(raw))
    return parsed.success ? parsed.data : null
  } catch {
    return null
  }
}

export function storeActiveConnectionInfo(connection: {
  id: string
  name: string
  urls: string[]
  readOnly?: boolean
  label?: ConnectionLabel | null
}): void {
  safeSetItem(
    ACTIVE_CONNECTION_INFO_KEY,
    JSON.stringify({
      id: connection.id,
      name: connection.name,
      urls: connection.urls,
      readOnly: connection.readOnly ?? false,
      label: connection.label ?? null,
    }),
  )
}
