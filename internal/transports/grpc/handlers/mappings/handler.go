// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package mappings

import (
	"context"
	"net/http"

	"connectrpc.com/connect"

	"github.com/altessa-s/go-atlas/core/collections/slices"
	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/transports/grpc/helpers"

	mappingssvc "github.com/dmit-4884/natscope/internal/services/mappings"
	protosvc "github.com/dmit-4884/natscope/internal/services/proto"
	mappingspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/mappings/v1/mappings"
	mappingsconnect "github.com/dmit-4884/natscope/proto/gen/services/grpc/mappings/v1/mappings/grpc_mappingsconnect"
	mappingstypespb "github.com/dmit-4884/natscope/proto/gen/types/mappings"
)

// Handler implements the Connect MappingsServiceHandler interface.
type Handler struct {
	mappingsService mappingssvc.Service
	protoService    protosvc.Registry
}

// New creates a new MappingsService handler.
func New(mappingsService mappingssvc.Service, protoService protosvc.Registry) *Handler {
	return &Handler{
		mappingsService: mappingsService,
		protoService:    protoService,
	}
}

// HTTPHandler returns the Connect route + handler, registering
// StatusErrorConvert as an interceptor.
func (h *Handler) HTTPHandler(opts ...connect.HandlerOption) (string, http.Handler) {
	opts = append(opts, connect.WithInterceptors(
		grpchelpers.NewHandlerErrorInterceptor(h.StatusErrorConvert),
	))
	return mappingsconnect.NewMappingsServiceHandler(h, opts...)
}

// CreateMapping creates a new subject mapping.
func (h *Handler) CreateMapping(
	ctx context.Context,
	req *connect.Request[mappingspb.CreateMappingRequest],
) (*connect.Response[mappingspb.CreateMappingResponse], error) {
	createReq := converter.Convert(req.Msg, &entities.SubjectMappingCreate{}, converter.WithIgnoreFields("Framing"))
	createReq.Framing = grpchelpers.FramingFromProto(req.Msg.Framing)

	mapping, err := h.mappingsService.Create(ctx, createReq)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&mappingspb.CreateMappingResponse{Mapping: mappingToProto(mapping)}), nil
}

// UpdateMapping changes the fields set in the request.
func (h *Handler) UpdateMapping(
	ctx context.Context,
	req *connect.Request[mappingspb.UpdateMappingRequest],
) (*connect.Response[mappingspb.UpdateMappingResponse], error) {
	updateReq := converter.Convert(req.Msg, &entities.SubjectMappingUpdate{}, converter.WithIgnoreFields("Framing"))
	if req.Msg.Framing != nil {
		framing := grpchelpers.FramingFromProto(req.Msg.Framing)
		updateReq.Framing = &framing
	}

	mapping, err := h.mappingsService.Update(ctx, updateReq)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&mappingspb.UpdateMappingResponse{Mapping: mappingToProto(mapping)}), nil
}

func mappingToProto(m *entities.SubjectMapping) *mappingstypespb.SubjectMapping {
	pb := converter.Convert(m, &mappingstypespb.SubjectMapping{},
		converter.WithHandleEmbeddedStructs(true), grpchelpers.ProtoCodecs, converter.WithIgnoreFields("Framing"))
	pb.Framing = grpchelpers.FramingToProto(m.Framing)
	return pb
}

// ListMappings returns subject mappings with AIP-158 pagination (page_token
// maps onto the storage cursor, page_size onto the limit).
func (h *Handler) ListMappings(
	ctx context.Context,
	req *connect.Request[mappingspb.ListMappingsRequest],
) (*connect.Response[mappingspb.ListMappingsResponse], error) {
	in := req.Msg
	listReq := &entities.SubjectMappingsList{}
	if ps := in.GetPageSize(); ps > 0 {
		listReq.Limit = new(int64(ps))
	}
	listReq.Cursor = in.GetPageToken()
	listReq.IncludeTotalCount = in.GetIncludeTotalCount()

	list, err := h.mappingsService.List(ctx, listReq)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&mappingspb.ListMappingsResponse{
		Mappings:      slices.To(list.Items, mappingToProto),
		NextPageToken: list.NextCursor,
		TotalSize:     list.Total,
	}), nil
}

// DeleteMapping deletes a subject mapping.
func (h *Handler) DeleteMapping(
	ctx context.Context,
	req *connect.Request[mappingspb.DeleteMappingRequest],
) (*connect.Response[mappingspb.DeleteMappingResponse], error) {
	if err := h.mappingsService.Delete(ctx, req.Msg.Id); err != nil {
		return nil, err
	}
	return connect.NewResponse(&mappingspb.DeleteMappingResponse{}), nil
}

// BatchSaveMappings creates or updates multiple mappings at once.
func (h *Handler) BatchSaveMappings(
	ctx context.Context,
	req *connect.Request[mappingspb.BatchSaveMappingsRequest],
) (*connect.Response[mappingspb.BatchSaveMappingsResponse], error) {
	in := req.Msg
	mappingEntities := slices.To(in.Mappings, func(m *mappingspb.MappingBulkItem) *entities.SubjectMapping {
		mapping := converter.Convert(m, entities.SubjectMappingNew(), converter.WithIgnoreFields("Framing"))
		mapping.Framing = grpchelpers.FramingFromProto(m.Framing)
		return mapping
	})

	result, err := h.mappingsService.BulkSave(ctx, mappingEntities)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&mappingspb.BatchSaveMappingsResponse{
		Created: int32(result.Created),
		Updated: int32(result.Updated),
		Deleted: int32(result.Deleted),
	}), nil
}

// BatchCheckMappingHealth computes health states for the given mapping ids.
func (h *Handler) BatchCheckMappingHealth(
	ctx context.Context,
	req *connect.Request[mappingspb.BatchCheckMappingHealthRequest],
) (*connect.Response[mappingspb.BatchCheckMappingHealthResponse], error) {
	in := req.Msg
	if len(in.Ids) == 0 {
		return connect.NewResponse(&mappingspb.BatchCheckMappingHealthResponse{}), nil
	}

	healths, err := h.protoService.MappingHealth(ctx, in.Ids)
	if err != nil {
		return nil, err
	}

	items := slices.To(healths, func(hh entities.SubjectMappingHealth) *mappingspb.MappingHealthItem {
		return converter.Convert(hh, &mappingspb.MappingHealthItem{})
	})
	return connect.NewResponse(&mappingspb.BatchCheckMappingHealthResponse{Items: items}), nil
}
