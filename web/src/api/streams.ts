import { durToNanos, tsToMillis } from '@/utils/timestamp'
import type { StreamInfo as ProtoStreamInfo, StreamConfig as ProtoStreamConfig, StreamState as ProtoStreamState, ConsumerInfo as ProtoConsumerInfo, ConsumerConfig as ProtoConsumerConfig, ClusterInfo as ProtoClusterInfo, ConsumerLimits as ProtoConsumerLimits } from '../gen/types/nats/nats_stream_pb'
import type { StreamInfo, StreamDetail, StreamConfig, StreamConsumerLimits, StreamState, ConsumerInfo, ConsumerConfig, ClusterInfo } from '../types/nats'
import { streamsClient } from './grpc/clients'

export interface GetStreamsParams {
  connection_id: string
}

export interface StreamsResponse {
  streams: StreamInfo[]
}

// Enum int→string mappings (match Go entity iota ordering)

const RETENTION_STR: Record<number, string> = { 0: 'limits', 1: 'interest', 2: 'workqueue' }
export const RETENTION_INT: Record<string, number> = { limits: 0, interest: 1, workqueue: 2 }

export const STORAGE_STR: Record<number, string> = { 0: 'file', 1: 'memory' }
export const STORAGE_INT: Record<string, number> = { file: 0, memory: 1 }

const DISCARD_STR: Record<number, string> = { 0: 'old', 1: 'new' }
export const DISCARD_INT: Record<string, number> = { old: 0, new: 1 }

const COMPRESSION_STR: Record<number, string> = { 0: 'none', 1: 's2' }
export const COMPRESSION_INT: Record<string, number> = { none: 0, s2: 1 }

const DELIVER_POLICY_STR: Record<number, string> = { 0: 'all', 1: 'last', 2: 'new', 3: 'by_start_sequence', 4: 'by_start_time', 5: 'last_per_subject' }
export const DELIVER_POLICY_INT: Record<string, number> = { all: 0, last: 1, new: 2, by_start_sequence: 3, by_start_time: 4, last_per_subject: 5 }

const ACK_POLICY_STR: Record<number, string> = { 0: 'explicit', 1: 'all', 2: 'none' }
export const ACK_POLICY_INT: Record<string, number> = { explicit: 0, all: 1, none: 2 }

const REPLAY_POLICY_STR: Record<number, string> = { 0: 'instant', 1: 'original' }
export const REPLAY_POLICY_INT: Record<string, number> = { instant: 0, original: 1 }

// Proto → legacy type converters

function toClusterInfo(c: ProtoClusterInfo | undefined): ClusterInfo | undefined {
  if (!c) return undefined
  return {
    name: c.name || undefined,
    leader: c.leader || undefined,
    replicas: c.replicas.length > 0 ? c.replicas.map(r => ({
      name: r.name,
      current: r.current,
      active: durToNanos(r.active),
    })) : undefined,
  }
}

function toConsumerConfig(c: ProtoConsumerConfig | undefined): ConsumerConfig | undefined {
  if (!c) return undefined
  return {
    durable_name: c.durable || undefined,
    description: c.description || undefined,
    deliver_policy: DELIVER_POLICY_STR[c.deliverPolicy] || undefined,
    opt_start_seq: Number(c.optStartSeq) || undefined,
    opt_start_time: c.optStartTime || undefined,
    ack_policy: ACK_POLICY_STR[c.ackPolicy] || undefined,
    ack_wait: durToNanos(c.ackWait) || undefined,
    max_deliver: c.maxDeliver || undefined,
    backoff: c.backOff.length > 0 ? c.backOff.map(durToNanos) : undefined,
    filter_subject: c.filterSubject || undefined,
    filter_subjects: c.filterSubjects.length > 0 ? c.filterSubjects : undefined,
    replay_policy: REPLAY_POLICY_STR[c.replayPolicy] || undefined,
    rate_limit_bps: Number(c.rateLimit) || undefined,
    sample_freq: c.sampleFrequency || undefined,
    max_waiting: c.maxWaiting || undefined,
    max_ack_pending: c.maxAckPending || undefined,
    flow_control: c.flowControl || undefined,
    idle_heartbeat: durToNanos(c.idleHeartbeat) || undefined,
    headers_only: c.headersOnly || undefined,
    max_batch: c.maxRequestBatch || undefined,
    max_expires: durToNanos(c.maxRequestExpires) || undefined,
    inactive_threshold: durToNanos(c.inactiveThreshold) || undefined,
    num_replicas: c.replicas || undefined,
    mem_storage: c.memoryStorage || undefined,
    metadata: Object.keys(c.metadata).length > 0 ? c.metadata : undefined,
  }
}

/**
 * Pause state lives only in the server's raw ConsumerInfo JSON (`paused`,
 * `config.pause_until`) — the proto contract carries neither field.
 */
export function readConsumerPauseState(
  raw: Record<string, unknown> | undefined,
): { paused: boolean; pauseUntil: string | undefined } {
  if (!raw) return { paused: false, pauseUntil: undefined }

  const config = raw.config
  const pauseUntilRaw =
    config !== null && typeof config === 'object'
      ? (config as Record<string, unknown>).pause_until
      : undefined

  return {
    paused: raw.paused === true,
    pauseUntil: typeof pauseUntilRaw === 'string' && pauseUntilRaw ? pauseUntilRaw : undefined,
  }
}

export function toConsumerInfo(c: ProtoConsumerInfo): ConsumerInfo {
  let raw: Record<string, unknown> | undefined
  if (c.raw) {
    try { raw = JSON.parse(c.raw) } catch { /* ignore */ }
  }
  const pause = readConsumerPauseState(raw)
  return {
    name: c.name,
    stream_name: c.stream || undefined,
    config: toConsumerConfig(c.config),
    created: c.created != null ? tsToMillis(c.created) : undefined,
    delivered: {
      consumer_seq: Number(c.delivered?.consumer ?? 0n),
      stream_seq: Number(c.delivered?.stream ?? 0n),
    },
    ack_floor: {
      consumer_seq: Number(c.ackFloor?.consumer ?? 0n),
      stream_seq: Number(c.ackFloor?.stream ?? 0n),
    },
    num_pending: Number(c.numPending),
    num_ack_pending: c.numAckPending,
    num_redelivered: c.numRedelivered || undefined,
    num_waiting: c.numWaiting || undefined,
    push_bound: c.pushBound || undefined,
    paused: pause.paused,
    pause_until: pause.pauseUntil,
    cluster: toClusterInfo(c.cluster),
    raw,
  }
}

function toStreamConfig(c: ProtoStreamConfig | undefined): StreamConfig {
  if (!c) return { retention: '', max_msgs: 0, max_bytes: 0, max_age: 0 }
  return {
    retention: RETENTION_STR[c.retention] ?? '',
    max_msgs: Number(c.maxMsgs),
    max_bytes: Number(c.maxBytes),
    max_age: durToNanos(c.maxAge),
    max_consumers: c.maxConsumers || undefined,
    max_msgs_per_subject: Number(c.maxMsgsPerSubject) || undefined,
    max_msg_size: c.maxMsgSize || undefined,
    storage: STORAGE_STR[c.storage] ?? undefined,
    discard: DISCARD_STR[c.discard] ?? undefined,
    discard_new_per_subject: c.discardNewPerSubject || undefined,
    num_replicas: c.replicas || undefined,
    duplicate_window: durToNanos(c.duplicates) || undefined,
    compression: COMPRESSION_STR[c.compression] ?? undefined,
    sealed: c.sealed || undefined,
    deny_delete: c.denyDelete || undefined,
    deny_purge: c.denyPurge || undefined,
    allow_rollup_hdrs: c.allowRollup || undefined,
    allow_direct: c.allowDirect || undefined,
    mirror_direct: c.mirrorDirect || undefined,
    metadata: Object.keys(c.metadata).length > 0 ? c.metadata : undefined,
    allow_msg_ttl: c.allowMsgTtl || undefined,
    allow_atomic: c.allowAtomicPublish || undefined,
    mirror: c.mirror ? {
      name: c.mirror.name,
      opt_start_seq: Number(c.mirror.optStartSeq) || undefined,
      filter_subject: c.mirror.filterSubject || undefined,
      external: c.mirror.external ? {
        api_prefix: c.mirror.external.apiPrefix,
        deliver_prefix: c.mirror.external.deliverPrefix,
      } : undefined,
    } : undefined,
    sources: c.sources?.length ? c.sources.map(s => ({
      name: s.name,
      opt_start_seq: Number(s.optStartSeq) || undefined,
      filter_subject: s.filterSubject || undefined,
      external: s.external ? {
        api_prefix: s.external.apiPrefix,
        deliver_prefix: s.external.deliverPrefix,
      } : undefined,
    })) : undefined,
    republish: c.republish ? {
      src: c.republish.src,
      dest: c.republish.dest,
      headers_only: c.republish.headersOnly || undefined,
    } : undefined,
    subject_transform: c.subjectTransform ? {
      src: c.subjectTransform.source,
      dest: c.subjectTransform.destination,
    } : undefined,
    consumer_limits: toStreamConsumerLimits(c.consumerLimits),
  }
}

/** Consumer limits are only meaningful when at least one default is set. */
function toStreamConsumerLimits(l: ProtoConsumerLimits | undefined): StreamConsumerLimits | undefined {
  if (!l) return undefined
  const inactiveThreshold = durToNanos(l.inactiveThreshold) || undefined
  const maxAckPending = l.maxAckPending || undefined
  if (inactiveThreshold === undefined && maxAckPending === undefined) return undefined
  return { inactive_threshold: inactiveThreshold, max_ack_pending: maxAckPending }
}

function toStreamState(s: ProtoStreamState | undefined): StreamState | undefined {
  if (!s) return undefined
  return {
    messages: Number(s.msgs),
    bytes: Number(s.bytes),
    first_seq: Number(s.firstSeq),
    last_seq: Number(s.lastSeq),
    first_ts: tsToMillis(s.firstTime),
    last_ts: tsToMillis(s.lastTime),
    consumer_count: s.consumers || undefined,
  }
}

export function toStreamInfo(s: ProtoStreamInfo): StreamInfo {
  const state = toStreamState(s.state)
  let raw: Record<string, unknown> | undefined
  if (s.raw) {
    try { raw = JSON.parse(s.raw) } catch { /* ignore */ }
  }
  return {
    name: s.config?.name ?? '',
    description: s.config?.description || undefined,
    subjects: s.config?.subjects ?? [],
    messages: state?.messages ?? 0,
    bytes: state?.bytes ?? 0,
    consumer_count: state?.consumer_count ?? 0,
    created: tsToMillis(s.created),
    config: toStreamConfig(s.config),
    state,
    cluster: toClusterInfo(s.cluster),
    raw,
  }
}

export async function getStreams(
  params: GetStreamsParams,
  signal?: AbortSignal,
): Promise<StreamsResponse> {
  const response = await streamsClient.listStreams({
    connectionId: params.connection_id,
  }, { signal })
  return { streams: response.streams.map(toStreamInfo) }
}

export async function getStreamDetail(
  streamName: string,
  connectionId: string,
  signal?: AbortSignal,
): Promise<StreamDetail> {
  const response = await streamsClient.getStream({
    connectionId,
    streamName,
  }, { signal })
  const info = toStreamInfo(response.stream!)
  return {
    ...info,
    state: info.state ?? { messages: 0, bytes: 0, first_seq: 0, last_seq: 0, first_ts: 0, last_ts: 0 },
    consumers: info.consumers ?? [],
  }
}
