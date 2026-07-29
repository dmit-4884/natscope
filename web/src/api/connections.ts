import { tsToMillis, durToMillis, millisToDur } from '@/utils/timestamp'
import type { SavedConnection as ProtoSavedConnection } from '../gen/types/nats/nats_connection_pb'
import { AuthMethod as ProtoAuthMethod } from '../gen/types/nats/nats_connection_pb'
import type { AuthConfig, TlsConfig, ConnectionConfig, ReconnectConfig, PingConfig } from '../gen/types/nats/nats_connection_pb'
import type { TestConnectionResponse } from '../gen/services/grpc/nats/v1/connections/nats_connections_service_pb'
import { connectionsClient } from './grpc/clients'

export type { TestConnectionResponse }

/** Domain-level auth method; proto enum stays internal. */
export type AuthMethod = 'none' | 'userpass' | 'token' | 'nkey' | 'credentials'

const PROTO_TO_DOMAIN_AUTH_METHOD: Record<ProtoAuthMethod, AuthMethod> = {
  [ProtoAuthMethod.UNSPECIFIED]: 'none',
  [ProtoAuthMethod.USER_PASSWORD]: 'userpass',
  [ProtoAuthMethod.TOKEN]: 'token',
  [ProtoAuthMethod.NKEY]: 'nkey',
  [ProtoAuthMethod.CREDENTIALS]: 'credentials',
}

/** Most recent server probe (server-populated, never client). */
interface ConnectionMeta {
  lastTestedAt: number
  lastSuccess: boolean
  lastRttMs?: number
  serverVersion?: string
  serverName?: string
  serverId?: string
  clusterName?: string
  maxPayload?: number
  jetstreamEnabled?: boolean
  connectedUrl?: string
  lastError?: string
}

export interface SavedConnection {
  id: string
  name: string
  description?: string
  urls: string[]
  auth?: {
    method: AuthMethod
    username?: string
    // Secret values are input-only and never returned by the server; the has*
    // flags below report whether a secret is stored.
    password?: string
    token?: string
    nkeySeed?: string
    credentials?: string
    jwt?: string
    hasPassword: boolean
    hasToken: boolean
    hasNkeySeed: boolean
    hasCredentials: boolean
  }
  tls?: {
    caCert?: string
    clientCert?: string
    clientKey?: string
    skipVerify: boolean
    tlsFirst: boolean
    hasClientKey: boolean
  }
  connection?: {
    connectTimeout?: number
    connectionName?: string
    inboxPrefix?: string
    noEcho: boolean
    noRandomize: boolean
    ignoreDiscoveredServers: boolean
  }
  reconnect?: {
    maxReconnects?: number
    reconnectWait?: number
    reconnectBufSize?: number
    retryOnFailedConnect: boolean
  }
  ping?: {
    pingInterval?: number
    maxPingsOutstanding?: number
  }
  meta?: ConnectionMeta
  createdAt: number
  updatedAt: number
}

export interface CreateConnectionRequest {
  name: string
  description?: string
  urls: string[]
  auth?: AuthConfig
  tls?: TlsConfig
  connection?: ConnectionConfig
  reconnect?: ReconnectConfig
  ping?: PingConfig
}

export interface UpdateConnectionRequest {
  name?: string
  description?: string
  urls?: string[]
  auth?: AuthConfig
  tls?: TlsConfig
  connection?: ConnectionConfig
  reconnect?: ReconnectConfig
  ping?: PingConfig
}

export interface TestConnectionRequest {
  urls: string[]
  auth?: AuthConfig
  tls?: TlsConfig
  connectTimeout?: number
  /** When set, backend records the probe result into that connection's `meta`. */
  connectionId?: string
}

function toSavedConnection(proto: ProtoSavedConnection): SavedConnection {
  const result: SavedConnection = {
    id: proto.id,
    name: proto.name,
    description: proto.description,
    urls: proto.urls,
    createdAt: tsToMillis(proto.createdAt),
    updatedAt: tsToMillis(proto.updatedAt),
  }

  if (proto.auth) {
    result.auth = {
      method: PROTO_TO_DOMAIN_AUTH_METHOD[proto.auth.method] ?? 'none',
      username: proto.auth.username,
      password: proto.auth.password,
      token: proto.auth.token,
      nkeySeed: proto.auth.nkeySeed,
      credentials: proto.auth.credentials,
      jwt: proto.auth.jwt,
      hasPassword: proto.auth.hasPassword,
      hasToken: proto.auth.hasToken,
      hasNkeySeed: proto.auth.hasNkeySeed,
      hasCredentials: proto.auth.hasCredentials,
    }
  }

  if (proto.tls) {
    result.tls = {
      caCert: proto.tls.caCert,
      clientCert: proto.tls.clientCert,
      clientKey: proto.tls.clientKey,
      skipVerify: proto.tls.skipVerify,
      tlsFirst: proto.tls.tlsFirst,
      hasClientKey: proto.tls.hasClientKey,
    }
  }

  if (proto.connection) {
    result.connection = {
      connectTimeout: durToMillis(proto.connection.connectTimeout),
      connectionName: proto.connection.connectionName,
      inboxPrefix: proto.connection.inboxPrefix,
      noEcho: proto.connection.noEcho,
      noRandomize: proto.connection.noRandomize,
      ignoreDiscoveredServers: proto.connection.ignoreDiscoveredServers,
    }
  }

  if (proto.reconnect) {
    result.reconnect = {
      maxReconnects: proto.reconnect.maxReconnects,
      reconnectWait: durToMillis(proto.reconnect.reconnectWait),
      reconnectBufSize: proto.reconnect.reconnectBufSize,
      retryOnFailedConnect: proto.reconnect.retryOnFailedConnect,
    }
  }

  if (proto.ping) {
    result.ping = {
      pingInterval: durToMillis(proto.ping.pingInterval),
      maxPingsOutstanding: proto.ping.maxPingsOutstanding,
    }
  }

  if (proto.meta) {
    result.meta = {
      lastTestedAt: tsToMillis(proto.meta.lastTestedAt),
      lastSuccess: proto.meta.lastSuccess,
      lastRttMs: proto.meta.lastRttMs !== undefined ? Number(proto.meta.lastRttMs) : undefined,
      serverVersion: proto.meta.serverVersion,
      serverName: proto.meta.serverName,
      serverId: proto.meta.serverId,
      clusterName: proto.meta.clusterName,
      maxPayload: proto.meta.maxPayload !== undefined ? Number(proto.meta.maxPayload) : undefined,
      jetstreamEnabled: proto.meta.jetstreamEnabled,
      connectedUrl: proto.meta.connectedUrl,
      lastError: proto.meta.lastError,
    }
  }

  return result
}

export async function getConnections(): Promise<SavedConnection[]> {
  const response = await connectionsClient.listConnections({ pageSize: 500, pageToken: '' })
  return (response.connections || []).map(toSavedConnection)
}

export async function createConnection(req: CreateConnectionRequest): Promise<SavedConnection> {
  const response = await connectionsClient.createConnection({
    name: req.name,
    description: req.description,
    urls: req.urls,
    auth: req.auth,
    tls: req.tls,
    connection: req.connection,
    reconnect: req.reconnect,
    ping: req.ping,
  })
  return toSavedConnection(response.connection!)
}

export async function updateConnection(id: string, req: UpdateConnectionRequest): Promise<SavedConnection> {
  const response = await connectionsClient.updateConnection({
    id,
    name: req.name,
    description: req.description,
    urls: req.urls ?? [],
    auth: req.auth,
    tls: req.tls,
    connection: req.connection,
    reconnect: req.reconnect,
    ping: req.ping,
  })
  return toSavedConnection(response.connection!)
}

export async function deleteConnection(id: string): Promise<void> {
  await connectionsClient.deleteConnection({
    id,
  })
}

export async function testConnection(req: TestConnectionRequest): Promise<TestConnectionResponse> {
  return await connectionsClient.testConnection({
    urls: req.urls,
    auth: req.auth,
    tls: req.tls,
    connectTimeout: millisToDur(req.connectTimeout),
    connectionId: req.connectionId,
  })
}

export async function duplicateConnection(id: string, name: string): Promise<SavedConnection> {
  const response = await connectionsClient.duplicateConnection({ id, name })
  return toSavedConnection(response.connection!)
}
