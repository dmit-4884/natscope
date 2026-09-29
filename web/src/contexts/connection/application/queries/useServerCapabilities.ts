import { getServerInfo } from '@/api/stats'
import { useConnectionQuery } from '@/hooks/useConnectionQuery'

export type CapabilityKey =
  | 'consumerPause'
  | 'messageTtl'
  | 'atomicPublish'
  | 'priorityGroups'
  | 'msgCounters'
  | 'msgSchedules'
  | 'priorityPrioritized'
  | 'asyncPersist'
  | 'consumerReset'
  | 'cronSchedules'
  | 'batchPublish'

/**
 * Display-only requirement labels for tooltips. The boolean truth always
 * comes from the backend (internal/pkg/natsclient/natsgo/capabilities.go).
 */
const CAPABILITY_REQUIREMENTS: Record<CapabilityKey, string> = {
  consumerPause: 'NATS 2.11+',
  messageTtl: 'NATS 2.11+',
  atomicPublish: 'NATS 2.12+',
  priorityGroups: 'NATS 2.11+',
  msgCounters: 'NATS 2.12+',
  msgSchedules: 'NATS 2.12+',
  priorityPrioritized: 'NATS 2.12+',
  asyncPersist: 'NATS 2.12+',
  consumerReset: 'NATS 2.14+',
  cronSchedules: 'NATS 2.14+',
  batchPublish: 'NATS 2.14+',
}

type ServerCapabilities = Record<CapabilityKey, boolean> & { apiLevel: number }

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

  const caps = data?.capabilities
  const capabilities: ServerCapabilities | undefined = caps
    ? {
        apiLevel: caps.api_level,
        consumerPause: caps.consumer_pause,
        messageTtl: caps.message_ttl,
        atomicPublish: caps.atomic_publish,
        priorityGroups: caps.priority_groups,
        msgCounters: caps.msg_counters,
        msgSchedules: caps.msg_schedules,
        priorityPrioritized: caps.priority_prioritized,
        asyncPersist: caps.async_persist,
        consumerReset: caps.consumer_reset,
        cronSchedules: caps.cron_schedules,
        batchPublish: caps.batch_publish,
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
