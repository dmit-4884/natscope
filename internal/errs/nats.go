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
	// (nats.ErrAuthorization/ErrPermissionViolation or async violation message).
	ErrNATSPermissionViolation = errors.New("nats: permissions violation")
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
)

// ErrWorkQueueConsumerNotAllowed ephemeral AckNone consumers auto-ack on
// delivery, draining a WorkQueue stream; only direct GetMsg is safe there.
var ErrWorkQueueConsumerNotAllowed = errors.New("nats: cannot create read consumer on WorkQueue retention stream (would consume messages)")

// ErrLiveNoSubscriptions a Live request resolved to zero subscriptions across
// all configured streams.
var ErrLiveNoSubscriptions = errors.New("nats: failed to create any live subscriptions")

// ErrNATSInvalidArgument is a client-side rejection of a NATS request.
var ErrNATSInvalidArgument = errors.New("nats: invalid argument")

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
