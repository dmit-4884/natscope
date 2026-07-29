// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package live is the thin Connect adapter for the live subscription service;
// session logic lives in internal/services/live.
package live

import (
	"context"
	"net/http"
	"time"

	"connectrpc.com/connect"

	"github.com/altessa-s/go-atlas/core/collections/slices"
	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/pkg/convcodecs"
	"github.com/dmit-4884/natscope/internal/transports/grpc/helpers"

	"google.golang.org/protobuf/types/known/timestamppb"

	livesvc "github.com/dmit-4884/natscope/internal/services/live"
	livepb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/live"
	liveconnect "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/live/grpc_nats_liveconnect"
	natspb "github.com/dmit-4884/natscope/proto/gen/types/nats"
)

// liveMessageOpts converts entities.NatsMessage to natspb.NatsMessage.
// WithIgnoreZeroValues keeps empty Stream nil: codecs bypass the primitive registry.
var liveMessageOpts = []converter.Option{
	grpchelpers.ProtoCodecs,
	converter.WithCodecs(convcodecs.BytesBase64, convcodecs.StringSliceFirst),
	converter.WithFieldMappings(map[string]string{
		"Data":   "DataBase64",
		"Header": "Headers",
	}),
	converter.WithIgnoreZeroValues(),
}

// Handler is the Connect adapter for the live service.
type Handler struct {
	service livesvc.Service
}

// New creates a new live handler.
func New(service livesvc.Service) *Handler {
	return &Handler{service: service}
}

// HTTPHandler returns the Connect route + handler; live errors use the shared
// interceptor.
func (h *Handler) HTTPHandler(opts ...connect.HandlerOption) (string, http.Handler) {
	opts = append(opts, connect.WithInterceptors(
		grpchelpers.NewHandlerErrorInterceptor(nil),
	))
	return liveconnect.NewLiveServiceHandler(h, opts...)
}

// Subscribe is the server-streaming endpoint: hands the session to the live
// service and converts each emitted event back to proto for the stream.
func (h *Handler) Subscribe(
	ctx context.Context,
	req *connect.Request[livepb.SubscribeRequest],
	stream *connect.ServerStream[livepb.LiveEvent],
) error {
	entReq := converter.Convert(req.Msg, &entities.LiveSubscribeRequest{})
	return h.service.Subscribe(ctx, entReq, func(ev *entities.LiveEvent) error {
		return stream.Send(toProtoLiveEvent(ev))
	})
}

// toProtoLiveEvent expands the entities.LiveEvent union into the proto oneof.
func toProtoLiveEvent(ev *entities.LiveEvent) *livepb.LiveEvent {
	out := &livepb.LiveEvent{}
	switch {
	case ev.Batch != nil:
		out.Event = &livepb.LiveEvent_Batch{
			Batch: &livepb.LiveMessageBatch{
				Messages: slices.To(ev.Batch.Messages, toProtoLiveBatchMessage),
			},
		}
	case ev.Stats != nil:
		out.Event = &livepb.LiveEvent_Stats{
			Stats: &livepb.LiveStats{
				TotalMessages:     ev.Stats.TotalMessages,
				MessagesPerSecond: ev.Stats.MessagesPerSecond,
				SubjectCounts:     ev.Stats.SubjectCounts,
				MessagesDropped:   ev.Stats.MessagesDropped,
			},
		}
	case ev.Error != nil:
		out.Event = &livepb.LiveEvent_Error{
			Error: &livepb.LiveError{
				Code:    ev.Error.Code,
				Message: ev.Error.Message,
			},
		}
	case ev.ProtoReload != nil:
		out.Event = &livepb.LiveEvent_ProtoReload{
			ProtoReload: &livepb.LiveProtoReload{
				MessagesCount: ev.ProtoReload.MessagesCount,
			},
		}
	}
	return out
}

// toProtoLiveMessage converts a raw NatsMessage to proto; kept distinct from
// the batch variant for isolated testing.
func toProtoLiveMessage(msg *entities.NatsMessage) *natspb.NatsMessage {
	pb := converter.Convert(msg, &natspb.NatsMessage{}, liveMessageOpts...)
	pb.DataSize = int32(len(msg.Data))
	pb.ContentType = string(msg.DetectContentType())
	if pb.Timestamp == nil {
		pb.Timestamp = timestamppb.New(time.Now())
	}
	return pb
}

// toProtoLiveBatchMessage layers the decoded fields onto toProtoLiveMessage.
func toProtoLiveBatchMessage(m *entities.LiveMessage) *natspb.NatsMessage {
	pb := toProtoLiveMessage(&m.NatsMessage)
	pb.Decoded = m.Decoded
	pb.DecodedType = m.DecodedType
	pb.DecodeError = m.DecodeError
	pb.Truncated = m.Truncated
	// Use pre-truncate size so the UI badge shows the real size, not the preview
	// size.
	if m.OriginalSize > 0 {
		pb.DataSize = int32(m.OriginalSize)
	}
	return pb
}
