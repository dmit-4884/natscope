// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package grpchelpers translates application errors (sentinels in errs,
// SDK-free) into gRPC/Connect status; per-handler converters delegate here for
// cross-cutting errors.
package grpchelpers

import (
	"context"
	"errors"
	"log/slog"
	"strconv"

	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/bbstore"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	coreerrs "github.com/altessa-s/go-atlas/core/errors"
	slogx "github.com/altessa-s/go-atlas/observability/slog"
)

// errorMapping holds the gRPC status fields for one logical error case.
type errorMapping struct {
	code   codes.Code
	msg    string
	reason string
}

// commonDomainErrors maps shared domain sentinels (2+ handlers) to gRPC status;
// the cross-cutting fallback for per-handler converters.
var commonDomainErrors = []struct {
	src error
	out errorMapping
}{
	// Cross-cutting (every list/CRUD endpoint).
	{errs.ErrInvalidRequest, errorMapping{codes.InvalidArgument, "invalid request", "INVALID_REQUEST"}},
	{errs.ErrNotFound, errorMapping{codes.NotFound, "not found", "NOT_FOUND"}},
	{errs.ErrDuplicateEntity, errorMapping{codes.AlreadyExists, "entity already exists", "DUPLICATE_ENTITY"}},
	{errs.ErrInvalidListCursor, errorMapping{codes.InvalidArgument, "invalid list cursor", "INVALID_LIST_CURSOR"}},
	{bbstore.ErrInvalidCursor, errorMapping{codes.InvalidArgument, "invalid list cursor", "INVALID_LIST_CURSOR"}},
	{errs.ErrUnauthorized, errorMapping{codes.Unauthenticated, "unauthorized", "UNAUTHORIZED"}},
	{errs.ErrPermissionDenied, errorMapping{codes.PermissionDenied, "permission denied", "PERMISSION_DENIED"}},

	// NATS — common (not copied into 8+ converters); SDK sentinels translated to
	// these in services/nats so the transport stays SDK-agnostic.
	{errs.ErrNATSConnectionClosed, errorMapping{codes.Unavailable, "nats connection closed", "NATS_CONNECTION_CLOSED"}},
	{errs.ErrNATSConnectionFailed, errorMapping{codes.Unavailable, "nats server unavailable", "NATS_CONNECTION_FAILED"}},
	{errs.ErrNATSTimeout, errorMapping{codes.DeadlineExceeded, "nats operation timed out", "NATS_TIMEOUT"}},
	{errs.ErrNATSPermissionViolation, errorMapping{codes.PermissionDenied, "nats permissions violation", "NATS_PERMISSION_VIOLATION"}},
	{errs.ErrNATSInvalidArgument, errorMapping{codes.InvalidArgument, "nats: invalid argument", "NATS_INVALID_ARGUMENT"}},
	{errs.ErrStreamNotFound, errorMapping{codes.NotFound, "stream not found", "NATS_STREAM_NOT_FOUND"}},
	{errs.ErrStreamNameInUse, errorMapping{codes.AlreadyExists, "stream name already in use", "NATS_STREAM_NAME_IN_USE"}},
	{errs.ErrConsumerNotFound, errorMapping{codes.NotFound, "consumer not found", "NATS_CONSUMER_NOT_FOUND"}},
	{errs.ErrJetStreamNotEnabled, errorMapping{codes.FailedPrecondition, "jetstream not enabled", "NATS_JETSTREAM_NOT_ENABLED"}},
	{errs.ErrBucketNotFound, errorMapping{codes.NotFound, "bucket not found", "NATS_BUCKET_NOT_FOUND"}},
	{errs.ErrBucketExists, errorMapping{codes.AlreadyExists, "bucket already exists", "NATS_BUCKET_EXISTS"}},
	{errs.ErrKeyNotFound, errorMapping{codes.NotFound, "key not found", "NATS_KEY_NOT_FOUND"}},
	{errs.ErrNoKeysFound, errorMapping{codes.NotFound, "no keys found", "NATS_NO_KEYS"}},
	{errs.ErrObjectNotFound, errorMapping{codes.NotFound, "object not found", "NATS_OBJECT_NOT_FOUND"}},
	{errs.ErrNoObjectsFound, errorMapping{codes.NotFound, "no objects found", "NATS_NO_OBJECTS"}},
	{errs.ErrObjectAlreadyExists, errorMapping{codes.AlreadyExists, "object already exists", "NATS_OBJECT_EXISTS"}},
	{errs.ErrMsgNotFound, errorMapping{codes.NotFound, "message not found", "NATS_MSG_NOT_FOUND"}},
	{errs.ErrMsgDeleteDenied, errorMapping{codes.FailedPrecondition, "message deletion disabled on stream", "NATS_MSG_DELETE_DENIED"}},
	{errs.ErrWorkQueueConsumerNotAllowed, errorMapping{
		codes.FailedPrecondition,
		"cannot create read consumer on WorkQueue stream",
		"NATS_WORKQUEUE_CONSUMER_NOT_ALLOWED",
	}},
	{errs.ErrLiveNoSubscriptions, errorMapping{codes.FailedPrecondition, "no live subscriptions could be created", "LIVE_NO_SUBSCRIPTIONS"}},

	// Proto codec/registry — shared by codec, registry, publish, live.
	{errs.ErrNoProtoSources, errorMapping{codes.FailedPrecondition, "no proto sources configured", "NO_PROTO_SOURCES"}},
	{errs.ErrProtoMessageNotFound, errorMapping{codes.NotFound, "proto message not found", "PROTO_MESSAGE_NOT_FOUND"}},
	{errs.ErrSchemaConflict, errorMapping{codes.FailedPrecondition, "schema conflict", "SCHEMA_CONFLICT"}},

	// Workspace import/export.
	{errs.ErrWorkspaceInvalidFile, errorMapping{codes.InvalidArgument, "invalid workspace file", "WORKSPACE_INVALID_FILE"}},
	{errs.ErrWorkspaceSectionInvalid, errorMapping{codes.InvalidArgument, "invalid workspace section payload", "WORKSPACE_SECTION_INVALID"}},
	{errs.ErrWorkspaceUnknownSection, errorMapping{codes.InvalidArgument, "unknown workspace section key", "WORKSPACE_UNKNOWN_SECTION"}},
}

// StatusErrorConvert maps cross-cutting errors (common+NATS domain sentinels,
// structured NATS API errors, context cancel/deadline) to gRPC status.
func StatusErrorConvert(_ context.Context, err error) error {
	if err == nil {
		return nil
	}

	if valErr, ok := errors.AsType[*errs.NATSValidationError](err); ok && valErr != nil {
		return NewStatus(codes.InvalidArgument, valErr.Error(), "NATS_INVALID_ARGUMENT")
	}

	// Per-handler errors are matched upstream; here we handle only the common set.
	for _, m := range commonDomainErrors {
		if errors.Is(err, m.src) {
			return statusFromMapping(m.out)
		}
	}

	// Structured JetStream API error: services/nats wraps it into
	// *errs.NATSAPIError to keep this SDK-agnostic.
	if apiErr, ok := errors.AsType[*errs.NATSAPIError](err); ok && apiErr != nil {
		return statusFromAPIError(apiErr)
	}

	// Context cancel/deadline: matched here for a stable reason code.
	if coreerrs.IsContextDeadlineExceeded(err) {
		return statusFromMapping(errorMapping{codes.DeadlineExceeded, "deadline exceeded", "DEADLINE_EXCEEDED"})
	}
	if coreerrs.IsContextCanceled(err) {
		return statusFromMapping(errorMapping{codes.Canceled, "canceled", "CANCELED"})
	}

	// Unmapped error: the full chain may carry internal detail (operation
	// chains, file paths), so log it server-side and return a static message.
	slog.Default().With(slogx.Module("transport:grpc")).Error("unmapped internal error", slogx.Error(err))

	return statusFromMapping(errorMapping{codes.Internal, "internal error", "INTERNAL"})
}

// statusFromAPIError keeps the server Description as the message and exposes
// err_code via ErrorInfo metadata for the UI.
func statusFromAPIError(api *errs.NATSAPIError) error {
	msg := api.Description
	if msg == "" {
		msg = "nats jetstream api error"
	}

	st := status.New(httpToGRPCCode(api.Code), msg)
	info := &errdetails.ErrorInfo{
		Reason: "NATS_API_ERROR",
		Domain: "nats.jetstream",
		Metadata: map[string]string{
			"err_code":  strconv.FormatUint(uint64(api.ErrorCode), 10),
			"http_code": strconv.Itoa(api.Code),
		},
	}
	if updated, withErr := st.WithDetails(info); withErr == nil {
		return updated.Err()
	}
	return st.Err()
}

// HTTP-style status codes used by NATS JetStream APIError.Code (RFC 9110).
const (
	httpStatusBadRequest          = 400
	httpStatusUnauthorized        = 401
	httpStatusForbidden           = 403
	httpStatusNotFound            = 404
	httpStatusConflict            = 409
	httpStatusPreconditionFailed  = 412
	httpStatusTooManyRequests     = 429
	httpStatusInternalServerError = 500
	httpStatusServiceUnavailable  = 503
)

// httpToGRPCCode maps NATS-server "HTTP-like" status codes (APIError.Code) to
// gRPC codes.
func httpToGRPCCode(code int) codes.Code {
	switch {
	case code == httpStatusBadRequest:
		return codes.InvalidArgument
	case code == httpStatusUnauthorized:
		return codes.Unauthenticated
	case code == httpStatusForbidden:
		return codes.PermissionDenied
	case code == httpStatusNotFound:
		return codes.NotFound
	case code == httpStatusConflict:
		return codes.AlreadyExists
	case code == httpStatusPreconditionFailed:
		return codes.FailedPrecondition
	case code == httpStatusTooManyRequests:
		return codes.ResourceExhausted
	case code == httpStatusServiceUnavailable:
		return codes.Unavailable
	case code >= httpStatusInternalServerError:
		return codes.Internal
	default:
		return codes.Unknown
	}
}

// statusFromMapping builds a gRPC status with an ErrorInfo Reason from a
// mapping entry.
func statusFromMapping(m errorMapping) error {
	return NewStatus(m.code, m.msg, m.reason)
}

// NewStatus builds a gRPC status error carrying an ErrorInfo Reason; public so
// per-handler converters produce identically-shaped statuses.
func NewStatus(code codes.Code, msg, reason string) error {
	st := status.New(code, msg)
	if updated, err := st.WithDetails(&errdetails.ErrorInfo{Reason: reason}); err == nil {
		st = updated
	}
	return st.Err()
}
