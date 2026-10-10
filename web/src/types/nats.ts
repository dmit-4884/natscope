export interface ClusterInfo {
  name?: string
  leader?: string
  replicas?: ReplicaInfo[]
}

export interface ReplicaInfo {
  name: string
  current: boolean
  offline: boolean
  active: number
  lag: number
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
  allow_msg_counter?: boolean
  allow_msg_schedules?: boolean
  subject_delete_marker_ttl?: number
  persist_mode?: 'default' | 'async'
  allow_batched?: boolean
  mirror?: StreamSourceRef
  sources?: StreamSourceRef[]
  republish?: StreamRePublish
  subject_transform?: StreamSubjectTransform
}

export interface ExternalStreamRef {
  api_prefix: string
  deliver_prefix: string
}

export interface StreamSourceRef {
  name: string
  opt_start_seq?: number
  opt_start_time?: number
  filter_subject?: string
  subject_transforms?: StreamSubjectTransform[]
  external?: ExternalStreamRef
}

export interface StreamRePublish {
  src: string
  dest: string
  headers_only?: boolean
}

interface StreamSubjectTransform {
  src: string
  dest: string
}

export type StreamRelationKind = 'source' | 'mirror' | 'republish'

export type StreamNodeKind = 'stream' | 'kv' | 'object_store' | 'external' | 'missing' | 'subject'

export interface StreamRelationNode {
  id: string
  name: string
  kind: StreamNodeKind
  info?: StreamInfo
  external?: ExternalStreamRef
}

interface StreamLinkState {
  lag: number
  active_ns: number
  error?: string
}

export interface StreamRelationEdge {
  kind: StreamRelationKind
  from: string
  to: string
  source?: StreamSourceRef
  state?: StreamLinkState
  republish?: StreamRePublish
}

export interface StreamRelations {
  nodes: StreamRelationNode[]
  edges: StreamRelationEdge[]
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
  deliver_subject?: string
  deliver_group?: string
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
  max_bytes?: number
  inactive_threshold?: number
  num_replicas?: number
  mem_storage?: boolean
  metadata?: Record<string, string>
  priority_policy?: PriorityPolicy
  priority_groups?: string[]
  priority_timeout?: number
}

export type PriorityPolicy = 'none' | 'pinned_client' | 'overflow' | 'prioritized'

export interface PriorityGroupState {
  group: string
  pinned_client_id?: string
  pinned_ts?: number // Unix milliseconds
}

export interface SequenceInfo {
  consumer_seq: number
  stream_seq: number
  last_active?: number // Unix milliseconds
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
  /** When the server answered, Unix milliseconds on the server clock. */
  time_stamp?: number
  cluster?: ClusterInfo
  priority_groups?: PriorityGroupState[]
  raw?: Record<string, unknown>
}

export interface Message {
  sequence: number
  subject: string
  timestamp: number // Unix milliseconds
  data_base64: string
  data_size: number
  content_type: 'json' | 'text' | 'binary'
  headers?: Record<string, string>
  // Decoded protobuf fields. When truncated this may be a raw string preview
  // instead of a parsed object — UIs must accept both shapes.
  decoded?: Record<string, unknown> | string
  decoded_type?: string
  decode_error?: string
  decoded_unknown_fields?: number
  decoded_valid_bytes?: number
  decoded_auto?: boolean
  decoded_source_id?: string
  // When true, data_base64/decoded were capped per
  // settings.messages.maxPayloadBytesInList; use MessagesService.Get for full payload.
  truncated?: boolean
}

export interface StreamDetail extends StreamInfo {
  state: StreamState
  consumers: ConsumerInfo[]
}

