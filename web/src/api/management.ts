import type { Duration, Timestamp } from '@bufbuild/protobuf/wkt'
import { ConnectError } from '@connectrpc/connect'
import { getErrorReason } from '@/api/errors'
import { durToNanos, nanosToDur, tsToMillis } from '@/utils/timestamp'
import { encodeBytesToBase64 } from '@/utils/base64'
import type { StreamInfo, ConsumerInfo } from '../types/nats'
import type {
  StreamCreateRequest,
  StreamUpdateRequest,
  StreamPurgeRequest,
  StreamSource,
  RePublishConfig,
  SubjectTransformConfig,
  ConsumerLimitsConfig,
  PlacementConfig,
  ConsumerCreateRequest,
  ConsumerUpdateRequest,
  ConsumerPauseResponse,
  KVBucketConfig,
  KVBucketInfo,
  KVEntry,
  ObjectBucketConfig,
  ObjectBucketInfo,
  ObjectInfo,
  ObjectGetResponse,
  PurgeResponse,
  StatusResponse,
  RevisionResponse,
} from '../types/management'
import { managementClient } from './grpc/clients'
import { toStreamInfo, toConsumerInfo, RETENTION_INT, STORAGE_INT, STORAGE_STR, DISCARD_INT, COMPRESSION_INT, DELIVER_POLICY_INT, ACK_POLICY_INT, REPLAY_POLICY_INT } from './streams'

function toPlacementInit(p: PlacementConfig) {
  return { cluster: p.cluster ?? '', tags: p.tags ?? [] }
}

function toStreamSourceInit(s: StreamSource) {
  return {
    name: s.name,
    optStartSeq: s.opt_start_seq != null ? BigInt(s.opt_start_seq) : BigInt(0),
    optStartTime: s.opt_start_time,
    filterSubject: s.filter_subject ?? '',
    subjectTransforms: (s.subject_transforms ?? []).map((t) => ({ source: t.src, destination: t.dest })),
    external: s.external
      ? { apiPrefix: s.external.api, deliverPrefix: s.external.deliver ?? '' }
      : undefined,
  }
}

function toRePublishInit(r: RePublishConfig) {
  return { src: r.src, dest: r.dest, headersOnly: r.headers_only ?? false }
}

function toSubjectTransformInit(t: SubjectTransformConfig) {
  return { source: t.src, destination: t.dest }
}

function toConsumerLimitsInit(l: ConsumerLimitsConfig) {
  return {
    inactiveThreshold: l.inactive_threshold != null ? nanosToDur(l.inactive_threshold) : undefined,
    maxAckPending: l.max_ack_pending ?? 0,
  }
}

export async function createStream(
  connectionId: string,
  config: StreamCreateRequest
): Promise<StreamInfo> {
  const response = await managementClient.createStream({
    connectionId,
    name: config.name,
    description: config.description ?? '',
    subjects: config.subjects ?? [],
    retention: RETENTION_INT[config.retention ?? ''] ?? 0,
    storage: STORAGE_INT[config.storage ?? ''] ?? 0,
    discard: DISCARD_INT[config.discard ?? ''] ?? 0,
    maxMsgs: config.max_msgs != null ? BigInt(config.max_msgs) : BigInt(0),
    maxBytes: config.max_bytes != null ? BigInt(config.max_bytes) : BigInt(0),
    maxAge: config.max_age != null ? nanosToDur(config.max_age) : undefined,
    maxMsgsPerSubject: config.max_msgs_per_subject != null ? BigInt(config.max_msgs_per_subject) : BigInt(0),
    maxMsgSize: config.max_msg_size ?? 0,
    maxConsumers: config.max_consumers ?? 0,
    replicas: config.num_replicas ?? 0,
    duplicates: config.duplicate_window != null ? nanosToDur(config.duplicate_window) : undefined,
    denyDelete: config.deny_delete ?? false,
    denyPurge: config.deny_purge ?? false,
    allowRollup: config.allow_rollup ?? false,
    allowDirect: config.allow_direct ?? false,
    mirrorDirect: config.mirror_direct ?? false,
    discardNewPerSubject: config.discard_new_per_subject ?? false,
    noAck: config.no_ack ?? false,
    allowMsgTtl: config.allow_msg_ttl ?? false,
    allowAtomicPublish: config.allow_atomic_publish ?? false,
    compression: COMPRESSION_INT[config.compression ?? ''] ?? 0,
    firstSeq: config.first_seq != null ? BigInt(config.first_seq) : BigInt(0),
    placement: config.placement ? toPlacementInit(config.placement) : undefined,
    mirror: config.mirror ? toStreamSourceInit(config.mirror) : undefined,
    sources: (config.sources ?? []).map(toStreamSourceInit),
    subjectTransform: config.subject_transform ? toSubjectTransformInit(config.subject_transform) : undefined,
    republish: config.republish ? toRePublishInit(config.republish) : undefined,
    consumerLimits: config.consumer_limits ? toConsumerLimitsInit(config.consumer_limits) : undefined,
    metadata: config.metadata ?? {},
  })
  return toStreamInfo(response.stream!)
}

export async function updateStream(
  connectionId: string,
  name: string,
  config: StreamUpdateRequest
): Promise<StreamInfo> {
  const response = await managementClient.updateStream({
    connectionId,
    streamName: name,
    subjects: config.subjects ?? [],
    description: config.description,
    maxMsgs: config.max_msgs != null ? BigInt(config.max_msgs) : undefined,
    maxBytes: config.max_bytes != null ? BigInt(config.max_bytes) : undefined,
    maxAge: config.max_age != null ? nanosToDur(config.max_age) : undefined,
    maxMsgsPerSubject: config.max_msgs_per_subject != null ? BigInt(config.max_msgs_per_subject) : undefined,
    maxMsgSize: config.max_msg_size,
    maxConsumers: config.max_consumers,
    duplicates: config.duplicate_window != null ? nanosToDur(config.duplicate_window) : undefined,
    discard: config.discard != null ? (DISCARD_INT[config.discard] ?? undefined) : undefined,
    discardNewPerSubject: config.discard_new_per_subject,
    allowDirect: config.allow_direct,
    mirrorDirect: config.mirror_direct,
    sources: (config.sources ?? []).map(toStreamSourceInit),
    compression: config.compression != null ? (COMPRESSION_INT[config.compression] ?? undefined) : undefined,
    republish: config.republish ? toRePublishInit(config.republish) : undefined,
    subjectTransform: config.subject_transform ? toSubjectTransformInit(config.subject_transform) : undefined,
    consumerLimits: config.consumer_limits ? toConsumerLimitsInit(config.consumer_limits) : undefined,
    allowMsgTtl: config.allow_msg_ttl,
    allowAtomicPublish: config.allow_atomic_publish,
    metadata: config.metadata ?? {},
  })
  return toStreamInfo(response.stream!)
}

export async function deleteStream(
  connectionId: string,
  name: string
): Promise<void> {
  await managementClient.deleteStream({ connectionId, streamName: name })
}

export async function purgeStream(
  connectionId: string,
  name: string,
  options?: StreamPurgeRequest
): Promise<PurgeResponse> {
  const response = await managementClient.purgeStream({
    connectionId,
    streamName: name,
    filter: options?.filter,
    sequence: options?.seq != null ? BigInt(options.seq) : undefined,
    keep: options?.keep != null ? BigInt(options.keep) : undefined,
  })
  return { purged: Number(response.purged) }
}

export async function sealStream(
  connectionId: string,
  name: string
): Promise<StatusResponse> {
  await managementClient.sealStream({ connectionId, streamName: name })
  return { status: 'ok' }
}

/**
 * Delete one message by sequence. `secure` overwrites data before removal.
 * Rejects FailedPrecondition on deny_delete, NotFound on missing sequence.
 */
export async function deleteMessage(
  connectionId: string,
  streamName: string,
  sequence: number,
  secure: boolean
): Promise<void> {
  await managementClient.deleteMessage({
    connectionId,
    streamName,
    sequence: BigInt(sequence),
    secure,
  })
}

export async function listConsumers(
  connectionId: string,
  streamName: string,
  signal?: AbortSignal,
): Promise<ConsumerInfo[]> {
  const response = await managementClient.listConsumers({
    connectionId,
    streamName,
  }, { signal })
  return response.consumers.map(toConsumerInfo)
}

export async function createConsumer(
  connectionId: string,
  streamName: string,
  config: ConsumerCreateRequest
): Promise<ConsumerInfo> {
  const response = await managementClient.createConsumer({
    connectionId,
    streamName,
    // `name` is initialized to '' not undefined, so `??` never falls through
    // to durable_name — use `||` so an untouched Name field still works.
    name: config.name || config.durable_name || '',
    description: config.description ?? '',
    deliverPolicy: DELIVER_POLICY_INT[config.deliver_policy ?? ''] ?? 0,
    optStartSeq: config.opt_start_seq != null ? BigInt(config.opt_start_seq) : BigInt(0),
    optStartTime: config.opt_start_time ?? '',
    ackPolicy: ACK_POLICY_INT[config.ack_policy ?? ''] ?? 0,
    ackWait: config.ack_wait != null ? nanosToDur(config.ack_wait) : undefined,
    maxDeliver: config.max_deliver ?? 0,
    backOff: (config.backoff ?? []).map(nanosToDur),
    filterSubject: config.filter_subject ?? '',
    filterSubjects: config.filter_subjects ?? [],
    replayPolicy: REPLAY_POLICY_INT[config.replay_policy ?? ''] ?? 0,
    rateLimit: config.rate_limit_bps != null ? BigInt(config.rate_limit_bps) : BigInt(0),
    sampleFrequency: config.sample_freq ?? '',
    maxWaiting: config.max_waiting ?? 0,
    maxAckPending: config.max_ack_pending ?? 0,
    headersOnly: config.headers_only ?? false,
    maxRequestBatch: config.max_batch ?? 0,
    maxRequestMaxBytes: config.max_bytes != null ? BigInt(config.max_bytes) : BigInt(0),
    maxRequestExpires: config.max_expires != null ? nanosToDur(config.max_expires) : undefined,
    inactiveThreshold: config.inactive_threshold != null ? nanosToDur(config.inactive_threshold) : undefined,
    replicas: config.num_replicas ?? 0,
    memoryStorage: config.memory_storage ?? false,
    metadata: config.metadata ?? {},
    deliverSubject: config.deliver_subject ?? '',
    deliverGroup: config.deliver_group ?? '',
    flowControl: config.flow_control ?? false,
    idleHeartbeat: config.idle_heartbeat != null ? nanosToDur(config.idle_heartbeat) : undefined,
    ephemeral: config.ephemeral ?? false,
  })
  return toConsumerInfo(response.consumer!)
}

export async function updateConsumer(
  connectionId: string,
  streamName: string,
  consumerName: string,
  config: ConsumerUpdateRequest
): Promise<ConsumerInfo> {
  const response = await managementClient.updateConsumer({
    connectionId,
    streamName,
    consumerName,
    description: config.description,
    ackWait: config.ack_wait != null ? nanosToDur(config.ack_wait) : undefined,
    maxDeliver: config.max_deliver,
    maxAckPending: config.max_ack_pending,
    maxWaiting: config.max_waiting,
    rateLimit: config.rate_limit_bps != null ? BigInt(config.rate_limit_bps) : undefined,
    sampleFrequency: config.sample_freq,
    maxRequestBatch: config.max_batch,
    maxRequestMaxBytes: config.max_bytes != null ? BigInt(config.max_bytes) : undefined,
    maxRequestExpires: config.max_expires != null ? nanosToDur(config.max_expires) : undefined,
    inactiveThreshold: config.inactive_threshold != null ? nanosToDur(config.inactive_threshold) : undefined,
    backOff: (config.backoff ?? []).map(nanosToDur),
    metadata: config.metadata ?? {},
    filterSubject: config.filter_subject,
    filterSubjects: config.filter_subjects?.length ? config.filter_subjects : [],
  })
  return toConsumerInfo(response.consumer!)
}

export async function deleteConsumer(
  connectionId: string,
  streamName: string,
  consumerName: string
): Promise<void> {
  await managementClient.deleteConsumer({
    connectionId,
    streamName,
    consumerName,
  })
}

export async function pauseConsumer(
  connectionId: string,
  streamName: string,
  consumerName: string,
  pauseUntil: Date
): Promise<ConsumerPauseResponse> {
  const response = await managementClient.pauseConsumer({
    connectionId,
    streamName,
    consumerName,
    pauseUntil: pauseUntil.toISOString(),
  })
  return {
    paused: response.paused,
    pause_until: response.pauseUntil || undefined,
  }
}

export async function resumeConsumer(
  connectionId: string,
  streamName: string,
  consumerName: string
): Promise<StatusResponse> {
  await managementClient.resumeConsumer({
    connectionId,
    streamName,
    consumerName,
  })
  return { status: 'ok' }
}

function toKVBucketInfo(b: { bucket: string; description: string; values: bigint; bytes: bigint; history: number; ttl?: Duration; storage: number; numReplicas: number; isCompressed: boolean; metadata: Record<string, string> }): KVBucketInfo {
  return {
    bucket: b.bucket,
    description: b.description || undefined,
    values: Number(b.values),
    bytes: Number(b.bytes),
    history: b.history,
    ttl: durToNanos(b.ttl) || undefined,
    storage: STORAGE_STR[b.storage] ?? 'file',
    num_replicas: b.numReplicas,
    is_compressed: b.isCompressed || undefined,
    metadata: Object.keys(b.metadata).length > 0 ? b.metadata : undefined,
  }
}

export async function listKVBuckets(
  connectionId: string,
  signal?: AbortSignal,
): Promise<KVBucketInfo[]> {
  const response = await managementClient.listKVBuckets({ connectionId }, { signal })
  return response.buckets.map(toKVBucketInfo)
}

export async function createKVBucket(
  connectionId: string,
  config: KVBucketConfig
): Promise<KVBucketInfo> {
  const response = await managementClient.createKVBucket({
    connectionId,
    config: {
      bucket: config.bucket,
      description: config.description ?? '',
      maxValueSize: config.max_value_size ?? 0,
      history: config.history ?? 0,
      ttl: config.ttl != null ? nanosToDur(config.ttl) : undefined,
      maxBytes: config.max_bytes != null ? BigInt(config.max_bytes) : BigInt(0),
      storage: STORAGE_INT[config.storage ?? ''] ?? 0,
      numReplicas: config.num_replicas ?? 0,
      metadata: config.metadata ?? {},
    },
  })
  return toKVBucketInfo(response.bucket!)
}

export async function deleteKVBucket(
  connectionId: string,
  bucket: string
): Promise<void> {
  await managementClient.deleteKVBucket({ connectionId, bucket })
}

// Empty JetStream lists come back as NotFound; treated as success so react-query
function emptyOn<T>(reason: string, messageSubstring: string, fallback: T) {
  return (err: unknown): T => {
    if (err instanceof ConnectError) {
      if (getErrorReason(err) === reason) return fallback
      if (err.message.toLowerCase().includes(messageSubstring)) return fallback
    }
    throw err
  }
}

export async function listKVKeys(
  connectionId: string,
  bucket: string,
  signal?: AbortSignal,
): Promise<string[]> {
  const response = await managementClient
    .listKVKeys({ connectionId, bucket }, { signal })
    .catch(emptyOn('NATS_NO_KEYS', 'no keys found', { keys: [] }))
  return response.keys || []
}

export async function getKVKey(
  connectionId: string,
  bucket: string,
  key: string,
  signal?: AbortSignal,
): Promise<KVEntry> {
  const response = await managementClient.getKVKey({ connectionId, bucket, key }, { signal })
  const entry = response.entry!
  return {
    bucket,
    key: entry.key,
    value: entry.value,
    revision: Number(entry.revision),
    created: tsToMillis(entry.created),
    operation: entry.operation as 'put' | 'delete' | 'purge',
  }
}

export async function getKVKeyHistory(
  connectionId: string,
  bucket: string,
  key: string,
  signal?: AbortSignal,
): Promise<KVEntry[]> {
  const response = await managementClient.getKVKeyHistory(
    { connectionId, bucket, key },
    { signal },
  )
  return response.entries.map((entry) => ({
    bucket,
    key: entry.key,
    value: entry.value,
    revision: Number(entry.revision),
    created: tsToMillis(entry.created),
    operation: entry.operation as 'put' | 'delete' | 'purge',
  }))
}

export async function putKVKey(
  connectionId: string,
  bucket: string,
  key: string,
  value: string | Uint8Array,
  expectedRevision?: number
): Promise<RevisionResponse> {
  const base64Value =
    typeof value === 'string'
      ? encodeBytesToBase64(new TextEncoder().encode(value))
      : encodeBytesToBase64(value)

  const response = await managementClient.putKVKey({
    connectionId,
    bucket,
    key,
    value: base64Value,
    revision: expectedRevision != null ? BigInt(expectedRevision) : BigInt(0),
  })
  return { revision: Number(response.revision) }
}

export async function deleteKVKey(
  connectionId: string,
  bucket: string,
  key: string
): Promise<void> {
  await managementClient.deleteKVKey({ connectionId, bucket, key })
}

export async function purgeKVKey(
  connectionId: string,
  bucket: string,
  key: string
): Promise<void> {
  await managementClient.purgeKVKey({ connectionId, bucket, key })
}

function toObjectBucketInfo(b: { bucket: string; description: string; size: bigint; objects: bigint; storage: number; numReplicas: number; sealed: boolean; isCompressed: boolean; metadata: Record<string, string> }): ObjectBucketInfo {
  return {
    bucket: b.bucket,
    description: b.description || undefined,
    size: Number(b.size),
    objects: Number(b.objects),
    storage: STORAGE_STR[b.storage] ?? 'file',
    num_replicas: b.numReplicas,
    sealed: b.sealed || undefined,
    is_compressed: b.isCompressed || undefined,
    metadata: Object.keys(b.metadata).length > 0 ? b.metadata : undefined,
  }
}

function toObjectInfo(o: { name: string; description: string; size: bigint; chunks: number; modTime?: Timestamp; digest: string; nuid: string; deleted: boolean; metadata: Record<string, string> }): ObjectInfo {
  return {
    name: o.name,
    description: o.description || undefined,
    size: Number(o.size),
    chunks: o.chunks,
    mod_time: tsToMillis(o.modTime),
    digest: o.digest,
    nuid: o.nuid,
    deleted: o.deleted || undefined,
    metadata: Object.keys(o.metadata).length > 0 ? o.metadata : undefined,
  }
}

export async function listObjectBuckets(
  connectionId: string,
  signal?: AbortSignal,
): Promise<ObjectBucketInfo[]> {
  const response = await managementClient.listObjectBuckets({ connectionId }, { signal })
  return response.buckets.map(toObjectBucketInfo)
}

export async function createObjectBucket(
  connectionId: string,
  config: ObjectBucketConfig
): Promise<ObjectBucketInfo> {
  const response = await managementClient.createObjectBucket({
    connectionId,
    config: {
      bucket: config.bucket,
      description: config.description ?? '',
      ttl: config.ttl != null ? nanosToDur(config.ttl) : undefined,
      storage: STORAGE_INT[config.storage ?? ''] ?? 0,
      numReplicas: config.num_replicas ?? 0,
      maxBytes: config.max_bytes != null ? BigInt(config.max_bytes) : BigInt(0),
      metadata: config.metadata ?? {},
    },
  })
  return toObjectBucketInfo(response.bucket!)
}

export async function deleteObjectBucket(
  connectionId: string,
  bucket: string
): Promise<void> {
  await managementClient.deleteObjectBucket({ connectionId, bucket })
}

export async function sealObjectBucket(
  connectionId: string,
  bucket: string
): Promise<StatusResponse> {
  await managementClient.sealObjectBucket({ connectionId, bucket })
  return { status: 'ok' }
}

export async function listObjects(
  connectionId: string,
  bucket: string,
  signal?: AbortSignal,
): Promise<ObjectInfo[]> {
  const response = await managementClient
    .listObjects({ connectionId, bucket }, { signal })
    .catch(emptyOn('NATS_NO_OBJECTS', 'no objects found', { objects: [] }))
  return response.objects.map(toObjectInfo)
}

export async function getObject(
  connectionId: string,
  bucket: string,
  name: string,
  signal?: AbortSignal,
): Promise<ObjectGetResponse> {
  const response = await managementClient.getObject({ connectionId, bucket, name }, { signal })
  const data = response.data
  const base64 = data.length > 0
    ? encodeBytesToBase64(data)
    : ''
  return {
    data: base64,
    info: toObjectInfo(response.info!),
  }
}

export async function putObject(
  connectionId: string,
  bucket: string,
  name: string,
  data: string | Uint8Array,
  options?: { description?: string; metadata?: Record<string, string> }
): Promise<ObjectInfo> {
  let bytes: Uint8Array
  if (typeof data === 'string') {
    bytes = new TextEncoder().encode(data)
  } else {
    bytes = data
  }

  const response = await managementClient.putObject({
    connectionId,
    bucket,
    name,
    data: new Uint8Array(bytes) as Uint8Array<ArrayBuffer>,
    description: options?.description ?? '',
    metadata: options?.metadata ?? {},
  })
  return toObjectInfo(response.info!)
}

export async function deleteObject(
  connectionId: string,
  bucket: string,
  name: string
): Promise<void> {
  await managementClient.deleteObject({ connectionId, bucket, name })
}

export { decodeBase64ToUtf8 as decodeBase64 } from '@/utils/base64'
