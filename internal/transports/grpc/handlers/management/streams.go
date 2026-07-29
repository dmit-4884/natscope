// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package management

import (
	"context"

	"connectrpc.com/connect"

	"github.com/altessa-s/go-atlas/core/collections/slices"
	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"

	managementpb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/management"
	natspb "github.com/dmit-4884/natscope/proto/gen/types/nats"
)

// CreateStream creates a new JetStream stream.
func (h *Handler) CreateStream(
	ctx context.Context,
	req *connect.Request[managementpb.CreateStreamRequest],
) (*connect.Response[managementpb.CreateStreamResponse], error) {
	in := req.Msg
	cr := converter.Convert(in, &entities.StreamCreateRequest{},
		protoCodecs,
		converter.WithIgnoreFields("Placement", "Mirror", "Sources", "SubjectTransform", "Republish", "ConsumerLimits"),
	)

	if in.GetPlacement() != nil {
		cr.Placement = converter.Convert(in.GetPlacement(), &entities.Placement{})
	}
	if in.GetMirror() != nil {
		cr.Mirror = protoStreamSourceToEntity(in.GetMirror())
	}
	cr.Sources = slices.To(in.GetSources(), protoStreamSourceToEntity)
	if in.GetSubjectTransform() != nil {
		cr.SubjectTransform = converter.Convert(in.GetSubjectTransform(), &entities.SubjectTransformConfig{})
	}
	if in.GetRepublish() != nil {
		cr.Republish = converter.Convert(in.GetRepublish(), &entities.StreamRePublish{})
	}
	if in.GetConsumerLimits() != nil {
		cr.ConsumerLimits = converter.Convert(in.GetConsumerLimits(), &entities.StreamConsumerLimits{})
	}

	stream, err := h.natsService.CreateStream(ctx, in.GetConnectionId(), *cr)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&managementpb.CreateStreamResponse{
		Stream: converter.Convert(stream, &natspb.StreamInfo{}, protoCodecs),
	}), nil
}

// UpdateStream updates an existing JetStream stream.
func (h *Handler) UpdateStream(
	ctx context.Context,
	req *connect.Request[managementpb.UpdateStreamRequest],
) (*connect.Response[managementpb.UpdateStreamResponse], error) {
	in := req.Msg
	// MaxConsumers/Discard/Compression map natively via converter's primitive
	// registry (matching numeric layout); no manual unwrapping.
	ur := converter.Convert(in, &entities.StreamUpdateRequest{},
		protoCodecs,
		converter.WithIgnoreFields("Sources", "Republish", "SubjectTransform", "ConsumerLimits"),
	)

	if in.GetRepublish() != nil {
		ur.Republish = converter.Convert(in.GetRepublish(), &entities.StreamRePublish{})
	}
	if in.GetSubjectTransform() != nil {
		ur.SubjectTransform = converter.Convert(in.GetSubjectTransform(), &entities.SubjectTransformConfig{})
	}
	if in.GetConsumerLimits() != nil {
		ur.ConsumerLimits = converter.Convert(in.GetConsumerLimits(), &entities.StreamConsumerLimits{})
	}
	ur.Sources = slices.To(in.GetSources(), protoStreamSourceToEntity)

	stream, err := h.natsService.UpdateStream(ctx, in.GetConnectionId(), in.GetStreamName(), *ur)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&managementpb.UpdateStreamResponse{
		Stream: converter.Convert(stream, &natspb.StreamInfo{}, protoCodecs),
	}), nil
}

// DeleteStream deletes a JetStream stream.
func (h *Handler) DeleteStream(
	ctx context.Context,
	req *connect.Request[managementpb.DeleteStreamRequest],
) (*connect.Response[managementpb.DeleteStreamResponse], error) {
	in := req.Msg
	if err := h.natsService.DeleteStream(ctx, in.GetConnectionId(), in.GetStreamName()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&managementpb.DeleteStreamResponse{}), nil
}

// PurgeStream purges messages from a JetStream stream.
func (h *Handler) PurgeStream(
	ctx context.Context,
	req *connect.Request[managementpb.PurgeStreamRequest],
) (*connect.Response[managementpb.PurgeStreamResponse], error) {
	in := req.Msg
	pr := *converter.Convert(in, &entities.StreamPurgeRequest{})
	purged, err := h.natsService.PurgeStream(ctx, in.GetConnectionId(), in.GetStreamName(), pr)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&managementpb.PurgeStreamResponse{Purged: purged}), nil
}

// SealStream seals a JetStream stream (makes it read-only).
func (h *Handler) SealStream(
	ctx context.Context,
	req *connect.Request[managementpb.SealStreamRequest],
) (*connect.Response[managementpb.SealStreamResponse], error) {
	in := req.Msg
	stream, err := h.natsService.SealStream(ctx, in.GetConnectionId(), in.GetStreamName())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&managementpb.SealStreamResponse{
		Stream: converter.Convert(stream, &natspb.StreamInfo{}, protoCodecs),
	}), nil
}

// DeleteMessage deletes a single message from a stream by sequence number.
func (h *Handler) DeleteMessage(
	ctx context.Context,
	req *connect.Request[managementpb.DeleteMessageRequest],
) (*connect.Response[managementpb.DeleteMessageResponse], error) {
	in := req.Msg
	if err := h.natsService.DeleteMessage(
		ctx,
		in.GetConnectionId(),
		in.GetStreamName(),
		in.GetSequence(),
		in.GetSecure(),
	); err != nil {
		return nil, err
	}
	return connect.NewResponse(&managementpb.DeleteMessageResponse{}), nil
}
