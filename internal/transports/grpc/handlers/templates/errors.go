// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

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
	switch {
	case errors.Is(err, errs.ErrMessageTemplateNotFound):
		return grpchelpers.NewStatus(codes.NotFound, "message template not found", "MESSAGE_TEMPLATE_NOT_FOUND")
	case errors.Is(err, errs.ErrMessageTemplateNameRequired):
		return grpchelpers.NewStatus(codes.InvalidArgument, "template name is required", "MESSAGE_TEMPLATE_NAME_REQUIRED")
	}
	return grpchelpers.StatusErrorConvert(ctx, err)
}
