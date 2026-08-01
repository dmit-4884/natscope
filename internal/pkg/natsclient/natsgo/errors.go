// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"context"
	"errors"
	"strings"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/dmit-4884/natscope/internal/errs"
)

// permissionViolationSubstr matches NATS async permission errors that arrive as
// plain strings without a sentinel.
const permissionViolationSubstr = "permissions violation"

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
	{nats.ErrAuthorization, errs.ErrNATSPermissionViolation},
	{nats.ErrPermissionViolation, errs.ErrNATSPermissionViolation},

	// JetStream entities
	{jetstream.ErrStreamNotFound, errs.ErrStreamNotFound},
	{jetstream.ErrStreamNameAlreadyInUse, errs.ErrStreamNameInUse},
	{jetstream.ErrConsumerNotFound, errs.ErrConsumerNotFound},
	{jetstream.ErrJetStreamNotEnabled, errs.ErrJetStreamNotEnabled},
	{jetstream.ErrJetStreamNotEnabledForAccount, errs.ErrJetStreamNotEnabled},
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
