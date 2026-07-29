// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package codec

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"

	protosvc "github.com/dmit-4884/natscope/internal/services/proto"
	codecpb "github.com/dmit-4884/natscope/proto/gen/services/grpc/proto/v1/codec"
)

// --- Mocks ---

// mockProtoSvc embeds proto.Codec; only codec methods are overridden.
type mockProtoSvc struct {
	protosvc.Codec
	decodeResult   *entities.DecodeResult
	decodeErr      error
	encodeResult   *entities.EncodeResult
	encodeErr      error
	validateResult *entities.ValidationResult
	validateErr    error
}

func (m *mockProtoSvc) Decode(_ context.Context, _ entities.CodecRequest) (*entities.DecodeResult, error) {
	return m.decodeResult, m.decodeErr
}

func (m *mockProtoSvc) Encode(_ context.Context, _ entities.CodecRequest) (*entities.EncodeResult, error) {
	return m.encodeResult, m.encodeErr
}

func (m *mockProtoSvc) Validate(
	_ context.Context, _ string, _ entities.CodecRequest,
) (*entities.ValidationResult, error) {
	return m.validateResult, m.validateErr
}

// --- Tests ---

func TestHandler_DecodeMessage(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoSvc{decodeResult: &entities.DecodeResult{
			Decoded:     []byte(`{"a":1}`),
			MessageType: "test.Msg",
		}}
		handler := New(svc)

		resp, err := handler.DecodeMessage(t.Context(), connect.NewRequest(&codecpb.DecodeMessageRequest{
			Data:        []byte{0x01},
			SourceId:    "src-1",
			MessageType: "test.Msg",
		}))
		require.NoError(t, err)
		require.NotNil(t, resp.Msg.Result)
		assert.Equal(t, "test.Msg", resp.Msg.Result.MessageType)
		assert.Equal(t, `{"a":1}`, resp.Msg.Result.Data)
	})

	t.Run("ServiceError", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoSvc{decodeErr: errs.ErrProtoMessageNotFound}
		handler := New(svc)

		_, err := handler.DecodeMessage(t.Context(), connect.NewRequest(&codecpb.DecodeMessageRequest{
			SourceId:    "src-1",
			MessageType: "missing.Msg",
		}))
		assert.ErrorIs(t, err, errs.ErrProtoMessageNotFound)
	})
}

func TestHandler_EncodeMessage(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		// "AQID" decodes to bytes {1,2,3}.
		svc := &mockProtoSvc{encodeResult: &entities.EncodeResult{DataBase64: "AQID", DataSize: 3}}
		handler := New(svc)

		resp, err := handler.EncodeMessage(t.Context(), connect.NewRequest(&codecpb.EncodeMessageRequest{
			Data:        `{"a":1}`,
			SourceId:    "src-1",
			MessageType: "test.Msg",
		}))
		require.NoError(t, err)
		require.NotNil(t, resp.Msg.Result)
		assert.Equal(t, int64(3), resp.Msg.Result.DataSize)
		assert.Equal(t, []byte{1, 2, 3}, resp.Msg.Result.Data)
	})

	t.Run("ServiceError", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoSvc{encodeErr: errs.ErrProtoMessageNotFound}
		handler := New(svc)

		_, err := handler.EncodeMessage(t.Context(), connect.NewRequest(&codecpb.EncodeMessageRequest{
			SourceId:    "src-1",
			MessageType: "missing.Msg",
		}))
		assert.ErrorIs(t, err, errs.ErrProtoMessageNotFound)
	})
}

func TestHandler_ValidateMessage(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoSvc{validateResult: &entities.ValidationResult{
			Valid:      false,
			Violations: []*entities.ValidationViolation{{FieldPath: "a", Message: "required"}},
		}}
		handler := New(svc)

		resp, err := handler.ValidateMessage(t.Context(), connect.NewRequest(&codecpb.ValidateMessageRequest{
			Data:        []byte{0x01},
			SourceId:    "src-1",
			MessageType: "test.Msg",
		}))
		require.NoError(t, err)
		require.NotNil(t, resp.Msg.Result)
		assert.False(t, resp.Msg.Result.Valid)
		assert.Len(t, resp.Msg.Result.Violations, 1)
	})

	t.Run("ServiceError", func(t *testing.T) {
		t.Parallel()
		svc := &mockProtoSvc{validateErr: errs.ErrProtoMessageNotFound}
		handler := New(svc)

		_, err := handler.ValidateMessage(t.Context(), connect.NewRequest(&codecpb.ValidateMessageRequest{
			SourceId:    "src-1",
			MessageType: "missing.Msg",
		}))
		assert.ErrorIs(t, err, errs.ErrProtoMessageNotFound)
	})
}
