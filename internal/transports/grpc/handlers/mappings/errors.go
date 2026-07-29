// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package mappings

import (
	"context"
	"errors"

	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/transports/grpc/helpers"

	"google.golang.org/grpc/codes"
)

// StatusErrorConvert maps mapping-domain errors to gRPC status; unrecognized
// errors fall through to grpchelpers.StatusErrorConvert.
func (h *Handler) StatusErrorConvert(ctx context.Context, err error) error {
	switch {
	case errors.Is(err, errs.ErrMappingNotFound):
		return grpchelpers.NewStatus(codes.NotFound, "mapping not found", "MAPPING_NOT_FOUND")
	case errors.Is(err, errs.ErrMappingPatternAlreadyInUse):
		return grpchelpers.NewStatus(codes.AlreadyExists, "pattern already mapped", "MAPPING_PATTERN_ALREADY_IN_USE")
	case errors.Is(err, errs.ErrMappingSourceIDRequired):
		return grpchelpers.NewStatus(codes.InvalidArgument, "source_id is required", "MAPPING_SOURCE_ID_REQUIRED")
	case errors.Is(err, errs.ErrMappingSourceNotFound):
		return grpchelpers.NewStatus(codes.FailedPrecondition, "mapping source not found", "MAPPING_SOURCE_NOT_FOUND")
	case errors.Is(err, errs.ErrMappingSourceDisabled):
		return grpchelpers.NewStatus(codes.FailedPrecondition, "mapping source disabled", "MAPPING_SOURCE_DISABLED")
	case errors.Is(err, errs.ErrMappingSelectionMissing):
		return grpchelpers.NewStatus(codes.FailedPrecondition, "mapping selection missing", "MAPPING_SELECTION_MISSING")
	case errors.Is(err, errs.ErrMappingDescriptorMissing):
		return grpchelpers.NewStatus(codes.FailedPrecondition, "mapping descriptor missing", "MAPPING_DESCRIPTOR_MISSING")
	case errors.Is(err, errs.ErrMessageTypeNotInSource):
		return grpchelpers.NewStatus(codes.FailedPrecondition, "message type not in source", "MESSAGE_TYPE_NOT_IN_SOURCE")
	}
	return grpchelpers.StatusErrorConvert(ctx, err)
}
