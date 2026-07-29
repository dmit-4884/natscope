// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package mappings

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/errs"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestStatusErrorConvert(t *testing.T) {
	t.Parallel()
	h := &Handler{}
	ctx := t.Context()

	tests := []struct {
		name       string
		err        error
		wantCode   codes.Code
		wantReason string
	}{
		{"MappingNotFound", errs.ErrMappingNotFound, codes.NotFound, "MAPPING_NOT_FOUND"},
		{"MappingPatternAlreadyInUse", errs.ErrMappingPatternAlreadyInUse, codes.AlreadyExists, "MAPPING_PATTERN_ALREADY_IN_USE"},
		{"MappingSourceIDRequired", errs.ErrMappingSourceIDRequired, codes.InvalidArgument, "MAPPING_SOURCE_ID_REQUIRED"},
		{"MappingSourceNotFound", errs.ErrMappingSourceNotFound, codes.FailedPrecondition, "MAPPING_SOURCE_NOT_FOUND"},
		{"MappingSourceDisabled", errs.ErrMappingSourceDisabled, codes.FailedPrecondition, "MAPPING_SOURCE_DISABLED"},
		{"MappingSelectionMissing", errs.ErrMappingSelectionMissing, codes.FailedPrecondition, "MAPPING_SELECTION_MISSING"},
		{"MappingDescriptorMissing", errs.ErrMappingDescriptorMissing, codes.FailedPrecondition, "MAPPING_DESCRIPTOR_MISSING"},
		{"MessageTypeNotInSource", errs.ErrMessageTypeNotInSource, codes.FailedPrecondition, "MESSAGE_TYPE_NOT_IN_SOURCE"},
		// Fallback
		{"ProtoMessageNotFound_viaFallback", errs.ErrProtoMessageNotFound, codes.NotFound, "PROTO_MESSAGE_NOT_FOUND"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := h.StatusErrorConvert(ctx, tt.err)
			require.Error(t, result)
			st, ok := status.FromError(result)
			require.True(t, ok)
			assert.Equal(t, tt.wantCode, st.Code())
			require.NotEmpty(t, st.Details())
			info, ok := st.Details()[0].(*errdetails.ErrorInfo)
			require.True(t, ok)
			assert.Equal(t, tt.wantReason, info.Reason)
		})
	}
}
