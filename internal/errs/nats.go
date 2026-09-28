// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package errs

import "errors"

// Connection-level domain sentinels from the [services/nats] pool; SDK
// sentinels they shadow never escape that package.
var (
	// ErrNATSConnectionClosed connection closed when a call asked for it (pool
	// stopped, or SDK ConnectionClosed/Draining).
	ErrNATSConnectionClosed = errors.New("nats: connection closed")

	// ErrNATSConnectionFailed nats.Connect failed to dial the configured URLs (no
	// SDK sentinel covers this).
	ErrNATSConnectionFailed = errors.New("nats: failed to connect to server")

	// ErrNATSTimeout operation deadline elapsed; wraps nats.ErrTimeout and
	// context.DeadlineExceeded.
	ErrNATSTimeout = errors.New("nats: operation timed out")

	// ErrNATSPermissionViolation server rejected op
	// (nats.ErrPermissionViolation or async violation message).
	ErrNATSPermissionViolation = errors.New("nats: permissions violation")

	// ErrNATSAuthorizationViolation server rejected the connection credentials
	// (nats.ErrAuthorization, expired or revoked auth).
	ErrNATSAuthorizationViolation = errors.New("nats: authorization violation")
)

// JetStream domain errors wrapping jetstream.Err* at the [services/nats]
// boundary so transport never imports the SDK.
var (
	ErrStreamNotFound      = errors.New("nats: stream not found")
	ErrStreamNameInUse     = errors.New("nats: stream name already in use")
	ErrConsumerNotFound    = errors.New("nats: consumer not found")
	ErrJetStreamNotEnabled = errors.New("nats: jetstream not enabled")
	ErrBucketNotFound      = errors.New("nats: bucket not found")
	ErrBucketExists        = errors.New("nats: bucket already exists")
	ErrKeyNotFound         = errors.New("nats: key not found")
	ErrNoKeysFound         = errors.New("nats: no keys found")
	ErrObjectNotFound      = errors.New("nats: object not found")
	ErrNoObjectsFound      = errors.New("nats: no objects found")
	ErrObjectAlreadyExists = errors.New("nats: object already exists")
	ErrMsgNotFound         = errors.New("nats: message not found")
	// ErrMsgDeleteDenied single-message delete on a deny_delete stream; never
	// succeeds, maps to FailedPrecondition.
	ErrMsgDeleteDenied = errors.New("nats: message deletion is disabled on this stream (deny_delete)")
	// ErrStreamPurgeDenied a stream purge on a deny_purge/sealed stream; never
	// succeeds, maps to FailedPrecondition.
	ErrStreamPurgeDenied = errors.New("nats: stream purge is disabled on this stream (deny_purge)")
)

// ErrWorkQueueConsumerNotAllowed ephemeral AckNone consumers auto-ack on
// delivery, draining a WorkQueue stream; only direct GetMsg is safe there.
var ErrWorkQueueConsumerNotAllowed = errors.New("nats: cannot create read consumer on WorkQueue retention stream (would consume messages)")

// ErrLiveNoSubscriptions a Live request resolved to zero subscriptions across
// all configured streams.
var ErrLiveNoSubscriptions = errors.New("nats: failed to create any live subscriptions")

// ErrLiveConsumerStalled a live session's client stopped reading the stream
// (a Send blocked past the session's send deadline); the session is torn
// down instead of buffering unbounded data for it.
var ErrLiveConsumerStalled = errors.New("nats: live consumer stopped reading, closing session")

// ErrLiveConnectionLost the pooled NATS connection behind a live session was
// closed or replaced; the client should resubscribe.
var ErrLiveConnectionLost = errors.New("nats: live connection was closed or replaced")

// ErrNATSInvalidArgument is a client-side rejection of a NATS request.
var ErrNATSInvalidArgument = errors.New("nats: invalid argument")

// ErrNotAKVOrObjectBucket the stream backing a KV/Object bucket operation
// isn't actually a KV/Object store (wrong subject shape, or missing the
// per-subject history nats.go's own KV sanity check requires); produced by
// [natsgo] before a bucket-shaped RPC would otherwise delete or seal a plain
// stream that merely shares its name.
var ErrNotAKVOrObjectBucket = errors.New("nats: stream is not a valid KV/Object bucket")

// ErrObjectBucketCapacityExceeded a PutObject would exceed the bucket's
// max_bytes and is refused before any chunk is written.
var ErrObjectBucketCapacityExceeded = errors.New("nats: object write would exceed the bucket's max_bytes")

// ErrObjectTooLargeToRetrieve GetObject refuses to buffer an object larger
// than natscope's read cap.
var ErrObjectTooLargeToRetrieve = errors.New("nats: object exceeds the maximum size retrievable via GetObject")

// ErrObjectLinkToBucket GetObject was called on an entry that links to an
// entire bucket rather than a single object.
var ErrObjectLinkToBucket = errors.New("nats: object is a link to a bucket, not a single object")

// NATSValidationError is the domain form of a client-side NATS validation failure.
type NATSValidationError struct {
	Description string
	Cause       error
}

// Error implements the error interface.
func (e *NATSValidationError) Error() string {
	if e == nil || e.Description == "" {
		return ErrNATSInvalidArgument.Error()
	}
	return e.Description
}

// Is reports whether target is ErrNATSInvalidArgument.
func (e *NATSValidationError) Is(target error) bool {
	return target == ErrNATSInvalidArgument
}

// Unwrap returns the originating error.
func (e *NATSValidationError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// NATSAPIError is the [services/nats] domain form of a structured JetStream
// API error, kept so transport never imports the SDK.
type NATSAPIError struct {
	// Code is the NATS-server HTTP-style status code (RFC 9110 subset).
	Code int
	// ErrorCode is the numeric err_code emitted by the JetStream server.
	ErrorCode uint16
	// Description is the human-readable cause from the server.
	Description string
}

// Error implements the error interface.
func (e *NATSAPIError) Error() string {
	if e == nil || e.Description == "" {
		return "nats jetstream api error"
	}
	return e.Description
}
