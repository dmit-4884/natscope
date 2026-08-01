export interface ClusterInfo {
  name?: string
  leader?: string
  replicas?: Array<{
    name: string
    current: boolean
    active: number
  }>
}

export interface StreamInfo {
  name: string
  description?: string
  subjects: string[]
  messages: number
  bytes: number
  consumer_count: number
  created: number // Unix milliseconds
  config: StreamConfig
  state?: StreamState
  consumers?: ConsumerInfo[]
  cluster?: ClusterInfo
  raw?: Record<string, unknown>
}

export interface StreamConfig {
  retention: string
  max_msgs: number
  max_bytes: number
  max_age: number
  max_consumers?: number
  max_msgs_per_subject?: number
  max_msg_size?: number
  storage?: string
  discard?: string
  discard_new_per_subject?: boolean
  num_replicas?: number
  duplicate_window?: number
  compression?: string
  sealed?: boolean
  deny_delete?: boolean
  deny_purge?: boolean
  allow_rollup_hdrs?: boolean
  allow_direct?: boolean
  mirror_direct?: boolean
  metadata?: Record<string, string>
  allow_msg_ttl?: boolean
  consumer_limits?: StreamConsumerLimits
  allow_atomic?: boolean
  mirror?: StreamSourceRef
  sources?: StreamSourceRef[]
  republish?: StreamRePublish
  subject_transform?: StreamSubjectTransform
}

export interface StreamSourceRef {
  name: string
  opt_start_seq?: number
  filter_subject?: string
  external?: { api_prefix: string; deliver_prefix: string }
}

export interface StreamRePublish {
  src: string
  dest: string
  headers_only?: boolean
}

export interface StreamSubjectTransform {
  src: string
  dest: string
}

export interface StreamConsumerLimits {
  inactive_threshold?: number
  max_ack_pending?: number
}

export interface StreamState {
  messages: number
  bytes: number
  first_seq: number
  last_seq: number
  first_ts: number // Unix milliseconds
  last_ts: number // Unix milliseconds
  consumer_count?: number
}

export interface ConsumerConfig {
  durable_name?: string
  description?: string
  deliver_policy?: string
  opt_start_seq?: number
  opt_start_time?: string
  ack_policy?: string
  ack_wait?: number
  max_deliver?: number
  backoff?: number[]
  filter_subject?: string
  filter_subjects?: string[]
  replay_policy?: string
  rate_limit_bps?: number
  sample_freq?: string
  max_waiting?: number
  max_ack_pending?: number
  flow_control?: boolean
  idle_heartbeat?: number
  headers_only?: boolean
  max_batch?: number
  max_expires?: number
  inactive_threshold?: number
  num_replicas?: number
  mem_storage?: boolean
  metadata?: Record<string, string>
}

interface SequenceInfo {
  consumer_seq: number
  stream_seq: number
}

export interface ConsumerInfo {
  name: string
  stream_name?: string
  config?: ConsumerConfig
  created?: number | null // Unix milliseconds
  delivered?: SequenceInfo
  ack_floor?: SequenceInfo
  num_pending: number
  num_ack_pending: number
  num_redelivered?: number
  num_waiting?: number
  push_bound?: boolean
  paused?: boolean
  /** RFC3339 instant the pause lifts; only meaningful while paused. */
  pause_until?: string
  cluster?: ClusterInfo
  raw?: Record<string, unknown>
}

export interface Message {
  sequence: number
  subject: string
  timestamp: number // Unix milliseconds
  data_base64: string
  data_raw_hex?: string
  data_size: number
  content_type: 'json' | 'text' | 'binary'
  headers?: Record<string, string>
  // Decoded protobuf fields. When truncated this may be a raw string preview
  // instead of a parsed object — UIs must accept both shapes.
  decoded?: Record<string, unknown> | string
  decoded_type?: string
  decode_error?: string
  // When true, data_base64/decoded were capped per
  // settings.messages.maxPayloadBytesInList; use MessagesService.Get for full payload.
  truncated?: boolean
}

export interface StreamDetail extends StreamInfo {
  state: StreamState
  consumers: ConsumerInfo[]
}

