// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package management

import (
	"context"
	"time"

	"connectrpc.com/connect"

	"github.com/altessa-s/go-atlas/core/collections/slices"
	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"

	"google.golang.org/protobuf/types/known/durationpb"

	managementpb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/management"
	natspb "github.com/dmit-4884/natscope/proto/gen/types/nats"
)

// ListConsumers returns all consumers for a stream.
func (h *Handler) ListConsumers(
	ctx context.Context,
	req *connect.Request[managementpb.ListConsumersRequest],
) (*connect.Response[managementpb.ListConsumersResponse], error) {
	in := req.Msg
	consumers, err := h.natsService.GetStreamConsumers(ctx, in.GetConnectionId(), in.GetStreamName())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&managementpb.ListConsumersResponse{
		Consumers: slices.To(consumers, func(c entities.ConsumerInfo) *natspb.ConsumerInfo {
			return converter.Convert(&c, &natspb.ConsumerInfo{}, protoCodecs)
		}),
	}), nil
}

// CreateConsumer creates a new consumer on a stream.
func (h *Handler) CreateConsumer(
	ctx context.Context,
	req *connect.Request[managementpb.CreateConsumerRequest],
) (*connect.Response[managementpb.CreateConsumerResponse], error) {
	in := req.Msg
	cr := converter.Convert(in, &entities.ConsumerCreateRequest{}, protoCodecs)
	consumer, err := h.natsService.CreateConsumer(ctx, in.GetConnectionId(), in.GetStreamName(), *cr)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&managementpb.CreateConsumerResponse{
		Consumer: converter.Convert(consumer, &natspb.ConsumerInfo{}, protoCodecs),
	}), nil
}

// UpdateConsumer updates an existing consumer.
func (h *Handler) UpdateConsumer(
	ctx context.Context,
	req *connect.Request[managementpb.UpdateConsumerRequest],
) (*connect.Response[managementpb.UpdateConsumerResponse], error) {
	in := req.Msg
	ur := converter.Convert(in, &entities.ConsumerUpdateRequest{}, protoCodecs)
	consumer, err := h.natsService.UpdateConsumer(ctx, in.GetConnectionId(), in.GetStreamName(), in.GetConsumerName(), *ur)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&managementpb.UpdateConsumerResponse{
		Consumer: converter.Convert(consumer, &natspb.ConsumerInfo{}, protoCodecs),
	}), nil
}

// DeleteConsumer deletes a consumer from a stream.
func (h *Handler) DeleteConsumer(
	ctx context.Context,
	req *connect.Request[managementpb.DeleteConsumerRequest],
) (*connect.Response[managementpb.DeleteConsumerResponse], error) {
	in := req.Msg
	if err := h.natsService.DeleteConsumer(
		ctx,
		in.GetConnectionId(),
		in.GetStreamName(),
		in.GetConsumerName(),
	); err != nil {
		return nil, err
	}
	return connect.NewResponse(&managementpb.DeleteConsumerResponse{}), nil
}

// PauseConsumer pauses a consumer until a specified RFC3339 timestamp.
func (h *Handler) PauseConsumer(
	ctx context.Context,
	req *connect.Request[managementpb.PauseConsumerRequest],
) (*connect.Response[managementpb.PauseConsumerResponse], error) {
	in := req.Msg
	resp, err := h.natsService.PauseConsumer(
		ctx,
		in.GetConnectionId(),
		in.GetStreamName(),
		in.GetConsumerName(),
		in.GetPauseUntil(),
	)
	if err != nil {
		return nil, err
	}

	pbResp := &managementpb.PauseConsumerResponse{
		Paused:         resp.Paused,
		PauseRemaining: durationpb.New(resp.PauseRemaining),
	}
	if resp.PauseUntil != nil {
		s := resp.PauseUntil.Format(time.RFC3339)
		pbResp.PauseUntil = &s
	}
	return connect.NewResponse(pbResp), nil
}

// ResumeConsumer resumes a paused consumer.
func (h *Handler) ResumeConsumer(
	ctx context.Context,
	req *connect.Request[managementpb.ResumeConsumerRequest],
) (*connect.Response[managementpb.ResumeConsumerResponse], error) {
	in := req.Msg
	if err := h.natsService.ResumeConsumer(
		ctx,
		in.GetConnectionId(),
		in.GetStreamName(),
		in.GetConsumerName(),
	); err != nil {
		return nil, err
	}
	return connect.NewResponse(&managementpb.ResumeConsumerResponse{Paused: false}), nil
}
