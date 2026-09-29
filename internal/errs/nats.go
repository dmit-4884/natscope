// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package errs

import (
	"errors"
	"fmt"
)

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
	ErrNATSPermissionViolation = errors.New("nats: permissions violation")

	// ErrNATSAuthorizationViolation server rejected the connection credentials.
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
	// ErrStreamPurgeDenied a purge on a deny_purge or sealed stream.
	ErrStreamPurgeDenied = errors.New("nats: stream purge is disabled on this stream (deny_purge)")
)

// ErrWorkQueueConsumerNotAllowed ephemeral AckNone consumers auto-ack on
// delivery, draining a WorkQueue stream; only direct GetMsg is safe there.
var ErrWorkQueueConsumerNotAllowed = errors.New("nats: cannot create read consumer on WorkQueue retention stream (would consume messages)")

// ErrLiveNoSubscriptions a Live request resolved to zero subscriptions across
// all configured streams.
var ErrLiveNoSubscriptions = errors.New("nats: failed to create any live subscriptions")

// ErrLiveConsumerStalled a live session's client stopped reading, so the session was closed.
var ErrLiveConsumerStalled = errors.New("nats: live consumer stopped reading, closing session")

// ErrLiveConnectionLost the NATS connection behind a live session was closed or replaced.
var ErrLiveConnectionLost = errors.New("nats: live connection was closed or replaced")

// ErrNATSNoResponders a core NATS request found no subscriber on its subject;
// the server reports it at once instead of letting the request time out.
var ErrNATSNoResponders = errors.New("nats: no responders available for request")

// ErrNATSInvalidArgument is a client-side rejection of a NATS request.
var ErrNATSInvalidArgument = errors.New("nats: invalid argument")

// ErrNotAKVOrObjectBucket the stream behind a bucket operation is not a KV or Object store.
var ErrNotAKVOrObjectBucket = errors.New("nats: stream is not a valid KV/Object bucket")

// ErrObjectBucketCapacityExceeded a PutObject would exceed the bucket's max_bytes.
var ErrObjectBucketCapacityExceeded = errors.New("nats: object write would exceed the bucket's max_bytes")

// ErrObjectTooLargeToRetrieve an object exceeds the GetObject size cap.
var ErrObjectTooLargeToRetrieve = errors.New("nats: object exceeds the maximum size retrievable via GetObject")

// ErrObjectLinkToBucket GetObject was called on a link to a whole bucket.
var ErrObjectLinkToBucket = errors.New("nats: object is a link to a bucket, not a single object")

// ErrFeatureUnsupported the connected server's JetStream API level is below
// the minimum a requested feature needs; see [FeatureUnsupportedError].
var ErrFeatureUnsupported = errors.New("nats: feature not supported by the connected server")

// FeatureUnsupportedError names the rejected feature, the NATS release that
// introduced it and the connected server version.
type FeatureUnsupportedError struct {
	Feature       string
	MinVersion    string
	ServerVersion string
}

// Error implements the error interface.
func (e *FeatureUnsupportedError) Error() string {
	server := "connected server version unknown"
	if e.ServerVersion != "" {
		server = "connected server v" + e.ServerVersion
	}
	return fmt.Sprintf("%s requires NATS %s+ (%s)", e.Feature, e.MinVersion, server)
}

// Is reports whether target is ErrFeatureUnsupported.
func (e *FeatureUnsupportedError) Is(target error) bool {
	return target == ErrFeatureUnsupported
}

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
