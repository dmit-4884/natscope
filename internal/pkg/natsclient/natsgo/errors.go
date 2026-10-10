// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/dmit-4884/natscope/internal/errs"
)

// permissionViolationSubstr matches NATS async permission errors that arrive as
// plain strings without a sentinel.
const permissionViolationSubstr = "permissions violation"

// invalidResetDescription replaces the SDK's bare "invalid reset" for a
// consumer reset the server refuses.
const invalidResetDescription = "consumer reset refused: a consumer can only be reset to a sequence when its deliver policy is " +
	"all, by_start_sequence or by_start_time, and never to a sequence before its start sequence or start time"

// jsErrCodeMessageTTLDisabled is the server's "per-message TTL is disabled" error, which nats.go has no constant for.
const jsErrCodeMessageTTLDisabled jetstream.ErrorCode = 10166

// isAPIErrorCode reports whether err carries a JetStream API error with the given code.
func isAPIErrorCode(err error, code jetstream.ErrorCode) bool {
	jsErr, ok := errors.AsType[jetstream.JetStreamError](err)
	return ok && jsErr.APIError() != nil && jsErr.APIError().ErrorCode == code
}

// noAnswer reports a JetStream request on an existing stream that found no responder as a timeout.
func noAnswer(err error) error {
	if !errors.Is(err, nats.ErrNoResponders) {
		return err
	}
	return fmt.Errorf("%w: JetStream did not answer, try again: %w", errs.ErrNATSTimeout, err)
}

// natsSentinelMap maps NATS/JetStream SDK sentinels → domain errs sentinels,
// matched via errors.Is.
var natsSentinelMap = []struct {
	src, dst error
}{
	// Connection-level
	{nats.ErrConnectionClosed, errs.ErrNATSConnectionClosed},
	{nats.ErrConnectionDraining, errs.ErrNATSConnectionClosed},
	{nats.ErrDisconnected, errs.ErrNATSConnectionClosed},
	{nats.ErrTimeout, errs.ErrNATSTimeout},
	{nats.ErrAuthorization, errs.ErrNATSAuthorizationViolation},
	{nats.ErrAuthExpired, errs.ErrNATSAuthorizationViolation},
	{nats.ErrAuthRevoked, errs.ErrNATSAuthorizationViolation},
	{nats.ErrAccountAuthExpired, errs.ErrNATSAuthorizationViolation},
	{nats.ErrPermissionViolation, errs.ErrNATSPermissionViolation},

	// JetStream entities
	{jetstream.ErrStreamNotFound, errs.ErrStreamNotFound},
	{jetstream.ErrStreamNameAlreadyInUse, errs.ErrStreamNameInUse},
	{jetstream.ErrConsumerNotFound, errs.ErrConsumerNotFound},
	{jetstream.ErrJetStreamNotEnabled, errs.ErrJetStreamNotEnabled},
	{jetstream.ErrJetStreamNotEnabledForAccount, errs.ErrJetStreamNotEnabled},
	{nats.ErrNoResponders, errs.ErrJetStreamNotEnabled},
	{jetstream.ErrBucketNotFound, errs.ErrBucketNotFound},
	{jetstream.ErrBucketExists, errs.ErrBucketExists},
	{jetstream.ErrKeyNotFound, errs.ErrKeyNotFound},
	{jetstream.ErrKeyDeleted, errs.ErrKeyNotFound},
	{jetstream.ErrNoKeysFound, errs.ErrNoKeysFound},
	{jetstream.ErrObjectNotFound, errs.ErrObjectNotFound},
	{jetstream.ErrNoObjectsFound, errs.ErrNoObjectsFound},
	{jetstream.ErrObjectAlreadyExists, errs.ErrObjectAlreadyExists},
	{jetstream.ErrMsgNotFound, errs.ErrMsgNotFound},
}

var natsValidationSentinels = []error{
	jetstream.ErrInvalidStreamName,
	jetstream.ErrStreamNameRequired,
	jetstream.ErrInvalidConsumerName,
	jetstream.ErrInvalidSubject,
	jetstream.ErrInvalidBucketName,
	jetstream.ErrInvalidKey,
	jetstream.ErrInvalidStoreName,
	jetstream.ErrNameRequired,
	jetstream.ErrBucketRequired,
	jetstream.ErrKeyValueConfigRequired,
	jetstream.ErrObjectConfigRequired,
	jetstream.ErrHistoryTooLarge,
	nats.ErrBadSubject,
	nats.ErrMaxPayload,
}

// wrapErr is the single place translating NATS/JetStream SDK errors into
// domain errs sentinels; transport never sees SDK error types.
func wrapErr(err error) error {
	if err == nil {
		return nil
	}

	// Idempotency: already a domain sentinel means a deeper call wrapped it —
	// don't double-wrap.
	for _, m := range natsSentinelMap {
		if errors.Is(err, m.dst) {
			return err
		}
	}
	if valErr, ok := errors.AsType[*errs.NATSValidationError](err); ok && valErr != nil {
		return err
	}

	if v, ok := ParsePermissionViolation(err); ok {
		return v.asError(err)
	}

	for _, m := range natsSentinelMap {
		if errors.Is(err, m.src) {
			// Preserve original chain for debug logs; callers use errors.Is for the
			// sentinel.
			return errors.Join(m.dst, err)
		}
	}

	for _, src := range natsValidationSentinels {
		if errors.Is(err, src) {
			return &errs.NATSValidationError{Description: err.Error(), Cause: err}
		}
	}

	if errors.Is(err, jetstream.ErrConsumerInvalidReset) {
		return &errs.NATSValidationError{Description: invalidResetDescription, Cause: err}
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return errors.Join(errs.ErrNATSTimeout, err)
	}

	// JetStream APIError → *errs.NATSAPIError so transport can errors.As without
	// importing the SDK.
	if jsErr, ok := errors.AsType[jetstream.JetStreamError](err); ok {
		if api := jsErr.APIError(); api != nil {
			return &errs.NATSAPIError{
				Code:        api.Code,
				ErrorCode:   uint16(api.ErrorCode),
				Description: api.Description,
			}
		}
	}

	if strings.Contains(strings.ToLower(err.Error()), permissionViolationSubstr) {
		return errors.Join(errs.ErrNATSPermissionViolation, err)
	}

	return err
}
