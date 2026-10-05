import { create } from '@bufbuild/protobuf'
import { ConnectionConfigSchema, type ConnectionConfig as ProtoConnectionConfig } from '@/gen/types/nats/nats_connection_pb'
import type { SavedConnection } from '@/api/connections'
import { millisToDur } from '@/utils/timestamp'

export function toApiConnectionConfig(
  saved: SavedConnection['connection'],
  inboxPrefix: string,
): ProtoConnectionConfig | undefined {
  const prefix = inboxPrefix.trim()
  if (!saved && !prefix) return undefined
  return create(ConnectionConfigSchema, {
    connectTimeout: millisToDur(saved?.connectTimeout),
    connectionName: saved?.connectionName,
    inboxPrefix: prefix,
    noEcho: saved?.noEcho ?? false,
    noRandomize: saved?.noRandomize ?? false,
    ignoreDiscoveredServers: saved?.ignoreDiscoveredServers ?? false,
  })
}

export function inboxPrefixError(inboxPrefix: string): string | undefined {
  const prefix = inboxPrefix.trim()
  if (!prefix) return undefined
  if (/\s/.test(prefix)) return 'The inbox prefix cannot contain spaces'
  if (/[*>]/.test(prefix)) return 'The inbox prefix cannot contain wildcards'
  if (prefix.split('.').some((token) => token === '')) return 'The inbox prefix cannot start or end with a dot'
  return undefined
}
