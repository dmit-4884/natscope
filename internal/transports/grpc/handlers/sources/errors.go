// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

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
	case errors.Is(err, errs.ErrProtoVersionNotFound):
		return grpchelpers.NewStatus(codes.NotFound, "proto version not found", "PROTO_VERSION_NOT_FOUND")
	case errors.Is(err, errs.ErrProtoVersionAlreadyExists):
		return grpchelpers.NewStatus(codes.AlreadyExists, "proto version already exists", "PROTO_VERSION_ALREADY_EXISTS")
	}
	return grpchelpers.StatusErrorConvert(ctx, err)
}
