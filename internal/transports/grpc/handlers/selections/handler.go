// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package selections

import (
	"context"
	"net/http"

	"connectrpc.com/connect"

	"github.com/altessa-s/go-atlas/core/collections/slices"
	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/transports/grpc/helpers"

	protosvc "github.com/dmit-4884/natscope/internal/services/proto"
	selectionspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/proto/v1/selections"
	selectionsconnect "github.com/dmit-4884/natscope/proto/gen/services/grpc/proto/v1/selections/grpc_proto_selectionsconnect"
	protopb "github.com/dmit-4884/natscope/proto/gen/types/proto"
)

// protoDeps is the narrow proto surface the selections handler needs:
// selection CRUD plus registry stats.
type protoDeps interface {
	protosvc.SelectionManager
	protosvc.Registry
}

// Handler implements the Connect SelectionsServiceHandler interface.
type Handler struct {
	protoService protoDeps
}

// New creates a new SelectionsService handler. The proto roles are taken
// separately so fx can resolve each, then combined into the narrow protoDeps
// surface used internally.
func New(selectionManager protosvc.SelectionManager, registry protosvc.Registry) *Handler {
	return &Handler{protoService: struct {
		protosvc.SelectionManager
		protosvc.Registry
	}{selectionManager, registry}}
}

// HTTPHandler returns the Connect route + handler, registering
// StatusErrorConvert as an interceptor.
func (h *Handler) HTTPHandler(opts ...connect.HandlerOption) (string, http.Handler) {
	opts = append(opts, connect.WithInterceptors(
		grpchelpers.NewHandlerErrorInterceptor(h.StatusErrorConvert),
	))
	return selectionsconnect.NewSelectionsServiceHandler(h, opts...)
}

// ListSelections returns all proto selections, optionally filtered by source.
func (h *Handler) ListSelections(
	ctx context.Context,
	req *connect.Request[selectionspb.ListSelectionsRequest],
) (*connect.Response[selectionspb.ListSelectionsResponse], error) {
	selections, err := h.protoService.ListSelections(ctx)
	if err != nil {
		return nil, err
	}
	if sourceID := req.Msg.GetSourceId(); sourceID != "" {
		selections = slices.ToWithFilter(
			selections,
			func(s *entities.ProtoSelection) bool { return s.SourceID == sourceID },
			func(s *entities.ProtoSelection) *entities.ProtoSelection { return s },
		)
	}
	return connect.NewResponse(&selectionspb.ListSelectionsResponse{
		Selections: slices.To(selections, func(s *entities.ProtoSelection) *protopb.ProtoSelection {
			return converter.Convert(
				s,
				&protopb.ProtoSelection{},
				converter.WithHandleEmbeddedStructs(true),
				grpchelpers.ProtoCodecs,
			)
		}),
	}), nil
}

// SelectVersion selects a proto version.
func (h *Handler) SelectVersion(
	ctx context.Context,
	req *connect.Request[selectionspb.SelectVersionRequest],
) (*connect.Response[selectionspb.SelectVersionResponse], error) {
	createReq := &entities.ProtoSelectionCreate{
		SourceID: req.Msg.SourceId,
		Tag:      req.Msg.Tag,
	}

	selection, err := h.protoService.Select(ctx, createReq)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&selectionspb.SelectVersionResponse{
		Selection: converter.Convert(
			selection,
			&protopb.ProtoSelection{},
			converter.WithHandleEmbeddedStructs(true),
			grpchelpers.ProtoCodecs,
		),
	}), nil
}

// DeleteSelection removes a proto selection.
func (h *Handler) DeleteSelection(
	ctx context.Context,
	req *connect.Request[selectionspb.DeleteSelectionRequest],
) (*connect.Response[selectionspb.DeleteSelectionResponse], error) {
	if err := h.protoService.DeleteSelection(ctx, req.Msg.Id); err != nil {
		return nil, err
	}
	return connect.NewResponse(&selectionspb.DeleteSelectionResponse{}), nil
}

// LoadSelections fetches and compiles every stored selection.
func (h *Handler) LoadSelections(
	ctx context.Context,
	_ *connect.Request[selectionspb.LoadSelectionsRequest],
) (*connect.Response[selectionspb.LoadSelectionsResponse], error) {
	result, err := h.protoService.LoadAllSelections(ctx)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&selectionspb.LoadSelectionsResponse{
		MessageCount: int32(result.MessageCount),
	}), nil
}

// GetSelectionStatus returns the current status of loaded proto files.
func (h *Handler) GetSelectionStatus(
	ctx context.Context,
	_ *connect.Request[selectionspb.GetSelectionStatusRequest],
) (*connect.Response[selectionspb.GetSelectionStatusResponse], error) {
	stats := h.protoService.Stats(ctx)
	return connect.NewResponse(&selectionspb.GetSelectionStatusResponse{
		Loaded:       stats.IsLoaded(),
		MessageCount: int32(stats.MessagesCount),
	}), nil
}
