// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package sources

import (
	"context"
	"errors"

	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/transports/grpc/helpers"

	"google.golang.org/grpc/codes"
)

// StatusErrorConvert maps proto-source-domain errors to gRPC status; unknown
// errors fall through.
func (h *Handler) StatusErrorConvert(ctx context.Context, err error) error {
	switch {
	case errors.Is(err, errs.ErrProtoSourceNotFound):
		return grpchelpers.NewStatus(codes.NotFound, "proto source not found", "PROTO_SOURCE_NOT_FOUND")
	case errors.Is(err, errs.ErrProtoSourceNameAlreadyInUse):
		return grpchelpers.NewStatus(codes.AlreadyExists, "proto source name already in use", "PROTO_SOURCE_NAME_ALREADY_IN_USE")
	case errors.Is(err, errs.ErrProtoRefNotFound):
		return grpchelpers.NewStatus(codes.NotFound, "ref not found", "PROTO_REF_NOT_FOUND")
	}
	if regErr, ok := errors.AsType[*errs.RegistryError](err); ok {
		switch regErr.Code {
		case "not_found":
			return grpchelpers.NewStatus(codes.NotFound, regErr.Error(), "BSR_NOT_FOUND")
		case "unauthenticated", "permission_denied":
			return grpchelpers.NewStatus(codes.PermissionDenied, regErr.Error(), "BSR_ACCESS_DENIED")
		default:
			return grpchelpers.NewStatus(codes.Unavailable, regErr.Error(), "BSR_UNAVAILABLE")
		}
	}
	return grpchelpers.StatusErrorConvert(ctx, err)
}
