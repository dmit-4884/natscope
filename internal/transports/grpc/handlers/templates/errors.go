// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package templates

import (
	"context"
	"errors"

	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/transports/grpc/helpers"

	"google.golang.org/grpc/codes"
)

// StatusErrorConvert maps template-domain errors to gRPC status; unrecognized
// ones fall through to grpchelpers.StatusErrorConvert.
func (h *Handler) StatusErrorConvert(ctx context.Context, err error) error {
	if errors.Is(err, errs.ErrMessageTemplateNotFound) {
		return grpchelpers.NewStatus(codes.NotFound, "message template not found", "MESSAGE_TEMPLATE_NOT_FOUND")
	}
	return grpchelpers.StatusErrorConvert(ctx, err)
}
