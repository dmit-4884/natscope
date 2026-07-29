// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package publish

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"

	historysvc "github.com/dmit-4884/natscope/internal/services/history"
	natssvc "github.com/dmit-4884/natscope/internal/services/nats"
	protosvc "github.com/dmit-4884/natscope/internal/services/proto"
	settingssvc "github.com/dmit-4884/natscope/internal/services/settings"
)

// --- Mock services ---
// Each mock embeds the interface as a nil field; unused methods panic if called.

type mockProtoService struct {
	protosvc.Codec
	encodeRawFn func(ctx context.Context, req entities.CodecRequest) ([]byte, error)
}

func (m *mockProtoService) EncodeRaw(ctx context.Context, req entities.CodecRequest) ([]byte, error) {
	return m.encodeRawFn(ctx, req)
}

type mockNATSService struct {
	natssvc.ConnectionManager
	natssvc.Publisher
	publishFn func(ctx context.Context, connID, subject string, data []byte, headers map[string]string) (*entities.PubAck, error)
	urlFn     func(connID string) (string, error)
}

func (m *mockNATSService) PublishToStream(ctx context.Context, connID, subject string, data []byte, headers map[string]string) (*entities.PubAck, error) {
	return m.publishFn(ctx, connID, subject, data, headers)
}

func (m *mockNATSService) GetConnectionURL(_ context.Context, connID string) (string, error) {
	if m.urlFn == nil {
		return "", nil
	}
	return m.urlFn(connID)
}

type recordedHistory struct {
	mu     sync.Mutex
	last   *entities.PublishHistoryCreate
	called int
	err    error
}

type mockHistoryService struct {
	historysvc.Service
	rec *recordedHistory
}

func (m *mockHistoryService) Record(_ context.Context, in *entities.PublishHistoryCreate) (*entities.PublishHistory, error) {
	m.rec.mu.Lock()
	defer m.rec.mu.Unlock()
	m.rec.last = in
	m.rec.called++
	if m.rec.err != nil {
		return nil, m.rec.err
	}
	return entities.PublishHistoryNew(), nil
}

type mockSettingsService struct {
	settingssvc.Service
	settings *entities.UserSettings
}

func (m *mockSettingsService) Get(_ context.Context) (*entities.UserSettings, error) {
	return m.settings, nil
}

// --- Tests ---

func TestPublish_RawJSON(t *testing.T) {
	t.Parallel()

	hist := &recordedHistory{}
	natsm := &mockNATSService{
		publishFn: func(_ context.Context, _, subject string, data []byte, _ map[string]string) (*entities.PubAck, error) {
			assert.Equal(t, "orders.created", subject)
			assert.Equal(t, []byte(`{"id":1}`), data, "raw JSON bytes should be passed through")
			return &entities.PubAck{Stream: "ORDERS", Sequence: 42}, nil
		},
	}
	s := New(natsm, natsm, &mockProtoService{}, &mockHistoryService{rec: hist}, &mockSettingsService{})

	resp, err := s.Publish(t.Context(), &entities.PublishRequest{
		ConnectionID: "conn-1",
		Subject:      "orders.created",
		Data:         `{"id":1}`,
	})

	require.NoError(t, err)
	require.Nil(t, resp.Error)
	assert.Equal(t, "ORDERS", resp.Stream)
	assert.Equal(t, uint64(42), resp.Sequence)

	hist.mu.Lock()
	defer hist.mu.Unlock()
	require.Equal(t, 1, hist.called)
	require.NotNil(t, hist.last)
	assert.Equal(t, entities.EncodingTypeJSON, hist.last.EncodingType)
	assert.True(t, hist.last.Success)
	require.NotNil(t, hist.last.Sequence)
	assert.Equal(t, uint64(42), *hist.last.Sequence)
}

func TestPublish_ProtoEncoded(t *testing.T) {
	t.Parallel()

	hist := &recordedHistory{}
	encoded := []byte{0x08, 0x01}
	proto := &mockProtoService{
		encodeRawFn: func(_ context.Context, req entities.CodecRequest) ([]byte, error) {
			assert.Equal(t, "api.v1.Order", req.MessageType)
			assert.Equal(t, "src-1", req.SourceID)
			var v map[string]interface{}
			require.NoError(t, json.Unmarshal(req.JSON, &v))
			return encoded, nil
		},
	}
	natsm := &mockNATSService{
		publishFn: func(_ context.Context, _, _ string, data []byte, _ map[string]string) (*entities.PubAck, error) {
			assert.Equal(t, encoded, data, "encoded protobuf bytes should be published")
			return &entities.PubAck{Stream: "ORDERS", Sequence: 7}, nil
		},
	}
	s := New(natsm, natsm, proto, &mockHistoryService{rec: hist}, &mockSettingsService{})

	msgType := "api.v1.Order"
	sourceID := "src-1"
	resp, err := s.Publish(t.Context(), &entities.PublishRequest{
		ConnectionID: "c1",
		Subject:      "orders.created",
		Data:         `{"id":1}`,
		MessageType:  &msgType,
		SourceID:     &sourceID,
	})

	require.NoError(t, err)
	require.Nil(t, resp.Error)

	hist.mu.Lock()
	defer hist.mu.Unlock()
	assert.Equal(t, entities.EncodingTypeProtobuf, hist.last.EncodingType)
	assert.Equal(t, "api.v1.Order", hist.last.MessageType)
}

func TestPublish_EncodeError(t *testing.T) {
	t.Parallel()

	hist := &recordedHistory{}
	encErr := errors.New("bad json")
	proto := &mockProtoService{
		encodeRawFn: func(_ context.Context, _ entities.CodecRequest) ([]byte, error) { return nil, encErr },
	}
	natsm := &mockNATSService{
		publishFn: func(_ context.Context, _, _ string, _ []byte, _ map[string]string) (*entities.PubAck, error) {
			t.Fatal("nats.PublishToStream should not be called on encode error")
			return nil, nil
		},
	}
	s := New(natsm, natsm, proto, &mockHistoryService{rec: hist}, &mockSettingsService{})

	msgType := "api.v1.Bad"
	sourceID := "src-1"
	resp, err := s.Publish(t.Context(), &entities.PublishRequest{
		ConnectionID: "c1", Subject: "x", Data: "not json", MessageType: &msgType, SourceID: &sourceID,
	})

	require.NoError(t, err) // soft failure — Error in body, not transport error
	require.NotNil(t, resp.Error)
	assert.Contains(t, *resp.Error, "bad json")
	assert.Equal(t, 0, hist.called, "encode failures are not recorded to history")
}

func TestPublish_PublishError_RecordedAsFailure(t *testing.T) {
	t.Parallel()

	hist := &recordedHistory{}
	natsm := &mockNATSService{
		publishFn: func(_ context.Context, _, _ string, _ []byte, _ map[string]string) (*entities.PubAck, error) {
			return nil, errors.New("stream not found")
		},
	}
	s := New(natsm, natsm, &mockProtoService{}, &mockHistoryService{rec: hist}, &mockSettingsService{})

	resp, err := s.Publish(t.Context(), &entities.PublishRequest{
		ConnectionID: "c1", Subject: "x", Data: "{}",
	})

	require.NoError(t, err)
	require.NotNil(t, resp.Error)
	assert.Contains(t, *resp.Error, "stream not found")

	hist.mu.Lock()
	defer hist.mu.Unlock()
	require.Equal(t, 1, hist.called, "publish failure must still be recorded to history")
	require.NotNil(t, hist.last)
	assert.False(t, hist.last.Success)
	require.NotNil(t, hist.last.Error)
	assert.Contains(t, *hist.last.Error, "stream not found")
}

func TestPublish_HistoryRecordFailureDoesNotPropagate(t *testing.T) {
	t.Parallel()

	hist := &recordedHistory{err: errors.New("disk full")}
	natsm := &mockNATSService{
		publishFn: func(_ context.Context, _, _ string, _ []byte, _ map[string]string) (*entities.PubAck, error) {
			return &entities.PubAck{Stream: "S", Sequence: 1}, nil
		},
	}
	s := New(natsm, natsm, &mockProtoService{}, &mockHistoryService{rec: hist}, &mockSettingsService{})

	resp, err := s.Publish(t.Context(), &entities.PublishRequest{
		ConnectionID: "c1", Subject: "x", Data: "{}",
	})

	require.NoError(t, err, "history failure must not bubble up to the caller")
	assert.Nil(t, resp.Error)
	assert.Equal(t, "S", resp.Stream)
}

func TestPublish_RespectsCustomTimeoutFromSettings(t *testing.T) {
	t.Parallel()

	called := make(chan struct{})
	natsm := &mockNATSService{
		publishFn: func(ctx context.Context, _, _ string, _ []byte, _ map[string]string) (*entities.PubAck, error) {
			deadline, ok := ctx.Deadline()
			require.True(t, ok, "publish must be called with a deadline")
			require.NotNil(t, deadline)
			close(called)
			return &entities.PubAck{Stream: "S", Sequence: 1}, nil
		},
	}
	timeoutSec := int32(7)
	settings := &entities.UserSettings{Publish: &entities.PublishSettings{PublishTimeoutSec: &timeoutSec}}

	s := New(natsm, natsm, &mockProtoService{}, &mockHistoryService{rec: &recordedHistory{}}, &mockSettingsService{settings: settings})

	_, err := s.Publish(t.Context(), &entities.PublishRequest{
		ConnectionID: "c1", Subject: "x", Data: "{}",
	})
	require.NoError(t, err)
	<-called
}

func TestPublish_MissingSourceIDForMessageType(t *testing.T) {
	t.Parallel()

	hist := &recordedHistory{}
	natsm := &mockNATSService{
		publishFn: func(_ context.Context, _, _ string, _ []byte, _ map[string]string) (*entities.PubAck, error) {
			t.Fatal("nats.PublishToStream should not be called when validation fails up-front")
			return nil, nil
		},
	}
	s := New(natsm, natsm, &mockProtoService{}, &mockHistoryService{rec: hist}, &mockSettingsService{})

	msgType := "api.v1.Order"
	resp, err := s.Publish(t.Context(), &entities.PublishRequest{
		ConnectionID: "c1", Subject: "x", Data: "{}", MessageType: &msgType,
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Error)
	assert.Contains(t, *resp.Error, "source_id is required")
	assert.Equal(t, 0, hist.called)
}
