// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package management

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"

	natssvc "github.com/dmit-4884/natscope/internal/services/nats"
	managementpb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/management"
	natspb "github.com/dmit-4884/natscope/proto/gen/types/nats"
)

// --- Mocks ---

// mockNatsService embeds the management handler's narrow NATS roles so only the
// management-exercised methods need overriding; unset methods panic if ever called.
type mockNatsService struct {
	natssvc.StreamReader
	natssvc.StreamManager
	natssvc.ConsumerManager
	natssvc.KVStore
	natssvc.ObjectStore

	streamInfo *entities.StreamInfo
	streamErr  error

	consumerInfo *entities.ConsumerInfo
	consumers    []entities.ConsumerInfo
	consumerErr  error

	kvInfo *entities.KVBucketInfo
	kvErr  error

	objInfo *entities.ObjectBucketInfo
	objErr  error

	deletedStream   string
	deletedConsumer string
	deletedBucket   string
}

func (m *mockNatsService) CreateStream(
	_ context.Context,
	_ string,
	_ entities.StreamCreateRequest,
) (*entities.StreamInfo, error) {
	return m.streamInfo, m.streamErr
}

func (m *mockNatsService) DeleteStream(_ context.Context, _ string, name string) error {
	m.deletedStream = name
	return m.streamErr
}

func (m *mockNatsService) GetStreamConsumers(
	_ context.Context,
	_ string,
	_ string,
) ([]entities.ConsumerInfo, error) {
	return m.consumers, m.consumerErr
}

func (m *mockNatsService) CreateConsumer(
	_ context.Context,
	_ string,
	_ string,
	_ entities.ConsumerCreateRequest,
) (*entities.ConsumerInfo, error) {
	return m.consumerInfo, m.consumerErr
}

func (m *mockNatsService) DeleteConsumer(_ context.Context, _ string, _ string, name string) error {
	m.deletedConsumer = name
	return m.consumerErr
}

func (m *mockNatsService) CreateKVBucket(
	_ context.Context,
	_ string,
	_ entities.KVBucketConfig,
) (*entities.KVBucketInfo, error) {
	return m.kvInfo, m.kvErr
}

func (m *mockNatsService) GetKVBucket(_ context.Context, _ string, _ string) (*entities.KVBucketInfo, error) {
	return m.kvInfo, m.kvErr
}

func (m *mockNatsService) DeleteKVBucket(_ context.Context, _ string, bucket string) error {
	m.deletedBucket = bucket
	return m.kvErr
}

func (m *mockNatsService) CreateObjectBucket(
	_ context.Context,
	_ string,
	_ entities.ObjectBucketConfig,
) (*entities.ObjectBucketInfo, error) {
	return m.objInfo, m.objErr
}

func (m *mockNatsService) GetObjectBucket(_ context.Context, _ string, _ string) (*entities.ObjectBucketInfo, error) {
	return m.objInfo, m.objErr
}

func (m *mockNatsService) DeleteObjectBucket(_ context.Context, _ string, bucket string) error {
	m.deletedBucket = bucket
	return m.objErr
}

// --- Stream tests ---

func TestHandler_CreateStream(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockNatsService{streamInfo: &entities.StreamInfo{}}
		handler := New(svc, svc, svc, svc, svc)

		resp, err := handler.CreateStream(t.Context(), connect.NewRequest(&managementpb.CreateStreamRequest{
			ConnectionId: "conn-1",
			Name:         "orders",
			Subjects:     []string{"orders.>"},
		}))
		require.NoError(t, err)
		require.NotNil(t, resp.Msg.Stream)
	})

	t.Run("ServiceError", func(t *testing.T) {
		t.Parallel()
		svc := &mockNatsService{streamErr: errs.ErrJetStreamNotEnabled}
		handler := New(svc, svc, svc, svc, svc)

		_, err := handler.CreateStream(t.Context(), connect.NewRequest(&managementpb.CreateStreamRequest{
			ConnectionId: "conn-1",
			Name:         "orders",
		}))
		assert.ErrorIs(t, err, errs.ErrJetStreamNotEnabled)
	})
}

func TestHandler_DeleteStream(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockNatsService{}
		handler := New(svc, svc, svc, svc, svc)

		resp, err := handler.DeleteStream(t.Context(), connect.NewRequest(&managementpb.DeleteStreamRequest{
			ConnectionId: "conn-1",
			StreamName:   "orders",
		}))
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "orders", svc.deletedStream)
	})

	t.Run("ServiceError", func(t *testing.T) {
		t.Parallel()
		svc := &mockNatsService{streamErr: errs.ErrNATSTimeout}
		handler := New(svc, svc, svc, svc, svc)

		_, err := handler.DeleteStream(t.Context(), connect.NewRequest(&managementpb.DeleteStreamRequest{
			ConnectionId: "conn-1",
			StreamName:   "orders",
		}))
		assert.ErrorIs(t, err, errs.ErrNATSTimeout)
	})
}

// --- Consumer tests ---

func TestHandler_ListConsumers(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockNatsService{consumers: []entities.ConsumerInfo{
			{Name: "c1", Stream: "orders"},
			{Name: "c2", Stream: "orders"},
		}}
		handler := New(svc, svc, svc, svc, svc)

		resp, err := handler.ListConsumers(t.Context(), connect.NewRequest(&managementpb.ListConsumersRequest{
			ConnectionId: "conn-1",
			StreamName:   "orders",
		}))
		require.NoError(t, err)
		require.Len(t, resp.Msg.Consumers, 2)
		assert.Equal(t, "c1", resp.Msg.Consumers[0].Name)
	})

	t.Run("ServiceError", func(t *testing.T) {
		t.Parallel()
		svc := &mockNatsService{consumerErr: errors.New("fetch failed")}
		handler := New(svc, svc, svc, svc, svc)

		_, err := handler.ListConsumers(t.Context(), connect.NewRequest(&managementpb.ListConsumersRequest{
			ConnectionId: "conn-1",
			StreamName:   "orders",
		}))
		assert.Error(t, err)
	})
}

func TestHandler_CreateConsumer(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockNatsService{consumerInfo: &entities.ConsumerInfo{Name: "c1", Stream: "orders"}}
		handler := New(svc, svc, svc, svc, svc)

		resp, err := handler.CreateConsumer(t.Context(), connect.NewRequest(&managementpb.CreateConsumerRequest{
			ConnectionId: "conn-1",
			StreamName:   "orders",
			Name:         "c1",
		}))
		require.NoError(t, err)
		require.NotNil(t, resp.Msg.Consumer)
		assert.Equal(t, "c1", resp.Msg.Consumer.Name)
	})

	t.Run("ServiceError", func(t *testing.T) {
		t.Parallel()
		svc := &mockNatsService{consumerErr: errs.ErrWorkQueueConsumerNotAllowed}
		handler := New(svc, svc, svc, svc, svc)

		_, err := handler.CreateConsumer(t.Context(), connect.NewRequest(&managementpb.CreateConsumerRequest{
			ConnectionId: "conn-1",
			StreamName:   "orders",
			Name:         "c1",
		}))
		assert.ErrorIs(t, err, errs.ErrWorkQueueConsumerNotAllowed)
	})
}

func TestHandler_DeleteConsumer(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockNatsService{}
		handler := New(svc, svc, svc, svc, svc)

		resp, err := handler.DeleteConsumer(t.Context(), connect.NewRequest(&managementpb.DeleteConsumerRequest{
			ConnectionId: "conn-1",
			StreamName:   "orders",
			ConsumerName: "c1",
		}))
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "c1", svc.deletedConsumer)
	})

	t.Run("ServiceError", func(t *testing.T) {
		t.Parallel()
		svc := &mockNatsService{consumerErr: errs.ErrNATSTimeout}
		handler := New(svc, svc, svc, svc, svc)

		_, err := handler.DeleteConsumer(t.Context(), connect.NewRequest(&managementpb.DeleteConsumerRequest{
			ConnectionId: "conn-1",
			StreamName:   "orders",
			ConsumerName: "c1",
		}))
		assert.ErrorIs(t, err, errs.ErrNATSTimeout)
	})
}

// --- KV tests ---

func TestHandler_CreateKVBucket(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockNatsService{kvInfo: &entities.KVBucketInfo{Bucket: "kv1"}}
		handler := New(svc, svc, svc, svc, svc)

		resp, err := handler.CreateKVBucket(t.Context(), connect.NewRequest(&managementpb.CreateKVBucketRequest{
			ConnectionId: "conn-1",
			Config:       &natspb.KVBucketConfig{Bucket: "kv1"},
		}))
		require.NoError(t, err)
		require.NotNil(t, resp.Msg.Bucket)
		assert.Equal(t, "kv1", resp.Msg.Bucket.Bucket)
	})

	t.Run("ServiceError", func(t *testing.T) {
		t.Parallel()
		svc := &mockNatsService{kvErr: errs.ErrJetStreamNotEnabled}
		handler := New(svc, svc, svc, svc, svc)

		_, err := handler.CreateKVBucket(t.Context(), connect.NewRequest(&managementpb.CreateKVBucketRequest{
			ConnectionId: "conn-1",
			Config:       &natspb.KVBucketConfig{Bucket: "kv1"},
		}))
		assert.ErrorIs(t, err, errs.ErrJetStreamNotEnabled)
	})
}

func TestHandler_GetKVBucket(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockNatsService{kvInfo: &entities.KVBucketInfo{Bucket: "kv1"}}
		handler := New(svc, svc, svc, svc, svc)

		resp, err := handler.GetKVBucket(t.Context(), connect.NewRequest(&managementpb.GetKVBucketRequest{
			ConnectionId: "conn-1",
			Bucket:       "kv1",
		}))
		require.NoError(t, err)
		require.NotNil(t, resp.Msg.Bucket)
		assert.Equal(t, "kv1", resp.Msg.Bucket.Bucket)
	})

	t.Run("NotFound", func(t *testing.T) {
		t.Parallel()
		svc := &mockNatsService{kvErr: errs.ErrNotFound}
		handler := New(svc, svc, svc, svc, svc)

		_, err := handler.GetKVBucket(t.Context(), connect.NewRequest(&managementpb.GetKVBucketRequest{
			ConnectionId: "conn-1",
			Bucket:       "missing",
		}))
		assert.ErrorIs(t, err, errs.ErrNotFound)
	})
}

func TestHandler_DeleteKVBucket(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockNatsService{}
		handler := New(svc, svc, svc, svc, svc)

		resp, err := handler.DeleteKVBucket(t.Context(), connect.NewRequest(&managementpb.DeleteKVBucketRequest{
			ConnectionId: "conn-1",
			Bucket:       "kv1",
		}))
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "kv1", svc.deletedBucket)
	})

	t.Run("ServiceError", func(t *testing.T) {
		t.Parallel()
		svc := &mockNatsService{kvErr: errors.New("delete failed")}
		handler := New(svc, svc, svc, svc, svc)

		_, err := handler.DeleteKVBucket(t.Context(), connect.NewRequest(&managementpb.DeleteKVBucketRequest{
			ConnectionId: "conn-1",
			Bucket:       "kv1",
		}))
		assert.Error(t, err)
	})
}

// --- Object store tests ---

func TestHandler_CreateObjectBucket(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockNatsService{objInfo: &entities.ObjectBucketInfo{Bucket: "obj1"}}
		handler := New(svc, svc, svc, svc, svc)

		resp, err := handler.CreateObjectBucket(t.Context(), connect.NewRequest(&managementpb.CreateObjectBucketRequest{
			ConnectionId: "conn-1",
			Config:       &natspb.ObjectBucketConfig{Bucket: "obj1"},
		}))
		require.NoError(t, err)
		require.NotNil(t, resp.Msg.Bucket)
		assert.Equal(t, "obj1", resp.Msg.Bucket.Bucket)
	})

	t.Run("ServiceError", func(t *testing.T) {
		t.Parallel()
		svc := &mockNatsService{objErr: errs.ErrObjectAlreadyExists}
		handler := New(svc, svc, svc, svc, svc)

		_, err := handler.CreateObjectBucket(t.Context(), connect.NewRequest(&managementpb.CreateObjectBucketRequest{
			ConnectionId: "conn-1",
			Config:       &natspb.ObjectBucketConfig{Bucket: "obj1"},
		}))
		assert.ErrorIs(t, err, errs.ErrObjectAlreadyExists)
	})
}

func TestHandler_GetObjectBucket(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockNatsService{objInfo: &entities.ObjectBucketInfo{Bucket: "obj1"}}
		handler := New(svc, svc, svc, svc, svc)

		resp, err := handler.GetObjectBucket(t.Context(), connect.NewRequest(&managementpb.GetObjectBucketRequest{
			ConnectionId: "conn-1",
			Bucket:       "obj1",
		}))
		require.NoError(t, err)
		require.NotNil(t, resp.Msg.Bucket)
		assert.Equal(t, "obj1", resp.Msg.Bucket.Bucket)
	})

	t.Run("NotFound", func(t *testing.T) {
		t.Parallel()
		svc := &mockNatsService{objErr: errs.ErrNotFound}
		handler := New(svc, svc, svc, svc, svc)

		_, err := handler.GetObjectBucket(t.Context(), connect.NewRequest(&managementpb.GetObjectBucketRequest{
			ConnectionId: "conn-1",
			Bucket:       "missing",
		}))
		assert.ErrorIs(t, err, errs.ErrNotFound)
	})
}

func TestHandler_DeleteObjectBucket(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		svc := &mockNatsService{}
		handler := New(svc, svc, svc, svc, svc)

		resp, err := handler.DeleteObjectBucket(t.Context(), connect.NewRequest(&managementpb.DeleteObjectBucketRequest{
			ConnectionId: "conn-1",
			Bucket:       "obj1",
		}))
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "obj1", svc.deletedBucket)
	})

	t.Run("ServiceError", func(t *testing.T) {
		t.Parallel()
		svc := &mockNatsService{objErr: errors.New("delete failed")}
		handler := New(svc, svc, svc, svc, svc)

		_, err := handler.DeleteObjectBucket(t.Context(), connect.NewRequest(&managementpb.DeleteObjectBucketRequest{
			ConnectionId: "conn-1",
			Bucket:       "obj1",
		}))
		assert.Error(t, err)
	})
}
