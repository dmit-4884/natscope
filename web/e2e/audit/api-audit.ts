/**
 * Comprehensive Connect-JSON helpers for the audit e2e suite.
 *
 * Wraps the thin `call()` from ../api for every backend service so audit
 * specs can seed/verify state without hand-writing service paths. Runs in the
 * Playwright Node process, so Node APIs (Buffer) are available here.
 *
 * Conventions:
 *   - int64/uint64 proto fields are JSON strings ("10000"), not numbers.
 *   - Duration fields are strings like "3600s" / "0.5s".
 *   - bytes fields are base64 strings.
 *   - enums may be the enum name string or the int; int32 "enum-like" fields
 *     (retention/storage/discard/deliverPolicy/ackPolicy) are plain numbers.
 */
import { call, ConnectError } from '../api'

export { call, ConnectError }

// ---- service type names -----------------------------------------------------
export const SVC = {
  connections: 'natscope.nats.connections.v1.ConnectionsService',
  streams: 'natscope.nats.streams.v1.StreamsService',
  management: 'natscope.nats.management.v1.ManagementService',
  messages: 'natscope.nats.messages.v1.MessagesService',
  publish: 'natscope.nats.publish.v1.PublishService',
  live: 'natscope.nats.live.v1.LiveService',
  stats: 'natscope.nats.stats.v1.StatsService',
  history: 'natscope.history.v1.HistoryService',
  templates: 'natscope.templates.v1.TemplatesService',
  mappings: 'natscope.mappings.v1.MappingsService',
  settings: 'natscope.settings.settings.v1.SettingsService',
  sources: 'natscope.proto.sources.v1.SourcesService',
  registry: 'natscope.proto.registry.v1.RegistryService',
  selections: 'natscope.proto.selections.v1.SelectionsService',
  codec: 'natscope.proto.codec.v1.CodecService',
  workspace: 'natscope.workspace.v1.WorkspaceService',
} as const

// ---- generic helpers --------------------------------------------------------
export const b64 = (s: string | Uint8Array): string =>
  Buffer.from(typeof s === 'string' ? Buffer.from(s, 'utf8') : s).toString('base64')
export const unb64 = (s: string): Buffer => Buffer.from(s, 'base64')

// ============================================================================
// Connections
// ============================================================================
export interface SavedConnection {
  id: string
  name: string
  urls: string[]
  auth?: Record<string, unknown>
  meta?: Record<string, unknown>
}
export async function listConnections(pageSize = 500): Promise<SavedConnection[]> {
  const r = await call<{ connections?: SavedConnection[] }>(SVC.connections, 'ListConnections', { pageSize })
  return r.connections ?? []
}
export async function getConnectionId(name = 'local'): Promise<string> {
  const c = (await listConnections()).find((x) => x.name === name)
  if (!c) throw new Error(`connection "${name}" not found`)
  return c.id
}
export const createConnection = (body: Record<string, unknown>) =>
  call<{ connection?: SavedConnection }>(SVC.connections, 'CreateConnection', body)
export const updateConnection = (body: Record<string, unknown>) =>
  call<{ connection?: SavedConnection }>(SVC.connections, 'UpdateConnection', body)
export const deleteConnection = (id: string) => call(SVC.connections, 'DeleteConnection', { id })
export const duplicateConnection = (id: string, name: string) =>
  call<{ connection?: SavedConnection }>(SVC.connections, 'DuplicateConnection', { id, name })
export const testConnection = (body: Record<string, unknown>) =>
  call<Record<string, unknown>>(SVC.connections, 'TestConnection', body)

// ============================================================================
// Streams / Management
// ============================================================================
export interface StreamCreateOpts {
  description?: string
  subjects?: string[]
  retention?: number // 0=limits 1=interest 2=workqueue
  storage?: number // 0=file 1=memory
  discard?: number // 0=old 1=new
  maxMsgs?: string
  maxBytes?: string
  maxAge?: string // duration e.g. "3600s"
  maxMsgsPerSubject?: string
  maxMsgSize?: number
  maxConsumers?: number
  duplicates?: string
  denyDelete?: boolean
  denyPurge?: boolean
  allowDirect?: boolean
  discardNewPerSubject?: boolean
  allowMsgTtl?: boolean
  metadata?: Record<string, string>
  [k: string]: unknown
}
export const createStream = (connectionId: string, name: string, opts: StreamCreateOpts = {}) =>
  call<{ stream?: unknown }>(SVC.management, 'CreateStream', { connectionId, name, ...opts })
export const updateStream = (connectionId: string, streamName: string, opts: Record<string, unknown>) =>
  call(SVC.management, 'UpdateStream', { connectionId, streamName, ...opts })
export const deleteStream = (connectionId: string, streamName: string) =>
  call(SVC.management, 'DeleteStream', { connectionId, streamName })
export const purgeStream = (connectionId: string, streamName: string, opts: { filter?: string; sequence?: string; keep?: string } = {}) =>
  call<{ purged?: string }>(SVC.management, 'PurgeStream', { connectionId, streamName, ...opts })
export const sealStream = (connectionId: string, streamName: string) =>
  call(SVC.management, 'SealStream', { connectionId, streamName })
export const deleteMessage = (connectionId: string, streamName: string, sequence: number, secure = false) =>
  call(SVC.management, 'DeleteMessage', { connectionId, streamName, sequence: String(sequence), secure })

export const listStreams = (connectionId: string) =>
  call<{ streams?: Array<{ config?: { name?: string } }> }>(SVC.streams, 'ListStreams', { connectionId })
export const getStream = (connectionId: string, streamName: string) =>
  call<{ stream?: { config?: Record<string, unknown>; state?: Record<string, unknown> } }>(SVC.streams, 'GetStream', { connectionId, streamName })
export async function streamExists(connectionId: string, streamName: string): Promise<boolean> {
  try {
    await getStream(connectionId, streamName)
    return true
  } catch (e) {
    if (e instanceof ConnectError && /not.?found/i.test(e.message)) return false
    throw e
  }
}
export async function deleteStreamIfExists(connectionId: string, streamName: string): Promise<void> {
  try {
    await deleteStream(connectionId, streamName)
  } catch (e) {
    if (!(e instanceof ConnectError && /not.?found|does not exist/i.test(e.message))) throw e
  }
}

// ============================================================================
// Consumers
// ============================================================================
export interface ConsumerOpts {
  description?: string
  deliverPolicy?: number // 0=all 1=last 2=new 3=byStartSeq 4=byStartTime 5=lastPerSubject
  optStartSeq?: string
  optStartTime?: string
  ackPolicy?: number // 0=explicit 1=none 2=all
  ackWait?: string
  maxDeliver?: number
  filterSubject?: string
  filterSubjects?: string[]
  replayPolicy?: number
  maxAckPending?: number
  deliverSubject?: string // push consumer when set
  deliverGroup?: string
  headersOnly?: boolean
  [k: string]: unknown
}
export const createConsumer = (connectionId: string, streamName: string, name: string, opts: ConsumerOpts = {}) =>
  call<{ consumer?: unknown }>(SVC.management, 'CreateConsumer', { connectionId, streamName, name, ...opts })
export const listConsumers = (connectionId: string, streamName: string) =>
  call<{ consumers?: Array<{ name?: string; config?: Record<string, unknown> }> }>(SVC.management, 'ListConsumers', { connectionId, streamName })
export const deleteConsumer = (connectionId: string, streamName: string, consumerName: string) =>
  call(SVC.management, 'DeleteConsumer', { connectionId, streamName, consumerName })
export const pauseConsumer = (connectionId: string, streamName: string, consumerName: string, pauseUntil: string) =>
  call<{ paused?: boolean; pauseUntil?: string }>(SVC.management, 'PauseConsumer', { connectionId, streamName, consumerName, pauseUntil })
export const resumeConsumer = (connectionId: string, streamName: string, consumerName: string) =>
  call<{ paused?: boolean }>(SVC.management, 'ResumeConsumer', { connectionId, streamName, consumerName })
export const updateConsumer = (connectionId: string, streamName: string, consumerName: string, opts: Record<string, unknown>) =>
  call(SVC.management, 'UpdateConsumer', { connectionId, streamName, consumerName, ...opts })

// ============================================================================
// Messages
// ============================================================================
export interface ListMessagesOpts {
  subjectFilter?: string
  direction?: 'DIRECTION_FORWARD' | 'DIRECTION_BACKWARD' | 'DIRECTION_UNSPECIFIED'
  startSeq?: string
  limit?: string
  contentFilter?: string
  maxPayloadBytes?: number
  startTime?: string // RFC3339
}
export interface NatsMessage {
  sequence?: string
  subject?: string
  // ListMessages/GetMessage return the payload as base64 under `dataBase64`.
  dataBase64?: string
  dataSize?: number
  truncated?: boolean
  timestamp?: string
  // headers come back as a flat string map in the messages responses.
  headers?: Record<string, string>
  contentType?: string
  decodedType?: string
  decodeError?: string
  [k: string]: unknown
}
/** Decode a NatsMessage payload (base64) to a UTF-8 string. */
export const msgText = (m: NatsMessage): string => (m.dataBase64 ? unb64(m.dataBase64).toString('utf8') : '')
export async function listMessages(connectionId: string, streamName: string, opts: ListMessagesOpts = {}): Promise<{ messages: NatsMessage[]; hasMore: boolean; nextSeq: string }> {
  const r = await call<{ messages?: NatsMessage[]; hasMore?: boolean; nextSeq?: string }>(SVC.messages, 'ListMessages', { connectionId, streamName, ...opts })
  return { messages: r.messages ?? [], hasMore: !!r.hasMore, nextSeq: r.nextSeq ?? '0' }
}
export const getMessage = (connectionId: string, streamName: string, sequence: number) =>
  call<{ message?: NatsMessage }>(SVC.messages, 'GetMessage', { connectionId, streamName, sequence: String(sequence) })

// ============================================================================
// Publish
// ============================================================================
export async function publish(connectionId: string, subject: string, subjectPattern: string, data: string, headers?: Record<string, string>): Promise<{ error?: string; sequence?: string }> {
  const r = await call<{ error?: string; sequence?: string }>(SVC.publish, 'PublishMessage', { connectionId, subject, subjectPattern, data, ...(headers ? { headers } : {}) })
  return r
}

// ============================================================================
// KV
// ============================================================================
export interface KVBucketConfig {
  bucket: string
  description?: string
  maxValueSize?: number
  maxBytes?: string
  history?: number
  ttl?: string
  storage?: number
  [k: string]: unknown
}
export const createKVBucket = (connectionId: string, config: KVBucketConfig) =>
  call<{ bucket?: unknown }>(SVC.management, 'CreateKVBucket', { connectionId, config })
export const listKVBuckets = (connectionId: string) =>
  call<{ buckets?: Array<{ bucket?: string; history?: number; values?: string }> }>(SVC.management, 'ListKVBuckets', { connectionId })
export const getKVBucket = (connectionId: string, bucket: string) =>
  call<{ bucket?: Record<string, unknown> }>(SVC.management, 'GetKVBucket', { connectionId, bucket })
export const deleteKVBucket = (connectionId: string, bucket: string) =>
  call(SVC.management, 'DeleteKVBucket', { connectionId, bucket })
export const putKVKey = (connectionId: string, bucket: string, key: string, value: string, revision = 0) =>
  call<{ revision?: string }>(SVC.management, 'PutKVKey', { connectionId, bucket, key, value, ...(revision ? { revision: String(revision) } : {}) })
// NOTE: entry.value is base64-encoded in the JSON response; use kvText() to decode.
export const getKVKey = (connectionId: string, bucket: string, key: string) =>
  call<{ entry?: { value?: string; revision?: string; operation?: string } }>(SVC.management, 'GetKVKey', { connectionId, bucket, key })
/** Decode a base64 KV entry value to a UTF-8 string. */
export const kvText = (entry?: { value?: string }): string => (entry?.value ? unb64(entry.value).toString('utf8') : '')
export const listKVKeys = (connectionId: string, bucket: string) =>
  call<{ keys?: string[] }>(SVC.management, 'ListKVKeys', { connectionId, bucket })
export const getKVKeyHistory = (connectionId: string, bucket: string, key: string) =>
  call<{ entries?: Array<{ value?: string; revision?: string; operation?: string }> }>(SVC.management, 'GetKVKeyHistory', { connectionId, bucket, key })
export const deleteKVKey = (connectionId: string, bucket: string, key: string) =>
  call(SVC.management, 'DeleteKVKey', { connectionId, bucket, key })
export const purgeKVKey = (connectionId: string, bucket: string, key: string) =>
  call(SVC.management, 'PurgeKVKey', { connectionId, bucket, key })
export async function deleteKVBucketIfExists(connectionId: string, bucket: string): Promise<void> {
  try {
    await deleteKVBucket(connectionId, bucket)
  } catch (e) {
    if (!(e instanceof ConnectError && /not.?found|does not exist|no.*bucket/i.test(e.message))) throw e
  }
}

// ============================================================================
// Object store
// ============================================================================
export interface ObjectBucketConfig {
  bucket: string
  description?: string
  maxBytes?: string
  ttl?: string
  storage?: number
  [k: string]: unknown
}
export const createObjectBucket = (connectionId: string, config: ObjectBucketConfig) =>
  call<{ bucket?: unknown }>(SVC.management, 'CreateObjectBucket', { connectionId, config })
export const listObjectBuckets = (connectionId: string) =>
  call<{ buckets?: Array<{ bucket?: string; objects?: string; sealed?: boolean }> }>(SVC.management, 'ListObjectBuckets', { connectionId })
export const getObjectBucket = (connectionId: string, bucket: string) =>
  call<{ bucket?: Record<string, unknown> }>(SVC.management, 'GetObjectBucket', { connectionId, bucket })
export const deleteObjectBucket = (connectionId: string, bucket: string) =>
  call(SVC.management, 'DeleteObjectBucket', { connectionId, bucket })
export const sealObjectBucket = (connectionId: string, bucket: string) =>
  call(SVC.management, 'SealObjectBucket', { connectionId, bucket })
export const putObject = (connectionId: string, bucket: string, name: string, data: string | Uint8Array, extra: { description?: string; metadata?: Record<string, string> } = {}) =>
  call<{ info?: { size?: string; chunks?: number; digest?: string } }>(SVC.management, 'PutObject', { connectionId, bucket, name, data: b64(data), ...extra })
export const getObject = (connectionId: string, bucket: string, name: string) =>
  call<{ info?: { size?: string; digest?: string }; data?: string }>(SVC.management, 'GetObject', { connectionId, bucket, name })
export const listObjects = (connectionId: string, bucket: string) =>
  call<{ objects?: Array<{ name?: string; size?: string; chunks?: number; deleted?: boolean }> }>(SVC.management, 'ListObjects', { connectionId, bucket })
export const deleteObject = (connectionId: string, bucket: string, name: string) =>
  call(SVC.management, 'DeleteObject', { connectionId, bucket, name })
export async function deleteObjectBucketIfExists(connectionId: string, bucket: string): Promise<void> {
  try {
    await deleteObjectBucket(connectionId, bucket)
  } catch (e) {
    if (!(e instanceof ConnectError && /not.?found|does not exist|no.*bucket/i.test(e.message))) throw e
  }
}

// ============================================================================
// Mappings
// ============================================================================
export const listMappings = (pageSize = 1000) =>
  call<{ mappings?: Array<{ id: string; pattern: string; messageType?: string; sourceId?: string }> }>(SVC.mappings, 'ListMappings', { pageSize })
export const createMapping = (body: { pattern: string; messageType: string; sourceId?: string }) =>
  call<{ mapping?: { id: string } }>(SVC.mappings, 'CreateMapping', body)
export const updateMapping = (body: Record<string, unknown>) => call(SVC.mappings, 'UpdateMapping', body)
export const deleteMapping = (id: string) => call(SVC.mappings, 'DeleteMapping', { id })

// ============================================================================
// Templates
// ============================================================================
export const listTemplates = (pageSize = 500) =>
  call<{ templates?: Array<{ id: string; name: string }> }>(SVC.templates, 'ListTemplates', { pageSize })
export const createTemplate = (body: Record<string, unknown>) =>
  call<{ template?: { id: string } }>(SVC.templates, 'CreateTemplate', body)
export const deleteTemplate = (id: string) => call(SVC.templates, 'DeleteTemplate', { id })

// ============================================================================
// Proto sources / registry / codec / selections
// ============================================================================
export const listSources = (pageSize = 500) =>
  call<{ sources?: Array<{ id: string; name: string; sourceType?: string }> }>(SVC.sources, 'ListSources', { pageSize })
export const createSource = (body: Record<string, unknown>) =>
  call<{ source?: { id: string } }>(SVC.sources, 'CreateSource', body)
export const compileFiles = (sourceId: string) => call(SVC.sources, 'CompileFiles', { sourceId })
export const deleteSource = (id: string) => call(SVC.sources, 'DeleteSource', { id })
export const validateFiles = (body: Record<string, unknown>) => call<Record<string, unknown>>(SVC.sources, 'ValidateFiles', body)
export const listProtoMessages = (body: Record<string, unknown>) =>
  call<{ messages?: Array<{ fullName?: string }> }>(SVC.registry, 'ListProtoMessages', body)
export const generateExample = (body: Record<string, unknown>) => call<Record<string, unknown>>(SVC.registry, 'GenerateExample', body)
export const encodeMessage = (body: Record<string, unknown>) => call<Record<string, unknown>>(SVC.codec, 'EncodeMessage', body)
export const decodeMessage = (body: Record<string, unknown>) => call<Record<string, unknown>>(SVC.codec, 'DecodeMessage', body)
// ValidateJson lives on PublishService (not CodecService).
export const validateJson = (body: Record<string, unknown>) => call<Record<string, unknown>>(SVC.publish, 'ValidateJson', body)

// ============================================================================
// Settings / History / Workspace / Stats
// ============================================================================
export const getSettings = () => call<Record<string, unknown>>(SVC.settings, 'GetSettings', {})
export const updateSettings = (body: Record<string, unknown>) => call(SVC.settings, 'UpdateSettings', body)
export const resetSettings = () => call(SVC.settings, 'ResetSettings', {})
export const listPublishHistory = (body: Record<string, unknown> = { pageSize: 100 }) =>
  call<{ entries?: unknown[] }>(SVC.history, 'ListPublishHistory', body)
export const exportWorkspace = (body: Record<string, unknown> = {}) => call<Record<string, unknown>>(SVC.workspace, 'ExportWorkspace', body)
export const importWorkspace = (body: Record<string, unknown>) => call<Record<string, unknown>>(SVC.workspace, 'ImportWorkspace', body)
export const validateWorkspace = (body: Record<string, unknown>) => call<Record<string, unknown>>(SVC.workspace, 'ValidateWorkspace', body)
export const listSections = () => call<{ sections?: unknown[] }>(SVC.workspace, 'ListSections', {})
export const getServerInfo = (connectionId: string) => call<Record<string, unknown>>(SVC.stats, 'GetServerInfo', { connectionId })
