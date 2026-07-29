// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsclient

import (
	"context"
	"time"

	"github.com/dmit-4884/natscope/internal/entities"
)

// Error contract: every method of [Dialer] and [Client] returns errors already
// translated to internal/errs domain sentinels (matchable with errors.Is /
// errors.As) or *errs.NATSAPIError. No nats.go / jetstream SDK error type ever
// escapes an implementation, so callers never import the SDK to classify a
// failure. The translation happens exactly once, at the implementation
// boundary.

// Dialer establishes NATS connections. Implementations are the vendor adapters
// (see the natsgo subpackage); a Dialer is safe for concurrent use.
type Dialer interface {
	// Dial opens a live connection for the saved configuration and returns a
	// ready [Client]. The returned client owns the underlying connection until
	// its Close is called.
	Dial(ctx context.Context, saved *entities.SavedConnection) (Client, error)

	// TestConnection probes a connection without saving or pooling it, dialing
	// ad hoc and returning server info plus diagnostics. Probe failures are
	// reported in the result (Success=false, Error set), not as a Go error.
	TestConnection(ctx context.Context, in *entities.TestConnectionRequest) (*entities.TestConnectionResult, error)
}

// ConnectionInfo exposes the health and identity of a single live connection.
type ConnectionInfo interface {
	// URL returns the NATS server URL the client dialed.
	URL() string

	// IsConnected reports whether the underlying connection is currently up.
	IsConnected() bool

	// IsReconnecting reports whether the client is mid-reconnect.
	IsReconnecting() bool

	// Status returns the connection status as a string (connected, reconnecting,
	// closed, ...).
	Status() string

	// Health returns health (RTT + state) for the connection.
	Health(ctx context.Context) (*entities.ConnectionHealth, error)

	// RTT measures round-trip time to the server via a flush.
	RTT() (time.Duration, error)

	// Close closes the underlying connection; idempotent.
	Close()
}

// StreamReader provides read-only access to JetStream streams and their
// messages.
type StreamReader interface {
	// ListStreams returns all JetStream streams (parallel fetch).
	ListStreams(ctx context.Context) ([]entities.StreamInfo, error)

	// GetStreamInfo returns detail for one stream.
	GetStreamInfo(ctx context.Context, streamName string) (*entities.StreamInfo, error)

	// GetStreamConsumers returns all consumers for a stream (parallel fetch).
	GetStreamConsumers(ctx context.Context, streamName string) ([]entities.ConsumerInfo, error)

	// GetStreamSubjects returns a stream's subject patterns.
	GetStreamSubjects(ctx context.Context, streamName string) ([]string, error)

	// GetMessages fetches a page with subject filtering (NATS wildcards * and >).
	GetMessages(ctx context.Context, streamName string, opts entities.GetMessagesOptions) (*entities.MessagesResponse, error)

	// GetMessage fetches one message by sequence number.
	GetMessage(ctx context.Context, streamName string, sequence uint64) (*entities.Message, error)
}

// StreamManager creates, mutates, and deletes JetStream streams and their
// messages.
type StreamManager interface {
	// CreateStream creates a JetStream stream.
	CreateStream(ctx context.Context, config entities.StreamCreateRequest) (*entities.StreamInfo, error)

	// UpdateStream updates mutable fields of a stream.
	UpdateStream(ctx context.Context, name string, config entities.StreamUpdateRequest) (*entities.StreamInfo, error)

	// DeleteStream deletes a stream and its data (irreversible).
	DeleteStream(ctx context.Context, name string) error

	// PurgeStream removes messages by subject or keep-count, returning the count
	// purged.
	PurgeStream(ctx context.Context, name string, req entities.StreamPurgeRequest) (uint64, error)

	// SealStream makes a stream read-only (irreversible); returns updated info.
	SealStream(ctx context.Context, name string) (*entities.StreamInfo, error)

	// DeleteMessage deletes a message by sequence (secure overwrites data first);
	// ErrMsgNotFound if missing, precondition error if deletes denied.
	DeleteMessage(ctx context.Context, streamName string, sequence uint64, secure bool) error
}

// ConsumerManager creates, mutates, and deletes JetStream consumers.
type ConsumerManager interface {
	// CreateConsumer creates a consumer on a stream.
	CreateConsumer(ctx context.Context, streamName string, config entities.ConsumerCreateRequest) (*entities.ConsumerInfo, error)

	// UpdateConsumer updates mutable fields of a consumer.
	UpdateConsumer(ctx context.Context, streamName, consumerName string, config entities.ConsumerUpdateRequest) (*entities.ConsumerInfo, error)

	// DeleteConsumer deletes a consumer.
	DeleteConsumer(ctx context.Context, streamName string, consumerName string) error

	// PauseConsumer pauses until the RFC3339 pauseUntil; non-RFC3339 yields
	// errs.ErrInvalidRequest.
	PauseConsumer(ctx context.Context, streamName string, consumerName string, pauseUntil string) (*entities.ConsumerPauseResponse, error)

	// ResumeConsumer resumes a paused consumer immediately.
	ResumeConsumer(ctx context.Context, streamName string, consumerName string) error
}

// Publisher publishes messages to JetStream streams.
type Publisher interface {
	// PublishToStream publishes to a JetStream stream, returning the ack.
	PublishToStream(ctx context.Context, subject string, data []byte, headers map[string]string) (*entities.PubAck, error)
}

// Subscriber creates live subscriptions over core NATS and JetStream.
type Subscriber interface {
	// Subscribe creates a Core NATS subscription for live streaming.
	Subscribe(ctx context.Context, subject string, handler entities.MessageHandler) (entities.Subscription, error)

	// SubscribeJetStream creates a JetStream ordered-consumer subscription for
	// live messages.
	SubscribeJetStream(
		ctx context.Context,
		streamName, subject, deliverPolicy string,
		handler entities.MessageHandler,
	) (entities.Subscription, error)
}

// StatsReader reports aggregated stream/consumer statistics and server info.
type StatsReader interface {
	// GetAllStreamsStats returns stats for all streams.
	GetAllStreamsStats(ctx context.Context) ([]entities.StreamStats, error)

	// GetStreamStats returns detailed stats for one stream (subject breakdown +
	// cluster).
	GetStreamStats(ctx context.Context, streamName string) (*entities.StreamStats, error)

	// GetAllConsumers returns all consumers across all streams.
	GetAllConsumers(ctx context.Context) ([]entities.ConsumerStats, error)

	// GetServerInfo returns server detail incl. cluster and JetStream account
	// stats.
	GetServerInfo(ctx context.Context) (*entities.ServerInfo, error)
}

// KVStore manages JetStream KeyValue buckets and keys.
type KVStore interface {
	// ListKVBuckets returns all KeyValue buckets.
	ListKVBuckets(ctx context.Context) ([]entities.KVBucketInfo, error)

	// CreateKVBucket creates a KeyValue bucket.
	CreateKVBucket(ctx context.Context, config entities.KVBucketConfig) (*entities.KVBucketInfo, error)

	// DeleteKVBucket deletes a KeyValue bucket and its data.
	DeleteKVBucket(ctx context.Context, bucket string) error

	// GetKVBucket returns info for one KeyValue bucket.
	GetKVBucket(ctx context.Context, bucket string) (*entities.KVBucketInfo, error)

	// ListKVKeys returns all keys in a bucket.
	ListKVKeys(ctx context.Context, bucket string) ([]string, error)

	// GetKVKey returns value + metadata for a key.
	GetKVKey(ctx context.Context, bucket string, key string) (*entities.KVEntry, error)

	// GetKVKeyHistory returns the stored revisions of a key, oldest first,
	// including delete/purge markers.
	GetKVKeyHistory(ctx context.Context, bucket string, key string) ([]entities.KVEntry, error)

	// PutKVKey stores a value, returning its revision. A non-zero
	// expectedRevision performs a compare-and-swap update.
	PutKVKey(ctx context.Context, bucket string, key string, value []byte, expectedRevision uint64) (uint64, error)

	// DeleteKVKey deletes a key.
	DeleteKVKey(ctx context.Context, bucket string, key string) error

	// PurgeKVKey purges all revisions of a key.
	PurgeKVKey(ctx context.Context, bucket string, key string) error
}

// ObjectStore manages JetStream Object Store buckets and objects.
type ObjectStore interface {
	// ListObjectBuckets returns all Object Store buckets.
	ListObjectBuckets(ctx context.Context) ([]entities.ObjectBucketInfo, error)

	// CreateObjectBucket creates an Object Store bucket.
	CreateObjectBucket(ctx context.Context, config entities.ObjectBucketConfig) (*entities.ObjectBucketInfo, error)

	// DeleteObjectBucket deletes a bucket and its objects.
	DeleteObjectBucket(ctx context.Context, bucket string) error

	// GetObjectBucket returns info for one Object Store bucket.
	GetObjectBucket(ctx context.Context, bucket string) (*entities.ObjectBucketInfo, error)

	// ListObjects returns all objects in a bucket.
	ListObjects(ctx context.Context, bucket string) ([]*entities.ObjectInfo, error)

	// GetObject returns content + metadata of an object.
	GetObject(ctx context.Context, bucket string, name string) ([]byte, *entities.ObjectInfo, error)

	// PutObject stores an object.
	PutObject(ctx context.Context, bucket string, meta entities.ObjectMeta, data []byte) (*entities.ObjectInfo, error)

	// DeleteObject deletes an object.
	DeleteObject(ctx context.Context, bucket string, name string) error

	// SealObjectBucket makes a bucket read-only (irreversible).
	SealObjectBucket(ctx context.Context, bucket string) error
}

// Client is a domain-typed handle to a single live NATS / JetStream connection.
// It aggregates the segregated operation roles plus connection lifecycle; every
// method returns already-translated internal/errs errors (see the package-level
// error contract). Implementations are safe for concurrent use.
type Client interface {
	ConnectionInfo
	StreamReader
	StreamManager
	ConsumerManager
	Publisher
	Subscriber
	StatsReader
	KVStore
	ObjectStore
}
