// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package messages

import (
	"context"
	"net/http"

	"connectrpc.com/connect"

	"github.com/altessa-s/go-atlas/core/collections/slices"
	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/transports/grpc/helpers"

	messagessvc "github.com/dmit-4884/natscope/internal/services/messages"
	messagespb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/messages"
	messagesconnect "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/messages/grpc_nats_messagesconnect"
	natspb "github.com/dmit-4884/natscope/proto/gen/types/nats"
)

// Handler implements the Connect MessagesServiceHandler interface.
type Handler struct {
	service messagessvc.Service
}

// New creates a new MessagesService handler.
func New(service messagessvc.Service) *Handler {
	return &Handler{service: service}
}

// HTTPHandler returns the Connect route + handler; uses the shared error
// interceptor (no handler-private domain errors).
func (h *Handler) HTTPHandler(opts ...connect.HandlerOption) (string, http.Handler) {
	opts = append(opts, connect.WithInterceptors(
		grpchelpers.NewHandlerErrorInterceptor(nil),
	))
	return messagesconnect.NewMessagesServiceHandler(h, opts...)
}

// ListMessages returns messages from a stream.
func (h *Handler) ListMessages(
	ctx context.Context,
	req *connect.Request[messagespb.ListMessagesRequest],
) (*connect.Response[messagespb.ListMessagesResponse], error) {
	in := req.Msg
	// StartTime (*timestamppb.Timestamp <-> *time.Time) is handled by the tspb
	// codec; Direction is an enum -> domain string with no codec, mapped below.
	listReq := converter.Convert(in, &entities.MessageListRequest{},
		grpchelpers.ProtoCodecs, converter.WithIgnoreFields("Direction"))
	listReq.Direction = directionFromProto(in.Direction)

	resp, err := h.service.List(ctx, listReq)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&messagespb.ListMessagesResponse{
		HasMore:  resp.HasMore,
		NextSeq:  resp.NextSeq,
		Messages: slices.To(resp.Messages, messageToProto),
	}), nil
}

// GetMessage returns a single message by sequence number.
func (h *Handler) GetMessage(
	ctx context.Context,
	req *connect.Request[messagespb.GetMessageRequest],
) (*connect.Response[messagespb.GetMessageResponse], error) {
	msg, err := h.service.Get(ctx, converter.Convert(req.Msg, &entities.MessageGetRequest{}))
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&messagespb.GetMessageResponse{Message: messageToProto(msg)}), nil
}

// directionFromProto maps the Direction enum to the entity string; UNSPECIFIED
// becomes empty for the settings-driven default.
func directionFromProto(d messagespb.Direction) string {
	switch d {
	case messagespb.Direction_DIRECTION_FORWARD:
		return "forward"
	case messagespb.Direction_DIRECTION_BACKWARD:
		return "backward"
	case messagespb.Direction_DIRECTION_UNSPECIFIED:
		return ""
	default:
		return ""
	}
}

// messageToProto converts a Message to proto; Decoded ([]byte → *string) is set
// manually (no converter codec).
func messageToProto(m *entities.Message) *natspb.NatsMessage {
	pb := converter.Convert(m, &natspb.NatsMessage{}, grpchelpers.ProtoCodecs, converter.WithIgnoreFields("Decoded"))
	if m.Decoded != nil {
		decoded := string(m.Decoded)
		pb.Decoded = &decoded
	}
	return pb
}
