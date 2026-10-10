// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package publish

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
)

func newRequestService(
	t *testing.T,
	proto *mockProtoService,
	requestFn func(ctx context.Context, connID, subject string, data []byte, headers map[string]string) (*entities.Reply, error),
) (*Service, *recordedHistory) {
	t.Helper()
	hist := &recordedHistory{}
	natsm := &mockNATSService{requestFn: requestFn}
	return New(natsm, natsm, natsm, proto, &mockHistoryService{rec: hist}, &mockSettingsService{}), hist
}

func TestRequest_RawPayload(t *testing.T) {
	t.Parallel()

	want := &entities.Reply{Subject: "_INBOX.x.1", Data: []byte("pong"), Duration: time.Millisecond}
	s, hist := newRequestService(t, &mockProtoService{},
		func(ctx context.Context, connID, subject string, data []byte, headers map[string]string) (*entities.Reply, error) {
			assert.Equal(t, "conn-1", connID)
			assert.Equal(t, "svc.ping", subject)
			assert.Equal(t, []byte("ping"), data, "a raw payload is sent verbatim")
			assert.Equal(t, map[string]string{"K": "v"}, headers)
			deadline, ok := ctx.Deadline()
			require.True(t, ok, "a request must be bounded by a deadline")
			assert.WithinDuration(t, time.Now().Add(defaultRequestTimeout), deadline, time.Second)
			return want, nil
		})

	got, err := s.Request(t.Context(), &entities.RequestMessage{
		ConnectionID: " conn-1 ",
		Subject:      "svc.ping",
		Data:         "ping",
		Headers:      map[string]string{"K": "v"}})
	require.NoError(t, err)
	assert.Equal(t, want, got)

	hist.mu.Lock()
	defer hist.mu.Unlock()
	assert.Zero(t, hist.called, "requests are not recorded to publish history")
}

func TestRequest_ProtoEncoded(t *testing.T) {
	t.Parallel()

	encoded := []byte{0x08, 0x01}
	proto := &mockProtoService{
		encodeRawFn: func(_ context.Context, req entities.CodecRequest) ([]byte, error) {
			assert.Equal(t, "api.v1.Ping", req.MessageType)
			assert.Equal(t, "src-1", req.SourceID)
			assert.JSONEq(t, `{"n":1}`, string(req.JSON))
			return encoded, nil
		},
	}
	s, _ := newRequestService(t, proto,
		func(_ context.Context, _, _ string, data []byte, _ map[string]string) (*entities.Reply, error) {
			assert.Equal(t, encoded, data, "the proto-encoded bytes are sent")
			return &entities.Reply{}, nil
		})

	msgType, sourceID := "api.v1.Ping", "src-1"
	_, err := s.Request(t.Context(), &entities.RequestMessage{
		ConnectionID: "conn-1",
		Subject:      "svc.ping",
		Data:         `{"n":1}`,
		MessageType:  &msgType,
		SourceID:     &sourceID})
	require.NoError(t, err)
}

func TestRequest_EncodeErrorIsTyped(t *testing.T) {
	t.Parallel()

	proto := &mockProtoService{
		encodeRawFn: func(context.Context, entities.CodecRequest) ([]byte, error) {
			return nil, errors.New("Cannot convert JSON to 'api.v1.Ping'")
		},
	}
	s, _ := newRequestService(t, proto,
		func(context.Context, string, string, []byte, map[string]string) (*entities.Reply, error) {
			t.Fatal("nats.Request must not be called on an encode error")
			return nil, nil
		})

	msgType, sourceID := "api.v1.Ping", "src-1"
	_, err := s.Request(t.Context(), &entities.RequestMessage{
		ConnectionID: "conn-1", Subject: "svc.ping", Data: "{}", MessageType: &msgType, SourceID: &sourceID})

	encErr, ok := errors.AsType[*errs.ProtoEncodeError](err)
	require.True(t, ok, "an encode failure must be a *errs.ProtoEncodeError, got %v", err)
	assert.Equal(t, "Cannot convert JSON to 'api.v1.Ping'", encErr.Description)
}

func TestRequest_CustomTimeout(t *testing.T) {
	t.Parallel()

	s, _ := newRequestService(t, &mockProtoService{},
		func(ctx context.Context, _, _ string, _ []byte, _ map[string]string) (*entities.Reply, error) {
			deadline, ok := ctx.Deadline()
			require.True(t, ok)
			assert.WithinDuration(t, time.Now().Add(250*time.Millisecond), deadline, 200*time.Millisecond)
			return &entities.Reply{}, nil
		})

	_, err := s.Request(t.Context(), &entities.RequestMessage{
		ConnectionID: "conn-1", Subject: "svc.ping",
		Timeout: 250 * time.Millisecond,
	})
	require.NoError(t, err)
}

func TestRequest_RejectsInvalidInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   entities.PublishRequest
	}{
		{name: "wildcard subject", in: entities.PublishRequest{ConnectionID: "conn-1", Subject: "svc.*"}},
		{name: "empty subject", in: entities.PublishRequest{ConnectionID: "conn-1", Subject: "  "}},
		{
			name: "invalid header name",
			in:   entities.PublishRequest{ConnectionID: "conn-1", Subject: "svc.ping", Headers: map[string]string{"Bad Key": "v"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s, _ := newRequestService(t, &mockProtoService{},
				func(context.Context, string, string, []byte, map[string]string) (*entities.Reply, error) {
					t.Fatal("nats.Request must not be called for invalid input")
					return nil, nil
				})

			_, err := s.Request(t.Context(), &entities.RequestMessage{PublishRequest: tt.in})
			require.ErrorIs(t, err, errs.ErrNATSInvalidArgument)
		})
	}
}

func TestRequest_PropagatesNATSErrors(t *testing.T) {
	t.Parallel()

	s, _ := newRequestService(t, &mockProtoService{},
		func(context.Context, string, string, []byte, map[string]string) (*entities.Reply, error) {
			return nil, errs.ErrNATSNoResponders
		})

	_, err := s.Request(t.Context(), &entities.RequestMessage{
		ConnectionID: "conn-1", Subject: "svc.ping",
	})
	require.ErrorIs(t, err, errs.ErrNATSNoResponders)
}
