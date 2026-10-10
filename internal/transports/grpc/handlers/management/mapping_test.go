// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package management

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"

	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	natssvc "github.com/dmit-4884/natscope/internal/services/nats"
	managementpb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/management"
	natspb "github.com/dmit-4884/natscope/proto/gen/types/nats"
)

type objectStoreStub struct {
	natssvc.ObjectStore

	info *entities.ObjectInfo
}

func (s *objectStoreStub) GetObject(_ context.Context, _ string, _ string, _ string) ([]byte, *entities.ObjectInfo, error) {
	return nil, s.info, nil
}

func TestHandler_CreateStream_MapsNestedConfig(t *testing.T) {
	t.Parallel()
	svc := &mockNatsService{streamInfo: &entities.StreamInfo{}}
	handler := New(svc, svc, svc, svc, svc, nil, nil)

	start := "2026-01-02T03:04:05Z"
	_, err := handler.CreateStream(t.Context(), connect.NewRequest(&managementpb.CreateStreamRequest{
		ConnectionId:     "conn-1",
		Name:             "orders",
		Placement:        &managementpb.PlacementConfig{Cluster: "c1", Tags: []string{"a", "b"}},
		SubjectTransform: &managementpb.SubjectTransform{Source: "in.>", Destination: "out.>"},
		Republish:        &natspb.RePublish{Src: "s", Dest: "d", HeadersOnly: true},
		ConsumerLimits:   &natspb.ConsumerLimits{InactiveThreshold: durationpb.New(time.Minute), MaxAckPending: 7},
		Mirror: &managementpb.StreamSourceConfig{
			Name:              "src",
			OptStartSeq:       3,
			OptStartTime:      &start,
			FilterSubject:     "f",
			SubjectTransforms: []*managementpb.SubjectTransform{{Source: "x", Destination: "y"}},
			External:          &managementpb.ExternalStreamConfig{ApiPrefix: "api", DeliverPrefix: "dlv"},
		},
		Sources: []*managementpb.StreamSourceConfig{{Name: "s1"}},
	}))
	require.NoError(t, err)

	got := svc.createdStream
	assert.Equal(t, &entities.Placement{Cluster: "c1", Tags: []string{"a", "b"}}, got.Placement)
	assert.Equal(t, &entities.SubjectTransformConfig{Source: "in.>", Destination: "out.>"}, got.SubjectTransform)
	assert.Equal(t, &entities.StreamRePublish{Src: "s", Dest: "d", HeadersOnly: true}, got.Republish)
	assert.Equal(t, &entities.StreamConsumerLimits{InactiveThreshold: time.Minute, MaxAckPending: 7}, got.ConsumerLimits)
	require.NotNil(t, got.Mirror)
	assert.Equal(t, "src", got.Mirror.Name)
	assert.Equal(t, uint64(3), got.Mirror.OptStartSeq)
	assert.Equal(t, "f", got.Mirror.FilterSubject)
	require.NotNil(t, got.Mirror.OptStartTime)
	assert.True(t, time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC).Equal(*got.Mirror.OptStartTime))
	assert.Equal(t, []*entities.SubjectTransformConfig{{Source: "x", Destination: "y"}}, got.Mirror.SubjectTransforms)
	assert.Equal(t, &entities.ExternalStream{APIPrefix: "api", DeliverPrefix: "dlv"}, got.Mirror.External)
	require.Len(t, got.Sources, 1)
	assert.Equal(t, "s1", got.Sources[0].Name)
	assert.Nil(t, got.Sources[0].External)
	assert.Nil(t, got.Sources[0].OptStartTime)
}

func TestHandler_CreateStream_LeavesUnsetNestedConfigNil(t *testing.T) {
	t.Parallel()
	svc := &mockNatsService{streamInfo: &entities.StreamInfo{}}
	handler := New(svc, svc, svc, svc, svc, nil, nil)

	_, err := handler.CreateStream(t.Context(), connect.NewRequest(&managementpb.CreateStreamRequest{ConnectionId: "conn-1", Name: "orders"}))
	require.NoError(t, err)

	got := svc.createdStream
	assert.Nil(t, got.Placement)
	assert.Nil(t, got.Mirror)
	assert.Empty(t, got.Sources)
	assert.Nil(t, got.SubjectTransform)
	assert.Nil(t, got.Republish)
	assert.Nil(t, got.ConsumerLimits)
}

func TestHandler_CreateStream_RejectsMalformedStartTime(t *testing.T) {
	t.Parallel()
	svc := &mockNatsService{streamInfo: &entities.StreamInfo{}}
	handler := New(svc, svc, svc, svc, svc, nil, nil)

	bad := "yesterday"
	_, err := handler.CreateStream(t.Context(), connect.NewRequest(&managementpb.CreateStreamRequest{
		ConnectionId: "conn-1",
		Name:         "orders",
		Sources:      []*managementpb.StreamSourceConfig{{Name: "s1", OptStartTime: &bad}},
	}))

	var validation *errs.NATSValidationError
	require.ErrorAs(t, err, &validation)
}

func TestHandler_UpdateStream_MapsNestedConfig(t *testing.T) {
	t.Parallel()
	svc := &mockNatsService{streamInfo: &entities.StreamInfo{}}
	handler := New(svc, svc, svc, svc, svc, nil, nil)

	_, err := handler.UpdateStream(t.Context(), connect.NewRequest(&managementpb.UpdateStreamRequest{
		ConnectionId:     "conn-1",
		StreamName:       "orders",
		SubjectTransform: &natspb.SubjectTransformConfig{Source: "in.>", Destination: "out.>"},
		Republish:        &natspb.RePublish{Src: "s", Dest: "d"},
		ConsumerLimits:   &natspb.ConsumerLimits{InactiveThreshold: durationpb.New(time.Second), MaxAckPending: 2},
		Sources:          []*managementpb.StreamSourceConfig{{Name: "s1", External: &managementpb.ExternalStreamConfig{ApiPrefix: "api"}}},
	}))
	require.NoError(t, err)

	got := svc.updatedStream
	assert.Equal(t, &entities.SubjectTransformConfig{Source: "in.>", Destination: "out.>"}, got.SubjectTransform)
	assert.Equal(t, &entities.StreamRePublish{Src: "s", Dest: "d"}, got.Republish)
	assert.Equal(t, &entities.StreamConsumerLimits{InactiveThreshold: time.Second, MaxAckPending: 2}, got.ConsumerLimits)
	require.Len(t, got.Sources, 1)
	assert.Equal(t, &entities.ExternalStream{APIPrefix: "api"}, got.Sources[0].External)
}

func TestHandler_UpdateStream_LeavesUnsetNestedConfigNil(t *testing.T) {
	t.Parallel()
	svc := &mockNatsService{streamInfo: &entities.StreamInfo{}}
	handler := New(svc, svc, svc, svc, svc, nil, nil)

	_, err := handler.UpdateStream(t.Context(), connect.NewRequest(&managementpb.UpdateStreamRequest{ConnectionId: "conn-1", StreamName: "orders"}))
	require.NoError(t, err)

	got := svc.updatedStream
	assert.Nil(t, got.SubjectTransform)
	assert.Nil(t, got.Republish)
	assert.Nil(t, got.ConsumerLimits)
	assert.Empty(t, got.Sources)
}

func TestHandler_CreateKVBucket_MapsNestedConfig(t *testing.T) {
	t.Parallel()
	svc := &mockNatsService{kvInfo: &entities.KVBucketInfo{Bucket: "kv1"}}
	handler := New(svc, svc, svc, svc, svc, nil, nil)

	start := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	_, err := handler.CreateKVBucket(t.Context(), connect.NewRequest(&managementpb.CreateKVBucketRequest{
		ConnectionId: "conn-1",
		Config: &natspb.KVBucketConfig{
			Bucket:      "kv1",
			History:     5,
			NumReplicas: 3,
			Ttl:         durationpb.New(time.Hour),
			Placement:   &natspb.Placement{Cluster: "c1", Tags: []string{"t"}},
			Mirror: &natspb.StreamSourceRef{
				Name:              "m",
				OptStartSeq:       9,
				FilterSubject:     "f",
				External:          &natspb.ExternalStream{ApiPrefix: "api", DeliverPrefix: "dlv"},
				OptStartTime:      timestamppb.New(start),
				SubjectTransforms: []*natspb.SubjectTransformConfig{{Source: "a", Destination: "b"}},
			},
			Sources:   []*natspb.StreamSourceRef{{Name: "s1"}, {Name: "s2", OptStartTime: timestamppb.New(start)}},
			Republish: &natspb.RePublish{Src: "s", Dest: "d", HeadersOnly: true},
		},
	}))
	require.NoError(t, err)

	got := svc.createdKV
	assert.Equal(t, uint8(5), got.History)
	assert.Equal(t, 3, got.Replicas)
	assert.Equal(t, time.Hour, got.TTL)
	assert.Equal(t, &entities.Placement{Cluster: "c1", Tags: []string{"t"}}, got.Placement)
	assert.Equal(t, &entities.StreamRePublish{Src: "s", Dest: "d", HeadersOnly: true}, got.Republish)
	require.NotNil(t, got.Mirror)
	assert.Equal(t, "m", got.Mirror.Name)
	assert.Equal(t, uint64(9), got.Mirror.OptStartSeq)
	assert.Equal(t, "f", got.Mirror.FilterSubject)
	assert.Equal(t, &entities.ExternalStream{APIPrefix: "api", DeliverPrefix: "dlv"}, got.Mirror.External)
	require.NotNil(t, got.Mirror.OptStartTime)
	assert.True(t, start.Equal(*got.Mirror.OptStartTime))
	assert.Equal(t, []*entities.SubjectTransformConfig{{Source: "a", Destination: "b"}}, got.Mirror.SubjectTransforms)
	require.Len(t, got.Sources, 2)
	assert.Nil(t, got.Sources[0].OptStartTime)
	assert.Nil(t, got.Sources[0].External)
	require.NotNil(t, got.Sources[1].OptStartTime)
}

func TestHandler_CreateKVBucket_LeavesUnsetNestedConfigNil(t *testing.T) {
	t.Parallel()
	svc := &mockNatsService{kvInfo: &entities.KVBucketInfo{Bucket: "kv1"}}
	handler := New(svc, svc, svc, svc, svc, nil, nil)

	_, err := handler.CreateKVBucket(t.Context(), connect.NewRequest(&managementpb.CreateKVBucketRequest{
		ConnectionId: "conn-1",
		Config:       &natspb.KVBucketConfig{Bucket: "kv1"},
	}))
	require.NoError(t, err)

	got := svc.createdKV
	assert.Nil(t, got.Placement)
	assert.Nil(t, got.Mirror)
	assert.Empty(t, got.Sources)
	assert.Nil(t, got.Republish)
}

func TestHandler_CreateObjectBucket_MapsPlacement(t *testing.T) {
	t.Parallel()
	svc := &mockNatsService{objInfo: &entities.ObjectBucketInfo{Bucket: "obj1"}}
	handler := New(svc, svc, svc, svc, svc, nil, nil)

	_, err := handler.CreateObjectBucket(t.Context(), connect.NewRequest(&managementpb.CreateObjectBucketRequest{
		ConnectionId: "conn-1",
		Config:       &natspb.ObjectBucketConfig{Bucket: "obj1", NumReplicas: 2, Placement: &natspb.Placement{Cluster: "c1", Tags: []string{"t"}}},
	}))
	require.NoError(t, err)

	assert.Equal(t, 2, svc.createdObject.Replicas)
	assert.Equal(t, &entities.Placement{Cluster: "c1", Tags: []string{"t"}}, svc.createdObject.Placement)

	_, err = handler.CreateObjectBucket(t.Context(), connect.NewRequest(&managementpb.CreateObjectBucketRequest{
		ConnectionId: "conn-1",
		Config:       &natspb.ObjectBucketConfig{Bucket: "obj1"},
	}))
	require.NoError(t, err)
	assert.Nil(t, svc.createdObject.Placement)
}

func TestHandler_GetObject_MapsLink(t *testing.T) {
	t.Parallel()
	modTime := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	t.Run("WithLink", func(t *testing.T) {
		t.Parallel()
		store := &objectStoreStub{info: &entities.ObjectInfo{
			Name:    "o",
			ModTime: modTime,
			Link:    &entities.ObjectLink{Bucket: "b", Name: "n"},
		}}
		handler := New(nil, nil, nil, nil, store, nil, nil)

		resp, err := handler.GetObject(t.Context(), connect.NewRequest(&managementpb.GetObjectRequest{ConnectionId: "conn-1", Bucket: "b", Name: "o"}))
		require.NoError(t, err)
		assert.Equal(t, "o", resp.Msg.Info.Name)
		assert.True(t, modTime.Equal(resp.Msg.Info.ModTime.AsTime()))
		assert.Equal(t, "b", resp.Msg.Info.Link.Bucket)
		assert.Equal(t, "n", resp.Msg.Info.Link.Name)
	})

	t.Run("WithoutLink", func(t *testing.T) {
		t.Parallel()
		store := &objectStoreStub{info: &entities.ObjectInfo{Name: "o", ModTime: modTime}}
		handler := New(nil, nil, nil, nil, store, nil, nil)

		resp, err := handler.GetObject(t.Context(), connect.NewRequest(&managementpb.GetObjectRequest{ConnectionId: "conn-1", Bucket: "b", Name: "o"}))
		require.NoError(t, err)
		assert.Nil(t, resp.Msg.Info.Link)
	})
}
