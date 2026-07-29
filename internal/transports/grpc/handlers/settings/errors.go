// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package settings

import (
	"context"
	"errors"

	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/transports/grpc/helpers"

	"google.golang.org/grpc/codes"
)

// StatusErrorConvert maps settings-domain errors to gRPC status; unknown errors
// fall through.
func (h *Handler) StatusErrorConvert(ctx context.Context, err error) error {
	if errors.Is(err, errs.ErrSettingsNotFound) {
		return grpchelpers.NewStatus(codes.NotFound, "settings not found", "SETTINGS_NOT_FOUND")
	}
	return grpchelpers.StatusErrorConvert(ctx, err)
}
