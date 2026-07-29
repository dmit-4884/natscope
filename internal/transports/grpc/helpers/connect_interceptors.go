// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package grpchelpers

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"github.com/dmit-4884/natscope/internal/transports/grpc/reasoncodes"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	bufhelpers "github.com/altessa-s/go-atlas/transport/grpc/interceptors/protovalidator/buf"
)

// validator is the singleton buf-validate function; runs protovalidate.Validate
// and emits structured field-path violations. The resolver turns raw rule IDs
// into canonical, client-facing reason codes (go-atlas no longer does this
// itself). Concurrency-safe.
var validator = bufhelpers.BuildValidator(
	bufhelpers.BuildValidationFilter(),
	bufhelpers.WithResolver(reasoncodes.NewResolver(reasoncodes.Catalog, reasoncodes.Prefix)),
)

// NewValidationConnectInterceptor validates proto messages via buf-validate,
// covering streaming RPCs too since a plain interceptor only wraps unary calls.
func NewValidationConnectInterceptor() connect.Interceptor {
	return &validationInterceptor{}
}

type validationInterceptor struct{}

func (validationInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if msg, ok := req.Any().(proto.Message); ok {
			if err := validator(ctx, msg); err != nil {
				return nil, statusToConnectError(err)
			}
		}
		return next(ctx, req)
	}
}

func (validationInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (validationInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		return next(ctx, &validatingStreamingHandlerConn{StreamingHandlerConn: conn, ctx: ctx})
	}
}

// validatingStreamingHandlerConn validates the request as soon as it's
// received; streaming RPCs have no separate "request" value like WrapUnary does.
type validatingStreamingHandlerConn struct {
	connect.StreamingHandlerConn
	ctx context.Context //nolint:containedctx // conn methods take no context param to attach it to
}

func (c *validatingStreamingHandlerConn) Receive(msg any) error {
	if err := c.StreamingHandlerConn.Receive(msg); err != nil {
		return err
	}
	if pm, ok := msg.(proto.Message); ok {
		if err := validator(c.ctx, pm); err != nil {
			return statusToConnectError(err)
		}
	}
	return nil
}

// ErrorConverter converts a service-layer error into a gRPC status; each
// handler exposes one via StatusErrorConvert.
type ErrorConverter func(ctx context.Context, err error) error

// NewHandlerErrorInterceptor translates handler errors via the per-handler
// converter into *connect.Error (unary+streaming).
func NewHandlerErrorInterceptor(convert ErrorConverter) connect.Interceptor {
	if convert == nil {
		convert = StatusErrorConvert
	}
	return &handlerErrorMapper{convert: convert}
}

type handlerErrorMapper struct {
	convert ErrorConverter
}

func (m *handlerErrorMapper) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		resp, err := next(ctx, req)
		if err != nil {
			return resp, statusErrToConnectWith(ctx, err, m.convert)
		}
		return resp, nil
	}
}

func (m *handlerErrorMapper) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (m *handlerErrorMapper) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		if err := next(ctx, conn); err != nil {
			return statusErrToConnectWith(ctx, err, m.convert)
		}
		return nil
	}
}

// statusErrToConnectWith translates a service-layer error into *connect.Error;
// handler converter runs first, falling through to StatusErrorConvert.
func statusErrToConnectWith(ctx context.Context, err error, convert ErrorConverter) error {
	if err == nil {
		return nil
	}
	// Already a connect.Error: pass through.
	if ce, _ := errors.AsType[*connect.Error](err); ce != nil {
		return err
	}

	// Converters fall through to StatusErrorConvert for the common set (or it's
	// the default).
	return statusToConnectError(convert(ctx, err))
}

// statusToConnectError rebuilds a gRPC status error as *connect.Error
// preserving code, message, and details (ErrorInfo, BadRequest).
func statusToConnectError(err error) error {
	if err == nil {
		return nil
	}
	if ce, _ := errors.AsType[*connect.Error](err); ce != nil {
		return err
	}
	st, ok := status.FromError(err)
	if !ok {
		return connect.NewError(connect.CodeUnknown, err)
	}
	cerr := connect.NewError(connectCodeFromGRPC(st.Code()), errors.New(st.Message()))
	for _, raw := range st.Proto().GetDetails() {
		if d, dErr := connect.NewErrorDetail(raw); dErr == nil {
			cerr.AddDetail(d)
		}
	}
	return cerr
}

// connectCodeFromGRPC maps grpc codes.Code to connect.Code (no codes.OK —
// that's a nil error).
func connectCodeFromGRPC(c codes.Code) connect.Code {
	switch c {
	case codes.Canceled:
		return connect.CodeCanceled
	case codes.Unknown:
		return connect.CodeUnknown
	case codes.InvalidArgument:
		return connect.CodeInvalidArgument
	case codes.DeadlineExceeded:
		return connect.CodeDeadlineExceeded
	case codes.NotFound:
		return connect.CodeNotFound
	case codes.AlreadyExists:
		return connect.CodeAlreadyExists
	case codes.PermissionDenied:
		return connect.CodePermissionDenied
	case codes.ResourceExhausted:
		return connect.CodeResourceExhausted
	case codes.FailedPrecondition:
		return connect.CodeFailedPrecondition
	case codes.Aborted:
		return connect.CodeAborted
	case codes.OutOfRange:
		return connect.CodeOutOfRange
	case codes.Unimplemented:
		return connect.CodeUnimplemented
	case codes.Internal:
		return connect.CodeInternal
	case codes.Unavailable:
		return connect.CodeUnavailable
	case codes.DataLoss:
		return connect.CodeDataLoss
	case codes.Unauthenticated:
		return connect.CodeUnauthenticated
	default:
		return connect.CodeUnknown
	}
}
