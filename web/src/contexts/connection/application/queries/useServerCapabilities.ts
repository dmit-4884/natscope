import { getServerInfo } from '@/api/stats'
import { useConnectionQuery } from '@/hooks/useConnectionQuery'

export type CapabilityKey = 'consumerPause' | 'messageTtl' | 'atomicPublish'

/**
 * Display-only requirement labels for tooltips. The boolean truth always
 * comes from the backend (internal/services/nats/nats/capabilities.go).
 */
const CAPABILITY_REQUIREMENTS: Record<CapabilityKey, string> = {
  consumerPause: 'NATS 2.11+',
  messageTtl: 'NATS 2.11+',
  atomicPublish: 'NATS 2.12+',
}

interface ServerCapabilities {
  apiLevel: number
  consumerPause: boolean
  messageTtl: boolean
  atomicPublish: boolean
}

export interface UseServerCapabilitiesResult {
  /** undefined until loaded or when the backend predates capabilities. */
  capabilities: ServerCapabilities | undefined
  serverVersion: string | undefined
  /** Fail open: true when capabilities are unknown; only explicit false gates. */
  isSupported: (feature: CapabilityKey) => boolean
  /** Tooltip text when a feature is unsupported, undefined otherwise. */
  unsupportedReason: (feature: CapabilityKey) => string | undefined
}

/** Capabilities change only on server upgrade — cache aggressively. */
const CAPABILITIES_STALE_TIME_MS = 5 * 60_000

/**
 * Version-gated feature support of the active connection's server.
 * Shares the ['serverInfo'] connection-scoped cache entry with the
 * ServerInfo modal, so at most one RPC is in flight.
 */
export function useServerCapabilities(
  connectionId: string | null | undefined,
): UseServerCapabilitiesResult {
  const { data } = useConnectionQuery({
    key: ['serverInfo'],
    connectionId,
    fetcher: (signal) => getServerInfo(connectionId!, signal),
    staleTime: CAPABILITIES_STALE_TIME_MS,
  })

  const capabilities: ServerCapabilities | undefined = data?.capabilities
    ? {
        apiLevel: data.capabilities.api_level,
        consumerPause: data.capabilities.consumer_pause,
        messageTtl: data.capabilities.message_ttl,
        atomicPublish: data.capabilities.atomic_publish,
      }
    : undefined

  const isSupported = (feature: CapabilityKey) => capabilities?.[feature] !== false

  const unsupportedReason = (feature: CapabilityKey) => {
    if (isSupported(feature)) return undefined
    const version = data?.version ? ` (connected server is v${data.version})` : ''
    return `Requires ${CAPABILITY_REQUIREMENTS[feature]}${version}`
  }

  return { capabilities, serverVersion: data?.version, isSupported, unsupportedReason }
}
