// Management types for streams, consumers, KV, and Object stores

// Stream Types

export interface PlacementConfig {
  cluster?: string
  tags?: string[]
}

export interface StreamSource {
  name: string
  opt_start_seq?: number
  opt_start_time?: string
  filter_subject?: string
  external?: {
    api: string
    deliver?: string
  }
  subject_transforms?: Array<{
    src: string
    dest: string
  }>
}

export interface RePublishConfig {
  src: string
  dest: string
  headers_only?: boolean
}

export interface SubjectTransformConfig {
  src: string
  dest: string
}

export interface ConsumerLimitsConfig {
  inactive_threshold?: number
  max_ack_pending?: number
}

export interface StreamCreateRequest {
  name: string
  description?: string
  subjects?: string[]
  retention?: 'limits' | 'interest' | 'workqueue'
  storage?: 'file' | 'memory'
  discard?: 'old' | 'new'
  max_msgs?: number
  max_bytes?: number
  max_age?: number
  max_msgs_per_subject?: number
  max_msg_size?: number
  max_consumers?: number
  num_replicas?: number
  duplicate_window?: number
  deny_delete?: boolean
  deny_purge?: boolean
  allow_direct?: boolean
  mirror_direct?: boolean
  allow_rollup?: boolean
  discard_new_per_subject?: boolean
  no_ack?: boolean
  allow_msg_ttl?: boolean
  allow_atomic_publish?: boolean
  compression?: 'none' | 's2'
  first_seq?: number
  placement?: PlacementConfig
  mirror?: StreamSource
  sources?: StreamSource[]
  republish?: RePublishConfig
  subject_transform?: SubjectTransformConfig
  consumer_limits?: ConsumerLimitsConfig
  metadata?: Record<string, string>
}

export interface StreamUpdateRequest {
  subjects?: string[]
  description?: string
  max_msgs?: number
  max_bytes?: number
  max_age?: number
  max_msgs_per_subject?: number
  max_msg_size?: number
  max_consumers?: number
  duplicate_window?: number
  discard?: 'old' | 'new'
  discard_new_per_subject?: boolean
  allow_direct?: boolean
  mirror_direct?: boolean
  sources?: StreamSource[]
  metadata?: Record<string, string>
  compression?: 'none' | 's2'
  republish?: RePublishConfig
  subject_transform?: SubjectTransformConfig
  consumer_limits?: ConsumerLimitsConfig
  allow_msg_ttl?: boolean
  allow_atomic_publish?: boolean
}

export interface StreamPurgeRequest {
  filter?: string
  keep?: number
  seq?: number
}

// Consumer Types

export interface ConsumerCreateRequest {
  name?: string
  durable_name?: string
  description?: string
  deliver_policy?: 'all' | 'last' | 'new' | 'by_start_sequence' | 'by_start_time' | 'last_per_subject'
  opt_start_seq?: number
  opt_start_time?: string
  ack_policy?: 'none' | 'all' | 'explicit'
  ack_wait?: number
  max_deliver?: number
  backoff?: number[]
  filter_subject?: string
  filter_subjects?: string[]
  replay_policy?: 'instant' | 'original'
  rate_limit_bps?: number
  sample_freq?: string
  max_waiting?: number
  max_ack_pending?: number
  headers_only?: boolean
  max_batch?: number
  max_expires?: number
  max_bytes?: number
  inactive_threshold?: number
  num_replicas?: number
  memory_storage?: boolean
  metadata?: Record<string, string>
  // Push consumer specific
  deliver_subject?: string
  deliver_group?: string
  flow_control?: boolean
  idle_heartbeat?: number
}

export interface ConsumerUpdateRequest {
  description?: string
  ack_wait?: number
  max_deliver?: number
  max_ack_pending?: number
  max_waiting?: number
  rate_limit_bps?: number
  sample_freq?: string
  inactive_threshold?: number
  backoff?: number[]
  max_batch?: number
  max_bytes?: number
  max_expires?: number
  metadata?: Record<string, string>
}

export interface ConsumerPauseResponse {
  paused: boolean
  pause_until?: string
  pause_remaining?: number
}

// Immutable fields that cannot be changed after consumer creation
export const CONSUMER_IMMUTABLE_FIELDS = [
  'name',
  'durable_name',
  'ack_policy',
  'deliver_policy',
  'filter_subject',
  'filter_subjects',
  'replay_policy',
  'opt_start_seq',
  'opt_start_time',
  'headers_only',
  'memory_storage',
  'num_replicas',
  'deliver_subject',
  'deliver_group',
  'flow_control',
  'idle_heartbeat',
] as const

// KeyValue Types

export interface KVBucketConfig {
  bucket: string
  description?: string
  max_value_size?: number
  history?: number
  ttl?: number
  max_bytes?: number
  storage?: 'file' | 'memory'
  num_replicas?: number
  placement?: PlacementConfig
  compression?: boolean
  metadata?: Record<string, string>
  mirror?: StreamSource
  sources?: StreamSource[]
  republish?: RePublishConfig
}

export interface KVBucketInfo {
  bucket: string
  description?: string
  values: number
  bytes: number
  history: number
  ttl?: number
  storage: string
  num_replicas: number
  is_compressed?: boolean
  metadata?: Record<string, string>
}

export interface KVEntry {
  bucket: string
  key: string
  value: string // base64 encoded
  revision: number
  created: number // Unix milliseconds
  operation: 'put' | 'delete' | 'purge'
}

// Object Store Types

export interface ObjectBucketConfig {
  bucket: string
  description?: string
  max_bytes?: number
  ttl?: number
  storage?: 'file' | 'memory'
  num_replicas?: number
  placement?: PlacementConfig
  compression?: boolean
  metadata?: Record<string, string>
}

export interface ObjectBucketInfo {
  bucket: string
  description?: string
  size: number
  objects: number
  storage: string
  num_replicas: number
  sealed?: boolean
  is_compressed?: boolean
  metadata?: Record<string, string>
}

export interface ObjectInfo {
  name: string
  description?: string
  size: number
  chunks: number
  mod_time: number // Unix milliseconds
  digest: string
  nuid: string
  deleted?: boolean
  headers?: Record<string, string[]>
  metadata?: Record<string, string>
}

// Response Types

export interface ObjectGetResponse {
  data: string // base64 encoded
  info: ObjectInfo
}

export interface PurgeResponse {
  purged: number
}

export interface StatusResponse {
  status: string
}

export interface RevisionResponse {
  revision: number
}
