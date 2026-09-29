// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package management

import (
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"

	"google.golang.org/protobuf/types/known/durationpb"

	managementpb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/management"
)

// TestHandler_StreamFeatureFlags verifies the NATS 2.11–2.14 stream flags
// travel proto → entity on create/update and entity → proto on the response.
func TestHandler_StreamFeatureFlags(t *testing.T) {
	t.Parallel()

	t.Run("create", func(t *testing.T) {
		t.Parallel()
		svc := &mockNatsService{streamInfo: &entities.StreamInfo{Config: entities.StreamConfig{
			Name:                   "flags",
			AllowMsgCounter:        true,
			AllowMsgSchedules:      true,
			SubjectDeleteMarkerTTL: time.Minute,
			PersistMode:            entities.PersistAsync,
			AllowBatchPublish:      true,
		}}}
		handler := New(svc, svc, svc, svc, svc)

		resp, err := handler.CreateStream(t.Context(), connect.NewRequest(&managementpb.CreateStreamRequest{
			ConnectionId:           "conn-1",
			Name:                   "flags",
			Subjects:               []string{"flags.>"},
			AllowMsgCounter:        true,
			AllowMsgSchedules:      true,
			SubjectDeleteMarkerTtl: durationpb.New(time.Minute),
			PersistMode:            1,
			AllowBatchPublish:      true,
		}))
		require.NoError(t, err)

		got := svc.createdStream
		assert.True(t, got.AllowMsgCounter)
		assert.True(t, got.AllowMsgSchedules)
		assert.Equal(t, time.Minute, got.SubjectDeleteMarkerTTL)
		assert.Equal(t, entities.PersistAsync, got.PersistMode)
		assert.True(t, got.AllowBatchPublish)

		cfg := resp.Msg.GetStream().GetConfig()
		require.NotNil(t, cfg)
		assert.True(t, cfg.GetAllowMsgCounter())
		assert.True(t, cfg.GetAllowMsgSchedules())
		assert.Equal(t, time.Minute, cfg.GetSubjectDeleteMarkerTtl().AsDuration())
		assert.Equal(t, int32(1), cfg.GetPersistMode())
		assert.True(t, cfg.GetAllowBatchPublish())
	})

	t.Run("update sets mutable flags", func(t *testing.T) {
		t.Parallel()
		svc := &mockNatsService{streamInfo: &entities.StreamInfo{}}
		handler := New(svc, svc, svc, svc, svc)

		_, err := handler.UpdateStream(t.Context(), connect.NewRequest(&managementpb.UpdateStreamRequest{
			ConnectionId:           "conn-1",
			StreamName:             "flags",
			AllowMsgSchedules:      new(true),
			SubjectDeleteMarkerTtl: durationpb.New(30 * time.Second),
			AllowBatchPublish:      new(false),
		}))
		require.NoError(t, err)

		got := svc.updatedStream
		require.NotNil(t, got.AllowMsgSchedules)
		assert.True(t, *got.AllowMsgSchedules)
		require.NotNil(t, got.SubjectDeleteMarkerTTL)
		assert.Equal(t, 30*time.Second, *got.SubjectDeleteMarkerTTL)
		require.NotNil(t, got.AllowBatchPublish)
		assert.False(t, *got.AllowBatchPublish)
	})

	t.Run("update leaves unset flags nil", func(t *testing.T) {
		t.Parallel()
		svc := &mockNatsService{streamInfo: &entities.StreamInfo{}}
		handler := New(svc, svc, svc, svc, svc)

		_, err := handler.UpdateStream(t.Context(), connect.NewRequest(&managementpb.UpdateStreamRequest{
			ConnectionId: "conn-1",
			StreamName:   "flags",
		}))
		require.NoError(t, err)

		got := svc.updatedStream
		assert.Nil(t, got.AllowMsgSchedules)
		assert.Nil(t, got.SubjectDeleteMarkerTTL)
		assert.Nil(t, got.AllowBatchPublish)
	})

	t.Run("update with zero delete marker ttl clears it", func(t *testing.T) {
		t.Parallel()
		svc := &mockNatsService{streamInfo: &entities.StreamInfo{}}
		handler := New(svc, svc, svc, svc, svc)

		_, err := handler.UpdateStream(t.Context(), connect.NewRequest(&managementpb.UpdateStreamRequest{
			ConnectionId:           "conn-1",
			StreamName:             "flags",
			SubjectDeleteMarkerTtl: durationpb.New(0),
		}))
		require.NoError(t, err)

		require.NotNil(t, svc.updatedStream.SubjectDeleteMarkerTTL)
		assert.Zero(t, *svc.updatedStream.SubjectDeleteMarkerTTL)
	})
}
