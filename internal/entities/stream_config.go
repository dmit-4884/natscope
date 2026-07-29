// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

import "time"

// StreamConfig is the full configuration set for a JetStream stream.
type StreamConfig struct {
	// Name is the stream name.
	Name string

	// Description is the optional stream description.
	Description string

	// Subjects is the list of subjects the stream is bound to.
	Subjects []string

	// Retention is the message retention policy.
	Retention RetentionPolicy

	// MaxMsgs is the maximum number of messages in the stream.
	MaxMsgs int64

	// MaxBytes is the maximum total size of messages in bytes.
	MaxBytes int64

	// MaxAge is the maximum age of messages.
	MaxAge time.Duration

	// MaxConsumers is the maximum number of consumers.
	MaxConsumers int

	// MaxMsgsPerSubject is the maximum messages per subject.
	MaxMsgsPerSubject int64

	// MaxMsgSize is the maximum message size in bytes.
	MaxMsgSize int32

	// Storage is the storage type.
	Storage StorageType

	// Discard is the discard policy when limits are reached.
	Discard DiscardPolicy

	// Replicas is the number of replicas for the stream.
	Replicas int

	// Duplicates is the duplicate detection window.
	Duplicates time.Duration

	// Compression is the compression algorithm.
	Compression StoreCompression

	// Sealed indicates if the stream is sealed (read-only).
	Sealed bool

	// DenyDelete prevents message deletion.
	DenyDelete bool

	// DenyPurge prevents stream purging.
	DenyPurge bool

	// AllowRollup allows rollup headers.
	AllowRollup bool

	// AllowDirect enables direct get operations.
	AllowDirect bool

	// MirrorDirect enables direct get on mirrors.
	MirrorDirect bool

	// Metadata is custom key-value metadata.
	Metadata map[string]string

	// AllowMsgTTL allows per-message TTL.
	AllowMsgTTL bool

	// NoAck disables message acknowledgments.
	NoAck bool

	// DiscardNewPerSubject applies discard policy per subject.
	DiscardNewPerSubject bool

	// FirstSeq is the initial sequence number for the stream.
	FirstSeq uint64

	// AllowAtomicPublish enables atomic publish operations.
	AllowAtomicPublish bool

	// ConsumerLimits is the per-consumer limit set inherited from this stream.
	ConsumerLimits *StreamConsumerLimits

	// Mirror is the mirror configuration (this stream mirrors another).
	Mirror *StreamSourceRef

	// Sources is the list of source stream configurations (aggregation).
	Sources []*StreamSourceRef

	// Republish is the republish configuration.
	Republish *StreamRePublish

	// SubjectTransform is the subject transformation applied to messages on storage.
	SubjectTransform *SubjectTransformConfig
}

// StreamConsumerLimits is the default limit set inherited by stream consumers.
type StreamConsumerLimits struct {
	// InactiveThreshold is the inactive threshold.
	InactiveThreshold time.Duration

	// MaxAckPending is the maximum pending acknowledgments.
	MaxAckPending int
}

// StreamSourceRef is a reference to a source or mirror stream.
type StreamSourceRef struct {
	Name          string
	OptStartSeq   uint64
	FilterSubject string
	External      *ExternalStreamRef
}

// ExternalStreamRef is an external stream reference for cross-account access.
type ExternalStreamRef struct {
	ApiPrefix     string
	DeliverPrefix string
}

// StreamRePublish is the re-publish routing rule for a stream.
type StreamRePublish struct {
	Src         string
	Dest        string
	HeadersOnly bool
}
