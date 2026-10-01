// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"context"
	"encoding/json"
	"net/http"

	"connectrpc.com/connect"

	"github.com/altessa-s/go-atlas/core/collections/slices"
	"github.com/altessa-s/go-atlas/core/errors"
	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/transports/grpc/helpers"

	protosvc "github.com/dmit-4884/natscope/internal/services/proto"
	registrypb "github.com/dmit-4884/natscope/proto/gen/services/grpc/proto/v1/registry"
	registryconnect "github.com/dmit-4884/natscope/proto/gen/services/grpc/proto/v1/registry/grpc_proto_registryconnect"
	protopb "github.com/dmit-4884/natscope/proto/gen/types/proto"
)

// Handler implements the Connect RegistryServiceHandler interface.
type Handler struct {
	protoService protosvc.Registry
}

// New creates a new RegistryService handler.
func New(protoService protosvc.Registry) *Handler {
	return &Handler{protoService: protoService}
}

// HTTPHandler returns the Connect route + handler; uses the shared error
// interceptor.
func (h *Handler) HTTPHandler(opts ...connect.HandlerOption) (string, http.Handler) {
	opts = append(opts, connect.WithInterceptors(
		grpchelpers.NewHandlerErrorInterceptor(nil),
	))
	return registryconnect.NewRegistryServiceHandler(h, opts...)
}

// ListTypes lists the schema types of one source, or of every enabled source.
func (h *Handler) ListTypes(
	ctx context.Context,
	req *connect.Request[registrypb.ListTypesRequest],
) (*connect.Response[registrypb.ListTypesResponse], error) {
	types, err := h.protoService.ListTypes(ctx, req.Msg.GetSourceId())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&registrypb.ListTypesResponse{Types: slices.To(types, typeToProto)}), nil
}

// DescribeType describes a message, enum or service of a source.
func (h *Handler) DescribeType(
	ctx context.Context,
	req *connect.Request[registrypb.DescribeTypeRequest],
) (*connect.Response[registrypb.DescribeTypeResponse], error) {
	desc, err := h.protoService.DescribeType(ctx, req.Msg.SourceId, req.Msg.FullName, req.Msg.IncludeReachable)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(converter.Convert(desc, &registrypb.DescribeTypeResponse{})), nil
}

func typeToProto(t entities.SchemaType) *protopb.SchemaType {
	pb := converter.Convert(t, &protopb.SchemaType{}, converter.WithIgnoreFields("Kind"))
	switch t.Kind {
	case entities.SchemaTypeMessage:
		pb.Kind = protopb.SchemaTypeKind_SCHEMA_TYPE_KIND_MESSAGE
	case entities.SchemaTypeEnum:
		pb.Kind = protopb.SchemaTypeKind_SCHEMA_TYPE_KIND_ENUM
	case entities.SchemaTypeService:
		pb.Kind = protopb.SchemaTypeKind_SCHEMA_TYPE_KIND_SERVICE
	}
	return pb
}

// GenerateExample generates an example JSON for a message type within a source.
func (h *Handler) GenerateExample(
	ctx context.Context,
	req *connect.Request[registrypb.GenerateExampleRequest],
) (*connect.Response[registrypb.GenerateExampleResponse], error) {
	example, err := h.protoService.GenerateExample(ctx, req.Msg.SourceId, req.Msg.FullName)
	if err != nil {
		return nil, err
	}

	jsonBytes, err := json.Marshal(example)
	if err != nil {
		return nil, errors.Wrap(err, "marshal proto example")
	}
	return connect.NewResponse(&registrypb.GenerateExampleResponse{Json: string(jsonBytes)}), nil
}

// GetProtoStatus returns the current proto loading status.
func (h *Handler) GetProtoStatus(
	ctx context.Context,
	_ *connect.Request[registrypb.GetProtoStatusRequest],
) (*connect.Response[registrypb.GetProtoStatusResponse], error) {
	stats := h.protoService.Stats(ctx)
	if stats == nil {
		return connect.NewResponse(&registrypb.GetProtoStatusResponse{}), nil
	}
	return connect.NewResponse(&registrypb.GetProtoStatusResponse{
		Loaded:       stats.MessagesCount > 0,
		MessageCount: int32(stats.MessagesCount),
	}), nil
}
