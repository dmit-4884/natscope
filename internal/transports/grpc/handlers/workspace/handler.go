// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package workspace exposes the WorkspaceService Connect handler for workspace
// export/import.
package workspace

import (
	"context"
	"net/http"

	"connectrpc.com/connect"

	"github.com/altessa-s/go-atlas/core/collections/slices"
	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/transports/grpc/helpers"

	workspacesvc "github.com/dmit-4884/natscope/internal/services/workspace"
	workspacepb "github.com/dmit-4884/natscope/proto/gen/services/grpc/workspace/v1/workspace"
	workspaceconnect "github.com/dmit-4884/natscope/proto/gen/services/grpc/workspace/v1/workspace/grpc_workspaceconnect"
)

// Handler implements the Connect WorkspaceServiceHandler interface.
type Handler struct {
	service workspacesvc.Service
}

// New creates a new WorkspaceService handler.
func New(service workspacesvc.Service) *Handler {
	return &Handler{service: service}
}

// HTTPHandler returns the Connect route + handler, registering
// StatusErrorConvert as the per-handler error interceptor.
func (h *Handler) HTTPHandler(opts ...connect.HandlerOption) (string, http.Handler) {
	opts = append(opts, connect.WithInterceptors(
		grpchelpers.NewHandlerErrorInterceptor(h.StatusErrorConvert),
	))
	return workspaceconnect.NewWorkspaceServiceHandler(h, opts...)
}

// StatusErrorConvert maps workspace errors to Connect codes via the shared
// converter.
func (h *Handler) StatusErrorConvert(ctx context.Context, err error) error {
	return grpchelpers.StatusErrorConvert(ctx, err)
}

// ListSections returns the registered sections with counts.
func (h *Handler) ListSections(
	ctx context.Context,
	_ *connect.Request[workspacepb.ListSectionsRequest],
) (*connect.Response[workspacepb.ListSectionsResponse], error) {
	infos, err := h.service.ListSections(ctx)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&workspacepb.ListSectionsResponse{
		Sections: slices.To(infos, func(i entities.WorkspaceSectionInfo) *workspacepb.SectionInfo {
			return converter.Convert(i, &workspacepb.SectionInfo{})
		}),
	}), nil
}

// ExportWorkspace serializes the selected sections.
func (h *Handler) ExportWorkspace(
	ctx context.Context,
	req *connect.Request[workspacepb.ExportWorkspaceRequest],
) (*connect.Response[workspacepb.ExportWorkspaceResponse], error) {
	payload, err := h.service.Export(ctx, req.Msg.GetSectionKeys())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&workspacepb.ExportWorkspaceResponse{Payload: payload}), nil
}

// ValidateWorkspace dry-runs an import.
func (h *Handler) ValidateWorkspace(
	ctx context.Context,
	req *connect.Request[workspacepb.ValidateWorkspaceRequest],
) (*connect.Response[workspacepb.ValidateWorkspaceResponse], error) {
	reports, err := h.service.Validate(
		ctx,
		req.Msg.GetPayload(),
		req.Msg.GetSectionKeys(),
		strategyFromProto(req.Msg.GetStrategy()),
	)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&workspacepb.ValidateWorkspaceResponse{
		Reports: slices.To(reports, func(r entities.WorkspaceSectionReport) *workspacepb.SectionReport {
			return converter.Convert(r, &workspacepb.SectionReport{})
		}),
	}), nil
}

// ImportWorkspace applies the selected sections.
func (h *Handler) ImportWorkspace(
	ctx context.Context,
	req *connect.Request[workspacepb.ImportWorkspaceRequest],
) (*connect.Response[workspacepb.ImportWorkspaceResponse], error) {
	results, err := h.service.Import(
		ctx,
		req.Msg.GetPayload(),
		req.Msg.GetSectionKeys(),
		strategyFromProto(req.Msg.GetStrategy()),
	)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&workspacepb.ImportWorkspaceResponse{
		Results: slices.To(results, func(r entities.WorkspaceSectionResult) *workspacepb.SectionResult {
			return converter.Convert(r, &workspacepb.SectionResult{})
		}),
	}), nil
}

func strategyFromProto(s workspacepb.Strategy) entities.WorkspaceStrategy {
	if s == workspacepb.Strategy_STRATEGY_REPLACE {
		return entities.WorkspaceStrategyReplace
	}
	return entities.WorkspaceStrategyMerge
}
