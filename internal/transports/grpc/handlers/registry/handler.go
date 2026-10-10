// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

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
	desc, err := h.protoService.DescribeType(ctx, req.Msg.SourceId, req.Msg.GetFingerprint(), req.Msg.FullName, req.Msg.IncludeReachable)
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
	example, err := h.protoService.GenerateExample(ctx, req.Msg.SourceId, req.Msg.GetFingerprint(), req.Msg.FullName)
	if err != nil {
		return nil, err
	}

	jsonBytes, err := json.Marshal(example)
	if err != nil {
		return nil, errors.Wrap(err, "marshal proto example")
	}
	return connect.NewResponse(&registrypb.GenerateExampleResponse{Json: string(jsonBytes)}), nil
}

// GetSchemaStatus counts the loaded message types and lists clashes between enabled sources.
func (h *Handler) GetSchemaStatus(
	ctx context.Context,
	_ *connect.Request[registrypb.GetSchemaStatusRequest],
) (*connect.Response[registrypb.GetSchemaStatusResponse], error) {
	status, err := h.protoService.SchemaStatus(ctx)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&registrypb.GetSchemaStatusResponse{
		MessageTypes: int32(status.MessageTypes), //nolint:gosec // bounded by the loaded schemas
		Conflicts:    slices.To(status.Conflicts, conflictToProto),
	}), nil
}

var conflictKinds = map[entities.ConflictKind]protopb.ConflictKind{
	entities.ConflictFileContent:    protopb.ConflictKind_CONFLICT_KIND_FILE_CONTENT,
	entities.ConflictSameShape:      protopb.ConflictKind_CONFLICT_KIND_SAME_SHAPE,
	entities.ConflictDifferentShape: protopb.ConflictKind_CONFLICT_KIND_DIFFERENT_SHAPE,
}

var conflictSeverities = map[entities.ConflictSeverity]protopb.ConflictSeverity{
	entities.SeverityInfo:  protopb.ConflictSeverity_CONFLICT_SEVERITY_INFO,
	entities.SeverityError: protopb.ConflictSeverity_CONFLICT_SEVERITY_ERROR,
}

func conflictToProto(c *entities.SchemaConflict) *protopb.SchemaConflict {
	pb := converter.Convert(c, &protopb.SchemaConflict{}, converter.WithIgnoreFields("Kind", "Severity"))
	pb.Kind, pb.Severity = conflictKinds[c.Kind], conflictSeverities[c.Severity]
	return pb
}
