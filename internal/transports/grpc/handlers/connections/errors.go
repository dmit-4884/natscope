// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package connections

import (
	"context"
	"errors"

	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/transports/grpc/helpers"

	"google.golang.org/grpc/codes"
)

// StatusErrorConvert maps connection-domain errors to gRPC status; unrecognized
// errors fall through to grpchelpers.StatusErrorConvert.
func (h *Handler) StatusErrorConvert(ctx context.Context, err error) error {
	switch {
	case errors.Is(err, errs.ErrSavedConnectionNotFound):
		return grpchelpers.NewStatus(codes.NotFound, "connection not found", "CONNECTION_NOT_FOUND")
	case errors.Is(err, errs.ErrConnectionNameAlreadyInUse):
		return grpchelpers.NewStatus(codes.AlreadyExists, "connection name already in use", "CONNECTION_NAME_ALREADY_IN_USE")
	case errors.Is(err, errs.ErrConnectionNameRequired):
		return grpchelpers.NewStatus(codes.InvalidArgument, "connection name is required", "CONNECTION_NAME_REQUIRED")
	case errors.Is(err, errs.ErrConnectionURLCredentialsMixed):
		return grpchelpers.NewStatus(codes.InvalidArgument,
			"server URLs embed different credentials", "CONNECTION_URL_CREDENTIALS_MIXED")
	case errors.Is(err, errs.ErrConnectionURLCredentialsConflict):
		return grpchelpers.NewStatus(codes.InvalidArgument,
			"credentials are set both in the server URL and in auth", "CONNECTION_URL_CREDENTIALS_CONFLICT")
	}
	return grpchelpers.StatusErrorConvert(ctx, err)
}
