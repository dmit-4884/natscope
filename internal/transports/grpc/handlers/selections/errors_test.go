// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package selections

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
		{"ProtoSelectionNotFound", errs.ErrProtoSelectionNotFound, codes.NotFound, "PROTO_SELECTION_NOT_FOUND"},
		{"ProtoDescriptorNotFound", errs.ErrProtoDescriptorNotFound, codes.NotFound, "PROTO_DESCRIPTOR_NOT_FOUND"},
		// Fallback
		{"NotFound_viaFallback", errs.ErrNotFound, codes.NotFound, "NOT_FOUND"},
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
