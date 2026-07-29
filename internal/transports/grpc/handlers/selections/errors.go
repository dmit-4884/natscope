// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package selections

import (
	"context"
	"errors"

	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/transports/grpc/helpers"

	"google.golang.org/grpc/codes"
)

// StatusErrorConvert maps proto-selection-domain errors to gRPC status;
// unrecognized errors fall through to grpchelpers.StatusErrorConvert.
func (h *Handler) StatusErrorConvert(ctx context.Context, err error) error {
	switch {
	case errors.Is(err, errs.ErrProtoSelectionNotFound):
		return grpchelpers.NewStatus(codes.NotFound, "proto selection not found", "PROTO_SELECTION_NOT_FOUND")
	case errors.Is(err, errs.ErrProtoDescriptorNotFound):
		return grpchelpers.NewStatus(codes.NotFound, "proto descriptor not found", "PROTO_DESCRIPTOR_NOT_FOUND")
	}
	return grpchelpers.StatusErrorConvert(ctx, err)
}
