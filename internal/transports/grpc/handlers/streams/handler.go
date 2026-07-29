// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package streams

import (
	"context"
	"net/http"

	"connectrpc.com/connect"

	"github.com/altessa-s/go-atlas/core/collections/slices"
	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/transports/grpc/helpers"

	natssvc "github.com/dmit-4884/natscope/internal/services/nats"
	streamspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/streams"
	streamsconnect "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/streams/grpc_nats_streamsconnect"
	natspb "github.com/dmit-4884/natscope/proto/gen/types/nats"
)

// Handler implements the Connect StreamsServiceHandler interface.
type Handler struct {
	natsService natssvc.StreamReader
}

// New creates a new StreamsService handler.
func New(natsService natssvc.StreamReader) *Handler {
	return &Handler{natsService: natsService}
}

// HTTPHandler returns the Connect route + handler; uses the shared error
// interceptor.
func (h *Handler) HTTPHandler(opts ...connect.HandlerOption) (string, http.Handler) {
	opts = append(opts, connect.WithInterceptors(
		grpchelpers.NewHandlerErrorInterceptor(nil),
	))
	return streamsconnect.NewStreamsServiceHandler(h, opts...)
}

// ListStreams returns all streams for a connection.
func (h *Handler) ListStreams(
	ctx context.Context,
	req *connect.Request[streamspb.ListStreamsRequest],
) (*connect.Response[streamspb.ListStreamsResponse], error) {
	streamList, err := h.natsService.ListStreams(ctx, req.Msg.ConnectionId)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&streamspb.ListStreamsResponse{
		Streams: slices.To(streamList, func(s entities.StreamInfo) *natspb.StreamInfo {
			return converter.Convert(&s, &natspb.StreamInfo{}, grpchelpers.ProtoCodecs)
		}),
	}), nil
}

// GetStream returns detailed information for a specific stream.
func (h *Handler) GetStream(
	ctx context.Context,
	req *connect.Request[streamspb.GetStreamRequest],
) (*connect.Response[streamspb.GetStreamResponse], error) {
	stream, err := h.natsService.GetStreamInfo(ctx, req.Msg.ConnectionId, req.Msg.StreamName)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&streamspb.GetStreamResponse{
		Stream: converter.Convert(stream, &natspb.StreamInfo{}, grpchelpers.ProtoCodecs),
	}), nil
}
