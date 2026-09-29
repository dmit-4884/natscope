// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package publish

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"

	"google.golang.org/protobuf/types/known/durationpb"

	protosvc "github.com/dmit-4884/natscope/internal/services/proto"
	publishpb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/publish"
)

// --- Mocks ---

// mockPublishService is the full publish service.
type mockPublishService struct {
	result *entities.PublishResult
	err    error

	called bool
	got    *entities.PublishRequest

	reply      *entities.Reply
	gotRequest *entities.RequestMessage
}

func (m *mockPublishService) Publish(_ context.Context, in *entities.PublishRequest) (*entities.PublishResult, error) {
	m.called = true
	m.got = in
	return m.result, m.err
}

func (m *mockPublishService) Request(_ context.Context, in *entities.RequestMessage) (*entities.Reply, error) {
	m.called = true
	m.gotRequest = in
	return m.reply, m.err
}

// stubProtoService embeds protosvc.Codec; only Encode/ValidateJSON are used here.
type stubProtoService struct {
	protosvc.Codec

	encodeResult   *entities.EncodeResult
	encodeErr      error
	validateResult *entities.ValidationResult
	validateErr    error
}

func (s *stubProtoService) Encode(_ context.Context, _ entities.CodecRequest) (*entities.EncodeResult, error) {
	return s.encodeResult, s.encodeErr
}

func (s *stubProtoService) ValidateJSON(_ context.Context, _ entities.CodecRequest) (*entities.ValidationResult, error) {
	return s.validateResult, s.validateErr
}

// --- Tests ---

func TestHandler_PublishMessage(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockPublishService{result: &entities.PublishResult{Stream: "orders", Sequence: 42}}
		handler := New(svc, &stubProtoService{})

		resp, err := handler.PublishMessage(t.Context(), connect.NewRequest(&publishpb.PublishMessageRequest{
			ConnectionId: "conn-1",
			Subject:      "orders.new",
			Data:         "{}",
		}))
		require.NoError(t, err)
		assert.Equal(t, "orders", resp.Msg.Stream)
		assert.Equal(t, uint64(42), resp.Msg.Sequence)
		assert.True(t, svc.called)
	})

	t.Run("ServiceError", func(t *testing.T) {
		t.Parallel()
		svc := &mockPublishService{err: errors.New("connection down")}
		handler := New(svc, &stubProtoService{})

		_, err := handler.PublishMessage(t.Context(), connect.NewRequest(&publishpb.PublishMessageRequest{
			ConnectionId: "conn-1",
			Subject:      "orders.new",
		}))
		assert.Error(t, err)
	})
}

func TestHandler_RequestMessage(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockPublishService{reply: &entities.Reply{
			Subject:  "_INBOX.abc.1",
			Data:     []byte(`{"ok":true}`),
			Headers:  map[string]string{"Trace": "t-1"},
			Duration: 1500 * time.Microsecond,
		}}
		handler := New(svc, &stubProtoService{})

		msgType, sourceID := "api.v1.Ping", "src-1"
		resp, err := handler.RequestMessage(t.Context(), connect.NewRequest(&publishpb.RequestMessageRequest{
			ConnectionId: "conn-1",
			Subject:      "svc.ping",
			Data:         `{"n":1}`,
			Headers:      map[string]string{"K": "v"},
			MessageType:  &msgType,
			SourceId:     &sourceID,
			Timeout:      durationpb.New(2 * time.Second),
		}))
		require.NoError(t, err)

		got := svc.gotRequest
		require.NotNil(t, got)
		assert.Equal(t, "conn-1", got.ConnectionID)
		assert.Equal(t, "svc.ping", got.Subject)
		assert.JSONEq(t, `{"n":1}`, got.Data)
		assert.Equal(t, map[string]string{"K": "v"}, got.Headers)
		require.NotNil(t, got.MessageType)
		assert.Equal(t, "api.v1.Ping", *got.MessageType)
		require.NotNil(t, got.SourceID)
		assert.Equal(t, "src-1", *got.SourceID)
		assert.Equal(t, 2*time.Second, got.Timeout)

		assert.Equal(t, "_INBOX.abc.1", resp.Msg.GetSubject())
		assert.Equal(t, []byte(`{"ok":true}`), resp.Msg.GetData())
		assert.Equal(t, map[string]string{"Trace": "t-1"}, resp.Msg.GetHeaders())
		assert.Equal(t, 1500*time.Microsecond, resp.Msg.GetDuration().AsDuration())
	})

	t.Run("NoTimeout", func(t *testing.T) {
		t.Parallel()
		svc := &mockPublishService{reply: &entities.Reply{Subject: "_INBOX.abc.2"}}
		handler := New(svc, &stubProtoService{})

		_, err := handler.RequestMessage(t.Context(), connect.NewRequest(&publishpb.RequestMessageRequest{
			ConnectionId: "conn-1",
			Subject:      "svc.ping",
		}))
		require.NoError(t, err)
		require.NotNil(t, svc.gotRequest)
		assert.Zero(t, svc.gotRequest.Timeout)
		assert.Nil(t, svc.gotRequest.MessageType)
	})

	t.Run("ServiceError", func(t *testing.T) {
		t.Parallel()
		svc := &mockPublishService{err: errs.ErrNATSNoResponders}
		handler := New(svc, &stubProtoService{})

		_, err := handler.RequestMessage(t.Context(), connect.NewRequest(&publishpb.RequestMessageRequest{
			ConnectionId: "conn-1",
			Subject:      "svc.ping",
		}))
		assert.ErrorIs(t, err, errs.ErrNATSNoResponders)
	})
}

func TestHandler_EncodeMessage(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		proto := &stubProtoService{encodeResult: &entities.EncodeResult{
			Success:    true,
			DataBase64: base64.StdEncoding.EncodeToString([]byte("hello")),
			DataSize:   5,
		}}
		handler := New(&mockPublishService{}, proto)

		resp, err := handler.EncodeMessage(t.Context(), connect.NewRequest(&publishpb.EncodeMessageRequest{
			SourceId:    "src-1",
			MessageType: "api.v1.Order",
			Data:        `{"id":"1"}`,
		}))
		require.NoError(t, err)
		assert.Equal(t, int64(5), resp.Msg.DataSize)
		assert.Equal(t, []byte("hello"), resp.Msg.Data)
	})

	t.Run("ServiceError", func(t *testing.T) {
		t.Parallel()
		proto := &stubProtoService{encodeErr: errs.ErrProtoMessageNotFound}
		handler := New(&mockPublishService{}, proto)

		_, err := handler.EncodeMessage(t.Context(), connect.NewRequest(&publishpb.EncodeMessageRequest{
			SourceId:    "src-1",
			MessageType: "api.v1.Unknown",
			Data:        `{}`,
		}))
		assert.ErrorIs(t, err, errs.ErrProtoMessageNotFound)
	})
}

func TestHandler_ValidateJson(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		proto := &stubProtoService{validateResult: &entities.ValidationResult{Valid: true}}
		handler := New(&mockPublishService{}, proto)

		resp, err := handler.ValidateJson(t.Context(), connect.NewRequest(&publishpb.ValidateJsonRequest{
			SourceId:    "src-1",
			MessageType: "api.v1.Order",
			Data:        `{"id":"1"}`,
		}))
		require.NoError(t, err)
		require.NotNil(t, resp.Msg.Result)
		assert.True(t, resp.Msg.Result.Valid)
	})

	t.Run("ServiceError", func(t *testing.T) {
		t.Parallel()
		proto := &stubProtoService{validateErr: errs.ErrNoProtoSources}
		handler := New(&mockPublishService{}, proto)

		_, err := handler.ValidateJson(t.Context(), connect.NewRequest(&publishpb.ValidateJsonRequest{
			SourceId:    "src-1",
			MessageType: "api.v1.Order",
			Data:        `{}`,
		}))
		assert.ErrorIs(t, err, errs.ErrNoProtoSources)
	})
}
