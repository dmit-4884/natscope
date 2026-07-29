import { statsClient } from './grpc/clients'

export interface ConnectionHealth {
  id: string
  url: string
  status: 'connected' | 'reconnecting' | 'disconnected'
  rtt?: string
  error?: string
  is_connected: boolean
  is_reconnecting: boolean
  server_version?: string
}


export interface ServerInfoResponse {
  server_id: string
  server_name: string
  version: string
  host: string
  port: number
  cluster_name: string
  max_payload: number
  connected_url: string
  jetstream: boolean
  auth_required: boolean
  tls_required: boolean
  connect_urls: string[]
  client_stats?: {
    in_msgs: number
    out_msgs: number
    in_bytes: number
    out_bytes: number
    reconnects: number
  }
  js_account?: {
    memory: number
    storage: number
    streams: number
    consumers: number
    memory_limit: number
    storage_limit: number
    stream_limit: number
    consumer_limit: number
    api_total: number
    api_errors: number
    domain: string
  }
  capabilities?: {
    api_level: number
    consumer_pause: boolean
    message_ttl: boolean
    atomic_publish: boolean
  }
}

export async function getServerInfo(
  connectionId: string,
  signal?: AbortSignal,
): Promise<ServerInfoResponse> {
  const response = await statsClient.getServerInfo({ connectionId }, { signal })
  const info = response.serverInfo!
  return {
    server_id: info.serverId,
    server_name: info.serverName,
    version: info.version,
    host: info.host,
    port: info.port,
    cluster_name: info.clusterName,
    max_payload: Number(info.maxPayload),
    connected_url: info.connectedUrl,
    jetstream: info.jetstream,
    auth_required: info.authRequired,
    tls_required: info.tlsRequired,
    connect_urls: info.connectUrls,
    client_stats: info.clientStats
      ? {
          in_msgs: Number(info.clientStats.inMsgs),
          out_msgs: Number(info.clientStats.outMsgs),
          in_bytes: Number(info.clientStats.inBytes),
          out_bytes: Number(info.clientStats.outBytes),
          reconnects: Number(info.clientStats.reconnects),
        }
      : undefined,
    js_account: info.jsAccount
      ? {
          memory: Number(info.jsAccount.memory),
          storage: Number(info.jsAccount.storage),
          streams: Number(info.jsAccount.streams),
          consumers: Number(info.jsAccount.consumers),
          memory_limit: Number(info.jsAccount.memoryLimit),
          storage_limit: Number(info.jsAccount.storageLimit),
          stream_limit: Number(info.jsAccount.streamLimit),
          consumer_limit: Number(info.jsAccount.consumerLimit),
          api_total: Number(info.jsAccount.apiTotal),
          api_errors: Number(info.jsAccount.apiErrors),
          domain: info.jsAccount.domain,
        }
      : undefined,
    capabilities: info.capabilities
      ? {
          api_level: info.capabilities.apiLevel,
          consumer_pause: info.capabilities.consumerPause,
          message_ttl: info.capabilities.messageTtl,
          atomic_publish: info.capabilities.atomicPublish,
        }
      : undefined,
  }
}

export async function getHealth(
  connectionId: string,
  signal?: AbortSignal,
): Promise<ConnectionHealth> {
  const response = await statsClient.getHealth({ connectionId }, { signal })
  const health = response.health
  if (!health) {
    return { id: connectionId, url: '', status: 'disconnected', is_connected: false, is_reconnecting: false }
  }
  return {
    id: health.id,
    url: health.url,
    status: health.isConnected ? 'connected' : health.isReconnecting ? 'reconnecting' : 'disconnected',
    rtt: health.rtt || undefined,
    error: health.error || undefined,
    is_connected: health.isConnected,
    is_reconnecting: health.isReconnecting,
    server_version: health.serverVersion || undefined,
  }
}
