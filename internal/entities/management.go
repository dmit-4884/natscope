// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

import "time"

// StreamCreateRequest is the create input for a new JetStream stream.
type StreamCreateRequest struct {
	// Name is the unique stream name (required, immutable after creation).
	Name string `normalize:"trim"`

	// Description is an optional stream description.
	Description string `normalize:"trim"`

	// Subjects is the list of subject patterns this stream will capture
	// (required).
	Subjects []string

	// Retention is the message retention policy.
	Retention RetentionPolicy

	// Storage is the storage type.
	Storage StorageType

	// Discard is the discard policy when limits are reached.
	Discard DiscardPolicy

	// MaxMsgs is the maximum number of messages in the stream (-1 for unlimited).
	MaxMsgs int64

	// MaxBytes is the maximum total size in bytes (-1 for unlimited).
	MaxBytes int64

	// MaxAge is the maximum message age (0 for unlimited).
	MaxAge time.Duration

	// MaxMsgsPerSubject is the maximum messages per subject (-1 for unlimited).
	MaxMsgsPerSubject int64

	// MaxMsgSize is the maximum individual message size in bytes (-1 for
	// unlimited).
	MaxMsgSize int32

	// MaxConsumers is the maximum number of consumers (-1 for unlimited).
	MaxConsumers int

	// Replicas is the number of stream replicas (1, 3, or 5).
	Replicas int

	// Duplicates is the duplicate detection window.
	Duplicates time.Duration

	// DenyDelete prevents message deletion when true.
	DenyDelete bool

	// DenyPurge prevents stream purging when true.
	DenyPurge bool

	// AllowRollup allows rollup headers.
	AllowRollup bool

	// AllowDirect enables direct get operations.
	AllowDirect bool

	// MirrorDirect enables direct get on mirrors.
	MirrorDirect bool

	// DiscardNewPerSubject applies discard policy per subject.
	DiscardNewPerSubject bool

	// Compression is the compression algorithm.
	Compression StoreCompression

	// FirstSeq is the initial sequence number for the stream.
	FirstSeq uint64

	// Placement specifies cluster placement preferences.
	Placement *Placement

	// Mirror configures this stream as a mirror of another stream.
	Mirror *StreamSource

	// Sources configures this stream to aggregate from other streams.
	Sources []*StreamSource

	// SubjectTransform configures subject transformation on storage.
	SubjectTransform *SubjectTransformConfig

	// Metadata is custom key-value metadata.
	Metadata map[string]string

	// NoAck disables message acknowledgments.
	NoAck bool

	// AllowMsgTTL allows per-message TTL via Nats-TTL header (NATS 2.11+).
	AllowMsgTTL bool

	// AllowAtomicPublish allows atomic batch publish operations.
	AllowAtomicPublish bool

	// Republish configures re-publishing of incoming messages to another subject.
	Republish *StreamRePublish

	// ConsumerLimits are the default limits inherited by all consumers on this
	// stream.
	ConsumerLimits *StreamConsumerLimits
}

// StreamUpdateRequest holds mutable stream fields; pointers distinguish "not
// sent" (nil) from "set to zero value".
type StreamUpdateRequest struct {
	// Subjects is the list of subject patterns (mutable).
	Subjects []string

	// Description is the stream description (mutable).
	Description *string `normalize:"trim,nil_on_empty"`

	// MaxMsgs is the maximum number of messages (mutable).
	MaxMsgs *int64

	// MaxBytes is the maximum total size in bytes (mutable).
	MaxBytes *int64

	// MaxAge is the maximum message age (mutable).
	MaxAge *time.Duration

	// MaxMsgsPerSubject is the maximum messages per subject (mutable).
	MaxMsgsPerSubject *int64

	// MaxMsgSize is the maximum individual message size (mutable).
	MaxMsgSize *int32

	// MaxConsumers is the maximum number of consumers (mutable).
	MaxConsumers *int

	// Duplicates is the duplicate detection window (mutable).
	Duplicates *time.Duration

	// AllowDirect enables direct get operations (mutable).
	AllowDirect *bool

	// MirrorDirect enables direct get on mirrors (mutable).
	MirrorDirect *bool

	// Discard policy (mutable).
	Discard *DiscardPolicy

	// DiscardNewPerSubject applies discard policy per subject (mutable).
	DiscardNewPerSubject *bool

	// Sources to add (can only add, not remove).
	Sources []*StreamSource

	// Metadata is custom key-value metadata (mutable).
	Metadata map[string]string

	// Compression is the storage compression algorithm (mutable since NATS 2.10).
	Compression *StoreCompression

	// Republish configures re-publishing (mutable). Empty struct = disable.
	Republish *StreamRePublish

	// SubjectTransform mutates subject transform (mutable).
	SubjectTransform *SubjectTransformConfig

	// ConsumerLimits updates default consumer limits (mutable).
	ConsumerLimits *StreamConsumerLimits

	// AllowMsgTTL updates per-message TTL allowance (mutable since NATS 2.11).
	AllowMsgTTL *bool

	// AllowAtomicPublish updates atomic publish allowance (mutable).
	AllowAtomicPublish *bool
}

// StreamPurgeRequest is the input for purging messages from a stream.
type StreamPurgeRequest struct {
	// Filter is an optional subject filter to purge only matching messages.
	Filter string

	// Sequence purges all messages up to and including this sequence.
	Sequence uint64

	// Keep preserves this many messages (from the end).
	Keep uint64
}

// Placement specifies cluster placement preferences.
type Placement struct {
	// Cluster is the preferred cluster name.
	Cluster string

	// Tags are placement tags for server selection.
	Tags []string
}

// StreamSource is a stream source for mirroring or multi-source feeds.
type StreamSource struct {
	// Name is the source stream name.
	Name string

	// OptStartSeq is the starting sequence number.
	OptStartSeq uint64

	// OptStartTime is the starting time.
	OptStartTime *time.Time

	// FilterSubject filters messages by subject.
	FilterSubject string

	// SubjectTransforms applies subject transformations.
	SubjectTransforms []*SubjectTransformConfig

	// External configures cross-account/domain access.
	External *ExternalStream
}

// ExternalStream is the cross-account stream config for account/domain bridges.
type ExternalStream struct {
	// APIPrefix is the API prefix for the remote account.
	APIPrefix string

	// DeliverPrefix is the delivery prefix for the remote account.
	DeliverPrefix string
}

// SubjectTransformConfig is a subject transform rule applied to incoming messages.
type SubjectTransformConfig struct {
	// Source is the source subject pattern.
	Source string

	// Destination is the destination subject pattern.
	Destination string
}

// ConsumerCreateRequest is the create DTO for a JetStream consumer.
type ConsumerCreateRequest struct {
	// Name is the consumer name (required for durable consumers).
	Name string `normalize:"trim"`

	// Description is an optional consumer description.
	Description string `normalize:"trim"`

	// DeliverPolicy determines where to start delivering messages.
	DeliverPolicy DeliverPolicy

	// OptStartSeq is the starting sequence for by_start_sequence policy.
	OptStartSeq uint64

	// OptStartTime is the starting time for by_start_time policy (RFC3339).
	OptStartTime string

	// AckPolicy is the acknowledgment policy.
	AckPolicy AckPolicy

	// AckWait is the acknowledgment wait time.
	AckWait time.Duration

	// MaxDeliver is the maximum delivery attempts (-1 for unlimited).
	MaxDeliver int

	// BackOff is the backoff intervals for retries.
	BackOff []time.Duration

	// FilterSubject filters messages by subject pattern.
	FilterSubject string

	// FilterSubjects filters messages by multiple subject patterns (NATS 2.10+).
	FilterSubjects []string

	// ReplayPolicy is the replay policy.
	ReplayPolicy ReplayPolicy

	// RateLimit is the rate limit in bits per second.
	RateLimit uint64

	// SampleFrequency is the sampling frequency for observability (0-100%).
	SampleFrequency string

	// MaxAckPending is the maximum pending acknowledgments.
	MaxAckPending int

	// MaxWaiting is the maximum pending pull requests.
	MaxWaiting int

	// HeadersOnly delivers only headers without message bodies.
	HeadersOnly bool

	// MaxRequestBatch is the maximum batch size for pull requests.
	MaxRequestBatch int

	// MaxRequestMaxBytes is the maximum bytes for pull requests.
	MaxRequestMaxBytes int64

	// MaxRequestExpires is the maximum pull request expiration.
	MaxRequestExpires time.Duration

	// InactiveThreshold is the time before an inactive ephemeral consumer is
	// removed.
	InactiveThreshold time.Duration

	// Replicas is the number of consumer replicas.
	Replicas int

	// MemoryStorage uses memory storage instead of file.
	MemoryStorage bool

	// Metadata is custom key-value metadata.
	Metadata map[string]string

	// Ephemeral creates a non-durable consumer removed after inactivity.
	Ephemeral bool

	// --- Push Consumer Specific ---

	// DeliverSubject is the subject for push delivery.
	DeliverSubject string

	// DeliverGroup is the queue group for load balancing.
	DeliverGroup string

	// FlowControl enables flow control for push consumers.
	FlowControl bool

	// IdleHeartbeat is the idle heartbeat interval.
	IdleHeartbeat time.Duration
}

// ConsumerUpdateRequest holds mutable consumer fields; pointers distinguish
// "not sent" (nil) from "set to zero value".
type ConsumerUpdateRequest struct {
	// Description is the consumer description (mutable).
	Description *string `normalize:"trim,nil_on_empty"`

	// AckWait is the acknowledgment wait time (mutable).
	AckWait *time.Duration

	// MaxDeliver is the maximum delivery attempts (mutable).
	MaxDeliver *int

	// MaxAckPending is the maximum pending acknowledgments (mutable).
	MaxAckPending *int

	// MaxWaiting is the maximum pending pull requests (mutable).
	MaxWaiting *int

	// RateLimit is the rate limit in bits per second (mutable).
	RateLimit *uint64

	// SampleFrequency is the sampling frequency (mutable).
	SampleFrequency *string

	// InactiveThreshold is the inactive threshold (mutable).
	InactiveThreshold *time.Duration

	// BackOff intervals (mutable).
	BackOff []time.Duration

	// MaxRequestBatch is the maximum batch size (mutable).
	MaxRequestBatch *int

	// MaxRequestMaxBytes is the maximum bytes (mutable).
	MaxRequestMaxBytes *int64

	// MaxRequestExpires is the maximum expiration (mutable).
	MaxRequestExpires *time.Duration

	// Metadata is custom key-value metadata (mutable).
	Metadata map[string]string

	// FilterSubject is the subject filter (mutable, NATS 2.10+).
	FilterSubject *string

	// FilterSubjects are the subject filters (mutable, NATS 2.10+).
	FilterSubjects []string
}

// ConsumerPauseResponse is the outcome of a consumer pause request.
type ConsumerPauseResponse struct {
	// Paused indicates if the consumer is currently paused.
	Paused bool

	// PauseUntil is when the pause will end.
	PauseUntil *time.Time

	// PauseRemaining is the remaining pause duration.
	PauseRemaining time.Duration
}

// KVBucketConfig is the create/update config for a KeyValue bucket.
type KVBucketConfig struct {
	// Bucket is the bucket name (required).
	Bucket string `normalize:"trim"`

	// Description is an optional bucket description.
	Description string `normalize:"trim"`

	// MaxValueSize is the maximum value size in bytes.
	MaxValueSize int32

	// MaxBytes is the maximum total bucket size in bytes.
	MaxBytes int64

	// History is the number of historical values to keep per key (1-64).
	History uint8

	// TTL is the time-to-live for keys.
	TTL time.Duration

	// Storage is the storage type.
	Storage StorageType

	// Replicas is the number of replicas.
	Replicas int

	// Placement specifies cluster placement preferences.
	Placement *Placement

	// Metadata is custom key-value metadata.
	Metadata map[string]string

	// Mirror configures this bucket as a mirror of another bucket (NATS 2.11+).
	Mirror *StreamSource

	// Sources configures this bucket to aggregate from other buckets (NATS 2.11+).
	Sources []*StreamSource

	// Republish configures re-publishing of bucket operations to another subject.
	Republish *StreamRePublish
}

// KVBucketInfo is the current state of a KeyValue bucket.
type KVBucketInfo struct {
	// Bucket is the bucket name.
	Bucket string

	// Description is the bucket description.
	Description string

	// Values is the number of keys in the bucket.
	Values uint64

	// Bytes is the total size in bytes.
	Bytes uint64

	// History is the number of historical values kept per key.
	History uint8

	// TTL is the time-to-live for keys.
	TTL time.Duration

	// Storage is the storage type.
	Storage StorageType

	// Replicas is the number of replicas.
	Replicas int

	// IsCompressed indicates if the bucket uses compression.
	IsCompressed bool

	// Metadata is custom key-value metadata.
	Metadata map[string]string
}

// KVEntry is a single key-value pair from a KV bucket.
type KVEntry struct {
	// Bucket is the bucket name.
	Bucket string

	// Key is the key name.
	Key string

	// Value is the value (base64 encoded for binary data).
	Value string

	// Revision is the entry revision number.
	Revision uint64

	// Created is when the entry was created.
	Created time.Time

	// Operation is the operation type: put, delete, or purge.
	Operation string
}

// ObjectBucketConfig is the create/update config for an Object Store bucket.
type ObjectBucketConfig struct {
	// Bucket is the bucket name (required).
	Bucket string `normalize:"trim"`

	// Description is an optional bucket description.
	Description string `normalize:"trim"`

	// MaxBytes is the maximum total bucket size in bytes.
	MaxBytes int64

	// TTL is the time-to-live for objects.
	TTL time.Duration

	// Storage is the storage type.
	Storage StorageType

	// Replicas is the number of replicas.
	Replicas int

	// Placement specifies cluster placement preferences.
	Placement *Placement

	// Metadata is custom key-value metadata.
	Metadata map[string]string
}

// ObjectBucketInfo is the current state of an Object Store bucket.
type ObjectBucketInfo struct {
	// Bucket is the bucket name.
	Bucket string

	// Description is the bucket description.
	Description string

	// Size is the total size in bytes.
	Size uint64

	// Objects is the number of objects in the bucket.
	Objects uint64

	// Storage is the storage type.
	Storage StorageType

	// Replicas is the number of replicas.
	Replicas int

	// Sealed indicates if the bucket is sealed (read-only).
	Sealed bool

	// IsCompressed indicates if the bucket uses compression.
	IsCompressed bool

	// Metadata is custom key-value metadata.
	Metadata map[string]string
}

// ObjectInfo is metadata and state for one object in an Object Store.
type ObjectInfo struct {
	// Name is the object name.
	Name string

	// Description is the object description.
	Description string

	// Size is the object size in bytes.
	Size uint64

	// Chunks is the number of chunks.
	Chunks uint32

	// ModTime is the last modification time.
	ModTime time.Time

	// Digest is the SHA-256 digest of the object.
	Digest string

	// NUID is the unique identifier.
	NUID string

	// Deleted indicates if the object is deleted.
	Deleted bool

	// Headers are custom headers.
	Headers map[string][]string

	// Metadata is custom key-value metadata.
	Metadata map[string]string
}

// ObjectMeta is the client-supplied metadata for an object upload.
type ObjectMeta struct {
	// Name is the object name (required).
	Name string `normalize:"trim"`

	// Description is an optional object description.
	Description string `normalize:"trim"`

	// Headers are custom headers.
	Headers map[string][]string

	// Metadata is custom key-value metadata.
	Metadata map[string]string
}
