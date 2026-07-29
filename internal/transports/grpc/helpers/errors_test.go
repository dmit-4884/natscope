// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package grpchelpers

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/bbstore"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// assertGRPCError verifies err is a gRPC status with the expected code and optional ErrorInfo Reason.
func assertGRPCError(t *testing.T, err error, wantCode codes.Code, wantReason string) {
	t.Helper()
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok, "error should be a gRPC status error")
	assert.Equal(t, wantCode, st.Code(), "unexpected gRPC status code")

	if wantReason != "" {
		details := st.Details()
		require.NotEmpty(t, details, "expected error details with reason")
		info, ok := details[0].(*errdetails.ErrorInfo)
		require.True(t, ok, "detail should be ErrorInfo")
		assert.Equal(t, wantReason, info.Reason)
	}
}

// TestStatusErrorConvert covers the cross-cutting common mapping; raw NATS SDK sentinels are tested in services/nats (already wrapped into domain sentinels by the time they reach here).
func TestStatusErrorConvert(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	tests := []struct {
		name       string
		err        error
		wantCode   codes.Code
		wantReason string
	}{
		// Common domain
		{name: "ErrInvalidRequest", err: errs.ErrInvalidRequest, wantCode: codes.InvalidArgument, wantReason: "INVALID_REQUEST"},
		{name: "ErrNotFound", err: errs.ErrNotFound, wantCode: codes.NotFound, wantReason: "NOT_FOUND"},
		{name: "ErrDuplicateEntity", err: errs.ErrDuplicateEntity, wantCode: codes.AlreadyExists, wantReason: "DUPLICATE_ENTITY"},
		{name: "ErrInvalidListCursor", err: errs.ErrInvalidListCursor, wantCode: codes.InvalidArgument, wantReason: "INVALID_LIST_CURSOR"},
		{name: "BBStoreInvalidCursor", err: bbstore.ErrInvalidCursor, wantCode: codes.InvalidArgument, wantReason: "INVALID_LIST_CURSOR"},
		{name: "WrappedBBStoreCursor", err: fmt.Errorf("decode cursor: %w", bbstore.ErrInvalidCursor), wantCode: codes.InvalidArgument, wantReason: "INVALID_LIST_CURSOR"},
		{name: "ErrUnauthorized", err: errs.ErrUnauthorized, wantCode: codes.Unauthenticated, wantReason: "UNAUTHORIZED"},
		{name: "ErrPermissionDenied", err: errs.ErrPermissionDenied, wantCode: codes.PermissionDenied, wantReason: "PERMISSION_DENIED"},
		{name: "WrappedCommonError", err: fmt.Errorf("wrap: %w", errs.ErrNotFound), wantCode: codes.NotFound, wantReason: "NOT_FOUND"},

		// NATS connection domain (service-layer wraps SDK errs into these)
		{name: "ErrNATSConnectionClosed", err: errs.ErrNATSConnectionClosed, wantCode: codes.Unavailable, wantReason: "NATS_CONNECTION_CLOSED"},
		{name: "ErrNATSConnectionFailed", err: errs.ErrNATSConnectionFailed, wantCode: codes.Unavailable, wantReason: "NATS_CONNECTION_FAILED"},
		{name: "ErrNATSTimeout", err: errs.ErrNATSTimeout, wantCode: codes.DeadlineExceeded, wantReason: "NATS_TIMEOUT"},
		{name: "ErrNATSPermissionViolation", err: errs.ErrNATSPermissionViolation, wantCode: codes.PermissionDenied, wantReason: "NATS_PERMISSION_VIOLATION"},

		// JetStream entities (service-layer wraps SDK errs into these)
		{name: "ErrStreamNotFound", err: errs.ErrStreamNotFound, wantCode: codes.NotFound, wantReason: "NATS_STREAM_NOT_FOUND"},
		{name: "ErrStreamNameInUse", err: errs.ErrStreamNameInUse, wantCode: codes.AlreadyExists, wantReason: "NATS_STREAM_NAME_IN_USE"},
		{name: "ErrConsumerNotFound", err: errs.ErrConsumerNotFound, wantCode: codes.NotFound, wantReason: "NATS_CONSUMER_NOT_FOUND"},
		{name: "ErrJetStreamNotEnabled", err: errs.ErrJetStreamNotEnabled, wantCode: codes.FailedPrecondition, wantReason: "NATS_JETSTREAM_NOT_ENABLED"},
		{name: "ErrBucketNotFound", err: errs.ErrBucketNotFound, wantCode: codes.NotFound, wantReason: "NATS_BUCKET_NOT_FOUND"},
		{name: "ErrBucketExists", err: errs.ErrBucketExists, wantCode: codes.AlreadyExists, wantReason: "NATS_BUCKET_EXISTS"},
		{name: "ErrKeyNotFound", err: errs.ErrKeyNotFound, wantCode: codes.NotFound, wantReason: "NATS_KEY_NOT_FOUND"},
		{name: "ErrNoKeysFound", err: errs.ErrNoKeysFound, wantCode: codes.NotFound, wantReason: "NATS_NO_KEYS"},
		{name: "ErrObjectNotFound", err: errs.ErrObjectNotFound, wantCode: codes.NotFound, wantReason: "NATS_OBJECT_NOT_FOUND"},
		{name: "ErrNoObjectsFound", err: errs.ErrNoObjectsFound, wantCode: codes.NotFound, wantReason: "NATS_NO_OBJECTS"},
		{name: "ErrObjectAlreadyExists", err: errs.ErrObjectAlreadyExists, wantCode: codes.AlreadyExists, wantReason: "NATS_OBJECT_EXISTS"},
		{name: "ErrMsgNotFound", err: errs.ErrMsgNotFound, wantCode: codes.NotFound, wantReason: "NATS_MSG_NOT_FOUND"},
		{name: "ErrMsgDeleteDenied", err: errs.ErrMsgDeleteDenied, wantCode: codes.FailedPrecondition, wantReason: "NATS_MSG_DELETE_DENIED"},
		{name: "WrappedErrMsgDeleteDenied", err: fmt.Errorf("delete: %w", errs.ErrMsgDeleteDenied), wantCode: codes.FailedPrecondition, wantReason: "NATS_MSG_DELETE_DENIED"},
		{name: "ErrWorkQueueConsumerNotAllowed", err: errs.ErrWorkQueueConsumerNotAllowed, wantCode: codes.FailedPrecondition, wantReason: "NATS_WORKQUEUE_CONSUMER_NOT_ALLOWED"},
		{name: "ErrLiveNoSubscriptions", err: errs.ErrLiveNoSubscriptions, wantCode: codes.FailedPrecondition, wantReason: "LIVE_NO_SUBSCRIPTIONS"},

		// Proto codec/registry shared
		{name: "ErrNoProtoSources", err: errs.ErrNoProtoSources, wantCode: codes.FailedPrecondition, wantReason: "NO_PROTO_SOURCES"},
		{name: "ErrProtoMessageNotFound", err: errs.ErrProtoMessageNotFound, wantCode: codes.NotFound, wantReason: "PROTO_MESSAGE_NOT_FOUND"},
		{name: "ErrSchemaConflict", err: errs.ErrSchemaConflict, wantCode: codes.FailedPrecondition, wantReason: "SCHEMA_CONFLICT"},

		// Workspace import/export shared
		{name: "ErrWorkspaceInvalidFile", err: errs.ErrWorkspaceInvalidFile, wantCode: codes.InvalidArgument, wantReason: "WORKSPACE_INVALID_FILE"},
		{name: "WrappedErrWorkspaceInvalidFile", err: fmt.Errorf("import: %w", errs.ErrWorkspaceInvalidFile), wantCode: codes.InvalidArgument, wantReason: "WORKSPACE_INVALID_FILE"},
		{name: "ErrWorkspaceSectionInvalid", err: errs.ErrWorkspaceSectionInvalid, wantCode: codes.InvalidArgument, wantReason: "WORKSPACE_SECTION_INVALID"},
		{name: "WrappedErrWorkspaceSectionInvalid", err: fmt.Errorf("section %q: %w", "mappings", errs.ErrWorkspaceSectionInvalid), wantCode: codes.InvalidArgument, wantReason: "WORKSPACE_SECTION_INVALID"},
		{name: "ErrWorkspaceUnknownSection", err: errs.ErrWorkspaceUnknownSection, wantCode: codes.InvalidArgument, wantReason: "WORKSPACE_UNKNOWN_SECTION"},
		{name: "WrappedErrWorkspaceUnknownSection", err: fmt.Errorf("export: %w", errs.ErrWorkspaceUnknownSection), wantCode: codes.InvalidArgument, wantReason: "WORKSPACE_UNKNOWN_SECTION"},

		// Structured JetStream API error (domain type, not SDK)
		{name: "NATSAPIError_Conflict", err: &errs.NATSAPIError{Code: 409, ErrorCode: 10074, Description: "stream replication factor invalid"}, wantCode: codes.AlreadyExists, wantReason: "NATS_API_ERROR"},
		{name: "NATSAPIError_BadRequest", err: &errs.NATSAPIError{Code: 400, ErrorCode: 10052, Description: "config validation failed"}, wantCode: codes.InvalidArgument, wantReason: "NATS_API_ERROR"},

		// Context
		{name: "ContextDeadlineExceeded", err: context.DeadlineExceeded, wantCode: codes.DeadlineExceeded, wantReason: "DEADLINE_EXCEEDED"},
		{name: "ContextCanceled", err: context.Canceled, wantCode: codes.Canceled, wantReason: "CANCELED"},

		// Unknown errors are replaced with a static internal status so internal
		// detail never reaches clients.
		{name: "UnknownError", err: errors.New("something unknown"), wantCode: codes.Internal, wantReason: "INTERNAL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := StatusErrorConvert(ctx, tt.err)
			assertGRPCError(t, result, tt.wantCode, tt.wantReason)
		})
	}
}

func TestStatusErrorConvert_Nil(t *testing.T) {
	t.Parallel()
	assert.NoError(t, StatusErrorConvert(t.Context(), nil))
}

// TestNATSAPIErrorMetadata verifies err_code/http_code survive in the ErrorInfo metadata.
func TestNATSAPIErrorMetadata(t *testing.T) {
	t.Parallel()

	apiErr := &errs.NATSAPIError{Code: 412, ErrorCode: 10026, Description: "maximum consumers exceeded"}
	result := StatusErrorConvert(t.Context(), apiErr)

	st, ok := status.FromError(result)
	require.True(t, ok)
	assert.Equal(t, codes.FailedPrecondition, st.Code())
	assert.Equal(t, "maximum consumers exceeded", st.Message())

	require.NotEmpty(t, st.Details())
	info, ok := st.Details()[0].(*errdetails.ErrorInfo)
	require.True(t, ok)
	assert.Equal(t, "NATS_API_ERROR", info.Reason)
	assert.Equal(t, "nats.jetstream", info.Domain)
	assert.Equal(t, "10026", info.Metadata["err_code"])
	assert.Equal(t, "412", info.Metadata["http_code"])
}
