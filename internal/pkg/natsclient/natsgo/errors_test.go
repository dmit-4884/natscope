// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/errs"
)

// TestWrapErr_SentinelMapping confirms each SDK sentinel is wrapped into the
// matching domain sentinel from errs/; transport only sees errors.Is matches.
func TestWrapErr_SentinelMapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		src  error
		want error
	}{
		// Connection
		{"ConnectionClosed", nats.ErrConnectionClosed, errs.ErrNATSConnectionClosed},
		{"ConnectionDraining", nats.ErrConnectionDraining, errs.ErrNATSConnectionClosed},
		{"Disconnected", nats.ErrDisconnected, errs.ErrNATSConnectionClosed},
		{"Timeout", nats.ErrTimeout, errs.ErrNATSTimeout},
		{"Authorization", nats.ErrAuthorization, errs.ErrNATSPermissionViolation},
		{"PermissionViolation", nats.ErrPermissionViolation, errs.ErrNATSPermissionViolation},

		// JetStream
		{"StreamNotFound", jetstream.ErrStreamNotFound, errs.ErrStreamNotFound},
		{"StreamNameInUse", jetstream.ErrStreamNameAlreadyInUse, errs.ErrStreamNameInUse},
		{"ConsumerNotFound", jetstream.ErrConsumerNotFound, errs.ErrConsumerNotFound},
		{"JetStreamNotEnabled", jetstream.ErrJetStreamNotEnabled, errs.ErrJetStreamNotEnabled},
		{"JetStreamNotEnabledForAccount", jetstream.ErrJetStreamNotEnabledForAccount, errs.ErrJetStreamNotEnabled},
		{"BucketNotFound", jetstream.ErrBucketNotFound, errs.ErrBucketNotFound},
		{"BucketExists", jetstream.ErrBucketExists, errs.ErrBucketExists},
		{"KeyNotFound", jetstream.ErrKeyNotFound, errs.ErrKeyNotFound},
		{"KeyDeleted", jetstream.ErrKeyDeleted, errs.ErrKeyNotFound},
		{"NoKeysFound", jetstream.ErrNoKeysFound, errs.ErrNoKeysFound},
		{"ObjectNotFound", jetstream.ErrObjectNotFound, errs.ErrObjectNotFound},
		{"NoObjectsFound", jetstream.ErrNoObjectsFound, errs.ErrNoObjectsFound},
		{"ObjectAlreadyExists", jetstream.ErrObjectAlreadyExists, errs.ErrObjectAlreadyExists},
		{"MsgNotFound", jetstream.ErrMsgNotFound, errs.ErrMsgNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := wrapErr(tt.src)
			require.Error(t, got)
			assert.ErrorIs(t, got, tt.want, "wrapped err should match domain sentinel")
			// Original SDK error stays in the chain for logs / debugging.
			assert.ErrorIs(t, got, tt.src, "original SDK err should remain in chain")
		})
	}
}

// TestWrapErr_WrappedSentinel ensures a sentinel wrapped via fmt.Errorf still
// gets mapped — services often add context with %w.
func TestWrapErr_WrappedSentinel(t *testing.T) {
	t.Parallel()
	got := wrapErr(fmt.Errorf("get stream: %w", jetstream.ErrStreamNotFound))
	assert.ErrorIs(t, got, errs.ErrStreamNotFound)
}

// TestWrapErr_ContextDeadlineExceeded asserts the ctx-deadline path produces
// the NATS timeout sentinel: batched ops use ctx as deadline, not cancellation.
func TestWrapErr_ContextDeadlineExceeded(t *testing.T) {
	t.Parallel()
	assert.ErrorIs(t, wrapErr(context.DeadlineExceeded), errs.ErrNATSTimeout)
}

// TestWrapErr_PermissionViolationString catches the async-error case: the
// server pushes a plain string with no SDK sentinel (nats.ErrorHandler path).
func TestWrapErr_PermissionViolationString(t *testing.T) {
	t.Parallel()
	got := wrapErr(errors.New("Permissions Violation for Publish to $JS.API.CONSUMER.CREATE"))
	assert.ErrorIs(t, got, errs.ErrNATSPermissionViolation)
}

// TestWrapErr_Idempotent ensures wrapping an already-wrapped error doesn't
// stack, since services may wrap at multiple call sites.
func TestWrapErr_Idempotent(t *testing.T) {
	t.Parallel()
	once := wrapErr(jetstream.ErrStreamNotFound)
	twice := wrapErr(once)
	assert.Equal(t, once, twice, "wrapping a domain sentinel again is a no-op")
}

// TestWrapErr_Unknown asserts unrecognised errors pass through. Transport's
// outer fallback turns these into codes.Internal.
func TestWrapErr_Unknown(t *testing.T) {
	t.Parallel()
	orig := errors.New("random non-NATS error")
	assert.Equal(t, orig, wrapErr(orig))
}

// TestWrapErr_Nil verifies the noop short-circuit.
func TestWrapErr_Nil(t *testing.T) {
	t.Parallel()
	assert.NoError(t, wrapErr(nil))
}
