// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package publish

import (
	"context"
	"encoding/base64"
	"net/http"

	"connectrpc.com/connect"

	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/transports/grpc/helpers"

	protosvc "github.com/dmit-4884/natscope/internal/services/proto"
	publishsvc "github.com/dmit-4884/natscope/internal/services/publish"
	publishpb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/publish"
	publishconnect "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/publish/grpc_nats_publishconnect"
	protopb "github.com/dmit-4884/natscope/proto/gen/types/proto"
)

// Handler implements the Connect PublishServiceHandler interface.
type Handler struct {
	publishService publishsvc.Service
	protoService   protosvc.Codec
}

// New creates a new PublishService handler.
func New(publishService publishsvc.Service, protoService protosvc.Codec) *Handler {
	return &Handler{
		publishService: publishService,
		protoService:   protoService,
	}
}

// HTTPHandler returns the Connect route + handler; uses the shared error
// interceptor.
func (h *Handler) HTTPHandler(opts ...connect.HandlerOption) (string, http.Handler) {
	opts = append(opts, connect.WithInterceptors(
		grpchelpers.NewHandlerErrorInterceptor(nil),
	))
	return publishconnect.NewPublishServiceHandler(h, opts...)
}

// PublishMessage publishes to JetStream; encode/publish failures return via
// Response.Error, never as transport errors.
func (h *Handler) PublishMessage(
	ctx context.Context,
	req *connect.Request[publishpb.PublishMessageRequest],
) (*connect.Response[publishpb.PublishMessageResponse], error) {
	in := req.Msg
	pr := converter.Convert(in, &entities.PublishRequest{})
	result, err := h.publishService.Publish(ctx, pr)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&publishpb.PublishMessageResponse{
		Stream:    result.Stream,
		Sequence:  result.Sequence,
		Duplicate: result.Duplicate,
		Error:     result.Error,
	}), nil
}

// EncodeMessage encodes a JSON message to protobuf (for validation/preview).
func (h *Handler) EncodeMessage(
	ctx context.Context,
	req *connect.Request[publishpb.EncodeMessageRequest],
) (*connect.Response[publishpb.EncodeMessageResponse], error) {
	in := req.Msg
	cr := entities.CodecRequest{
		JSON:        []byte(in.Data),
		SourceID:    in.SourceId,
		MessageType: in.MessageType,
	}
	if in.SourceTag != nil {
		cr.Tag = *in.SourceTag
	}
	result, err := h.protoService.Encode(ctx, cr)
	if err != nil {
		return nil, err
	}

	resp := &publishpb.EncodeMessageResponse{
		DataSize: int64(result.DataSize),
	}
	if result.DataBase64 != "" {
		data, decErr := base64.StdEncoding.DecodeString(result.DataBase64)
		if decErr == nil {
			resp.Data = data
		}
	}
	if result.Error != "" {
		resp.Error = &result.Error
	}
	return connect.NewResponse(resp), nil
}

// ValidateJson validates JSON data against protobuf validation rules.
func (h *Handler) ValidateJson(
	ctx context.Context,
	req *connect.Request[publishpb.ValidateJsonRequest],
) (*connect.Response[publishpb.ValidateJsonResponse], error) {
	in := req.Msg
	cr := entities.CodecRequest{
		JSON:        []byte(in.Data),
		SourceID:    in.SourceId,
		MessageType: in.MessageType,
	}
	if in.SourceTag != nil {
		cr.Tag = *in.SourceTag
	}
	result, err := h.protoService.ValidateJSON(ctx, cr)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&publishpb.ValidateJsonResponse{
		Result: converter.Convert(result, &protopb.ValidationResult{}),
	}), nil
}
