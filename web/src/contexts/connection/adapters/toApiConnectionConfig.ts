import { create } from '@bufbuild/protobuf'
import { ConnectionConfigSchema, type ConnectionConfig as ProtoConnectionConfig } from '@/gen/types/nats/nats_connection_pb'
import type { SavedConnection } from '@/api/connections'
import { millisToDur } from '@/utils/timestamp'

export interface ConnectionConfigEdits {
  inboxPrefix: string
  jetstreamDomain: string
  jetstreamApiPrefix: string
}

export function toApiConnectionConfig(
  saved: SavedConnection['connection'],
  edits: ConnectionConfigEdits,
): ProtoConnectionConfig | undefined {
  const prefix = edits.inboxPrefix.trim()
  const domain = edits.jetstreamDomain.trim()
  const apiPrefix = edits.jetstreamApiPrefix.trim()
  if (!saved && !prefix && !domain && !apiPrefix) return undefined
  return create(ConnectionConfigSchema, {
    connectTimeout: millisToDur(saved?.connectTimeout),
    connectionName: saved?.connectionName,
    inboxPrefix: prefix,
    noEcho: saved?.noEcho ?? false,
    noRandomize: saved?.noRandomize ?? false,
    ignoreDiscoveredServers: saved?.ignoreDiscoveredServers ?? false,
    jetstreamDomain: domain,
    jetstreamApiPrefix: apiPrefix,
  })
}

export function jetStreamTargetErrors(domain: string, apiPrefix: string): { domain?: string; prefix?: string } {
  const d = domain.trim()
  const p = apiPrefix.trim()
  if (d && !/^[A-Za-z0-9_-]+$/.test(d)) return { domain: 'A domain is one word: letters, digits, - and _' }
  if (p && /[\s*>]/.test(p)) return { prefix: 'The API prefix cannot contain spaces or wildcards' }
  if (d && p) return { prefix: 'Set a JetStream domain or an API prefix, not both' }
  return {}
}

export function inboxPrefixError(inboxPrefix: string): string | undefined {
  const prefix = inboxPrefix.trim()
  if (!prefix) return undefined
  if (/\s/.test(prefix)) return 'The inbox prefix cannot contain spaces'
  if (/[*>]/.test(prefix)) return 'The inbox prefix cannot contain wildcards'
  if (prefix.split('.').some((token) => token === '')) return 'The inbox prefix cannot start or end with a dot'
  return undefined
}
