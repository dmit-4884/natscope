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

	managementpb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/management"
)

// consumerActionsService records the consumer actions the handler forwards.
type consumerActionsService struct {
	mockNatsService

	resetStream   string
	resetConsumer string
	resetSequence *uint64
	resetResp     *entities.ConsumerResetResponse

	unpinStream   string
	unpinConsumer string
	unpinGroup    string

	createdConsumer entities.ConsumerCreateRequest
	updatedConsumer entities.ConsumerUpdateRequest

	actionErr error
}

func (m *consumerActionsService) UpdateConsumer(
	_ context.Context,
	_ string,
	_ string,
	_ string,
	req entities.ConsumerUpdateRequest,
) (*entities.ConsumerInfo, error) {
	m.updatedConsumer = req
	return m.consumerInfo, m.actionErr
}

func (m *consumerActionsService) UnpinConsumer(
	_ context.Context,
	_ string,
	streamName string,
	consumerName string,
	group string,
) error {
	m.unpinStream, m.unpinConsumer, m.unpinGroup = streamName, consumerName, group
	return m.actionErr
}

func (m *consumerActionsService) CreateConsumer(
	_ context.Context,
	_ string,
	_ string,
	req entities.ConsumerCreateRequest,
) (*entities.ConsumerInfo, error) {
	m.createdConsumer = req
	return m.consumerInfo, m.actionErr
}

func (m *consumerActionsService) ResetConsumer(
	_ context.Context,
	_ string,
	streamName string,
	consumerName string,
	sequence *uint64,
) (*entities.ConsumerResetResponse, error) {
	m.resetStream, m.resetConsumer, m.resetSequence = streamName, consumerName, sequence
	return m.resetResp, m.actionErr
}

func TestHandler_ResetConsumer(t *testing.T) {
	t.Parallel()

	t.Run("without sequence", func(t *testing.T) {
		t.Parallel()
		svc := &consumerActionsService{resetResp: &entities.ConsumerResetResponse{
			Consumer: &entities.ConsumerInfo{Name: "worker", Stream: "ORDERS", NumPending: 7},
			ResetSeq: 0,
		}}
		handler := New(svc, svc, svc, svc, svc)

		resp, err := handler.ResetConsumer(t.Context(), connect.NewRequest(&managementpb.ResetConsumerRequest{
			ConnectionId: "conn-1", StreamName: "ORDERS", ConsumerName: "worker",
		}))
		require.NoError(t, err)
		assert.Equal(t, "ORDERS", svc.resetStream)
		assert.Equal(t, "worker", svc.resetConsumer)
		assert.Nil(t, svc.resetSequence)
		assert.Equal(t, "worker", resp.Msg.GetConsumer().GetName())
		assert.Equal(t, uint64(7), resp.Msg.GetConsumer().GetNumPending())
		assert.Zero(t, resp.Msg.GetResetSeq())
	})

	t.Run("to a sequence", func(t *testing.T) {
		t.Parallel()
		svc := &consumerActionsService{resetResp: &entities.ConsumerResetResponse{
			Consumer: &entities.ConsumerInfo{Name: "worker"},
			ResetSeq: 42,
		}}
		handler := New(svc, svc, svc, svc, svc)

		resp, err := handler.ResetConsumer(t.Context(), connect.NewRequest(&managementpb.ResetConsumerRequest{
			ConnectionId: "conn-1", StreamName: "ORDERS", ConsumerName: "worker", Sequence: new(uint64(42)),
		}))
		require.NoError(t, err)
		require.NotNil(t, svc.resetSequence)
		assert.Equal(t, uint64(42), *svc.resetSequence)
		assert.Equal(t, uint64(42), resp.Msg.GetResetSeq())
	})

	t.Run("unsupported server", func(t *testing.T) {
		t.Parallel()
		svc := &consumerActionsService{actionErr: &errs.FeatureUnsupportedError{
			Feature: "consumer reset", MinVersion: "2.14", ServerVersion: "2.12.3",
		}}
		handler := New(svc, svc, svc, svc, svc)

		_, err := handler.ResetConsumer(t.Context(), connect.NewRequest(&managementpb.ResetConsumerRequest{
			ConnectionId: "conn-1", StreamName: "ORDERS", ConsumerName: "worker",
		}))
		assert.ErrorIs(t, err, errs.ErrFeatureUnsupported)
	})
}

func TestHandler_UnpinConsumer(t *testing.T) {
	t.Parallel()

	t.Run("forwards the group", func(t *testing.T) {
		t.Parallel()
		svc := &consumerActionsService{}
		handler := New(svc, svc, svc, svc, svc)

		_, err := handler.UnpinConsumer(t.Context(), connect.NewRequest(&managementpb.UnpinConsumerRequest{
			ConnectionId: "conn-1", StreamName: "ORDERS", ConsumerName: "worker", Group: "jobs",
		}))
		require.NoError(t, err)
		assert.Equal(t, "ORDERS", svc.unpinStream)
		assert.Equal(t, "worker", svc.unpinConsumer)
		assert.Equal(t, "jobs", svc.unpinGroup)
	})

	t.Run("service error", func(t *testing.T) {
		t.Parallel()
		svc := &consumerActionsService{actionErr: errs.ErrConsumerNotFound}
		handler := New(svc, svc, svc, svc, svc)

		_, err := handler.UnpinConsumer(t.Context(), connect.NewRequest(&managementpb.UnpinConsumerRequest{
			ConnectionId: "conn-1", StreamName: "ORDERS", ConsumerName: "worker", Group: "jobs",
		}))
		assert.ErrorIs(t, err, errs.ErrConsumerNotFound)
	})
}

func TestHandler_ConsumerPriorityGroups(t *testing.T) {
	t.Parallel()

	pinnedAt := time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC)
	svc := &consumerActionsService{}
	svc.consumerInfo = &entities.ConsumerInfo{
		Name: "worker",
		Config: &entities.ConsumerConfig{
			PriorityPolicy: entities.PriorityPinnedClient,
			PriorityGroups: []string{"jobs"},
			PinnedTTL:      time.Minute,
		},
		PriorityGroups: []entities.PriorityGroupState{{Group: "jobs", PinnedClientID: "pin-1", PinnedTS: pinnedAt}},
	}
	handler := New(svc, svc, svc, svc, svc)

	resp, err := handler.CreateConsumer(t.Context(), connect.NewRequest(&managementpb.CreateConsumerRequest{
		ConnectionId:   "conn-1",
		StreamName:     "ORDERS",
		Name:           "worker",
		PriorityPolicy: 1,
		PriorityGroups: []string{"jobs"},
		PinnedTtl:      durationpb.New(time.Minute),
	}))
	require.NoError(t, err)

	assert.Equal(t, entities.PriorityPinnedClient, svc.createdConsumer.PriorityPolicy)
	assert.Equal(t, []string{"jobs"}, svc.createdConsumer.PriorityGroups)
	assert.Equal(t, time.Minute, svc.createdConsumer.PinnedTTL)

	out := resp.Msg.GetConsumer()
	assert.Equal(t, int32(1), out.GetConfig().GetPriorityPolicy())
	assert.Equal(t, []string{"jobs"}, out.GetConfig().GetPriorityGroups())
	assert.Equal(t, time.Minute, out.GetConfig().GetPinnedTtl().AsDuration())
	require.Len(t, out.GetPriorityGroups(), 1)
	assert.Equal(t, "pin-1", out.GetPriorityGroups()[0].GetPinnedClientId())
	assert.Equal(t, pinnedAt, out.GetPriorityGroups()[0].GetPinnedTs().AsTime())
}

func TestHandler_UpdateConsumerPriority(t *testing.T) {
	t.Parallel()

	t.Run("sets priority fields", func(t *testing.T) {
		t.Parallel()
		svc := &consumerActionsService{}
		svc.consumerInfo = &entities.ConsumerInfo{Name: "worker"}
		handler := New(svc, svc, svc, svc, svc)

		_, err := handler.UpdateConsumer(t.Context(), connect.NewRequest(&managementpb.UpdateConsumerRequest{
			ConnectionId: "conn-1", StreamName: "ORDERS", ConsumerName: "worker",
			PriorityPolicy: new(int32(2)), PriorityGroups: []string{"a"}, PinnedTtl: durationpb.New(time.Minute),
		}))
		require.NoError(t, err)

		got := svc.updatedConsumer
		require.NotNil(t, got.PriorityPolicy)
		assert.Equal(t, entities.PriorityOverflow, *got.PriorityPolicy)
		assert.Equal(t, []string{"a"}, got.PriorityGroups)
		require.NotNil(t, got.PinnedTTL)
		assert.Equal(t, time.Minute, *got.PinnedTTL)
	})

	t.Run("absent priority fields stay nil", func(t *testing.T) {
		t.Parallel()
		svc := &consumerActionsService{}
		svc.consumerInfo = &entities.ConsumerInfo{Name: "worker"}
		handler := New(svc, svc, svc, svc, svc)

		_, err := handler.UpdateConsumer(t.Context(), connect.NewRequest(&managementpb.UpdateConsumerRequest{
			ConnectionId: "conn-1", StreamName: "ORDERS", ConsumerName: "worker", Description: new("d"),
		}))
		require.NoError(t, err)

		got := svc.updatedConsumer
		assert.Nil(t, got.PriorityPolicy)
		assert.Nil(t, got.PriorityGroups)
		assert.Nil(t, got.PinnedTTL)
	})
}
