// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package sources

import (
	"context"
	"net/http"

	"connectrpc.com/connect"

	"github.com/altessa-s/go-atlas/core/collections/slices"
	"github.com/altessa-s/go-atlas/core/types/ptr"
	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/transports/grpc/helpers"

	protosvc "github.com/dmit-4884/natscope/internal/services/proto"
	sourcespb "github.com/dmit-4884/natscope/proto/gen/services/grpc/proto/v1/sources"
	sourcesconnect "github.com/dmit-4884/natscope/proto/gen/services/grpc/proto/v1/sources/grpc_proto_sourcesconnect"
	protopb "github.com/dmit-4884/natscope/proto/gen/types/proto"
)

// Handler implements the Connect SourcesServiceHandler interface.
type Handler struct {
	protoService protosvc.SourceManager
}

// New creates a new SourcesService handler.
func New(protoService protosvc.SourceManager) *Handler {
	return &Handler{protoService: protoService}
}

// HTTPHandler returns the Connect route + handler, registering
// StatusErrorConvert as an interceptor.
func (h *Handler) HTTPHandler(opts ...connect.HandlerOption) (string, http.Handler) {
	opts = append(opts, connect.WithInterceptors(
		grpchelpers.NewHandlerErrorInterceptor(h.StatusErrorConvert),
	))
	return sourcesconnect.NewSourcesServiceHandler(h, opts...)
}

func (h *Handler) sourceToProto(s *entities.ProtoSource) *protopb.ProtoSource {
	pb := converter.Convert(s, &protopb.ProtoSource{}, converter.WithHandleEmbeddedStructs(true), grpchelpers.ProtoCodecs)
	pb.SourceType = sourceTypeToProto(s.SourceType)
	return pb
}

func sourceTypeToProto(st entities.SourceType) protopb.SourceType {
	switch st {
	case entities.SourceTypeGit:
		return protopb.SourceType_SOURCE_TYPE_GIT
	case entities.SourceTypeLocal:
		return protopb.SourceType_SOURCE_TYPE_LOCAL
	case entities.SourceTypeFiles:
		return protopb.SourceType_SOURCE_TYPE_FILES
	default:
		return protopb.SourceType_SOURCE_TYPE_GIT
	}
}

func sourceTypeFromProto(st protopb.SourceType) entities.SourceType {
	switch st {
	case protopb.SourceType_SOURCE_TYPE_LOCAL:
		return entities.SourceTypeLocal
	case protopb.SourceType_SOURCE_TYPE_FILES:
		return entities.SourceTypeFiles
	default:
		return entities.SourceTypeGit
	}
}

func diagnosticToProto(d entities.CompileDiagnostic) *protopb.CompileDiagnostic {
	return converter.Convert(d, &protopb.CompileDiagnostic{})
}

func diagnosticsToProto(diags []entities.CompileDiagnostic) []*protopb.CompileDiagnostic {
	return slices.To(diags, diagnosticToProto)
}

// CreateSource creates a new proto source.
func (h *Handler) CreateSource(
	ctx context.Context,
	req *connect.Request[sourcespb.CreateSourceRequest],
) (*connect.Response[sourcespb.CreateSourceResponse], error) {
	in := req.Msg
	createReq := converter.Convert(in, &entities.ProtoSourceCreate{},
		converter.WithIgnoreFields("SourceType"),
	)
	createReq.SourceType = sourceTypeFromProto(in.SourceType)

	source, err := h.protoService.CreateSource(ctx, createReq)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&sourcespb.CreateSourceResponse{Source: h.sourceToProto(source)}), nil
}

// GetSource returns a specific proto source.
func (h *Handler) GetSource(
	ctx context.Context,
	req *connect.Request[sourcespb.GetSourceRequest],
) (*connect.Response[sourcespb.GetSourceResponse], error) {
	source, err := h.protoService.GetSource(ctx, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&sourcespb.GetSourceResponse{Source: h.sourceToProto(source)}), nil
}

// ListSources returns proto sources with AIP-158 pagination
// (page_token->cursor, page_size->limit).
func (h *Handler) ListSources(
	ctx context.Context,
	req *connect.Request[sourcespb.ListSourcesRequest],
) (*connect.Response[sourcespb.ListSourcesResponse], error) {
	in := req.Msg
	listReq := &entities.ProtoSourcesList{}
	if ps := in.GetPageSize(); ps > 0 {
		listReq.Limit = ptr.Wrap(int64(ps))
	}
	listReq.Cursor = in.GetPageToken()

	list, err := h.protoService.ListSources(ctx, listReq)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&sourcespb.ListSourcesResponse{
		Sources:       slices.To(list.Items, h.sourceToProto),
		NextPageToken: list.NextCursor,
	}), nil
}

// UpdateSource updates a proto source.
func (h *Handler) UpdateSource(
	ctx context.Context,
	req *connect.Request[sourcespb.UpdateSourceRequest],
) (*connect.Response[sourcespb.UpdateSourceResponse], error) {
	updateReq := converter.Convert(req.Msg, &entities.ProtoSourceUpdate{})

	source, err := h.protoService.UpdateSource(ctx, updateReq)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&sourcespb.UpdateSourceResponse{Source: h.sourceToProto(source)}), nil
}

// DeleteSource deletes a proto source.
func (h *Handler) DeleteSource(
	ctx context.Context,
	req *connect.Request[sourcespb.DeleteSourceRequest],
) (*connect.Response[sourcespb.DeleteSourceResponse], error) {
	if err := h.protoService.DeleteSource(ctx, req.Msg.Id); err != nil {
		return nil, err
	}
	return connect.NewResponse(&sourcespb.DeleteSourceResponse{}), nil
}

// ValidateRepository validates that a repository is accessible.
func (h *Handler) ValidateRepository(
	ctx context.Context,
	req *connect.Request[sourcespb.ValidateRepositoryRequest],
) (*connect.Response[sourcespb.ValidateRepositoryResponse], error) {
	in := req.Msg
	result, err := h.protoService.ValidateRepository(ctx, in.Repository, in.Token)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&sourcespb.ValidateRepositoryResponse{
		Valid: result.Valid, Error: result.Error,
	}), nil
}

// ValidateLocalPath validates that a local directory exists and contains .proto
// files.
func (h *Handler) ValidateLocalPath(
	ctx context.Context,
	req *connect.Request[sourcespb.ValidateLocalPathRequest],
) (*connect.Response[sourcespb.ValidateLocalPathResponse], error) {
	result, err := h.protoService.ValidateLocalPath(ctx, req.Msg.Path)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&sourcespb.ValidateLocalPathResponse{
		Valid:          result.Valid,
		ProtoFileCount: int32(result.ProtoFileCount),
		Error:          result.Error,
	}), nil
}

// ListTags returns available tags from a source's Git repository.
func (h *Handler) ListTags(
	ctx context.Context,
	req *connect.Request[sourcespb.ListTagsRequest],
) (*connect.Response[sourcespb.ListTagsResponse], error) {
	tags, err := h.protoService.ListTags(ctx, req.Msg.SourceId)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&sourcespb.ListTagsResponse{Tags: tags}), nil
}

// FetchVersion fetches proto files for a specific tag.
func (h *Handler) FetchVersion(
	ctx context.Context,
	req *connect.Request[sourcespb.FetchVersionRequest],
) (*connect.Response[sourcespb.FetchVersionResponse], error) {
	in := req.Msg
	version, err := h.protoService.FetchVersion(ctx, in.SourceId, in.Tag)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&sourcespb.FetchVersionResponse{
		Version: converter.Convert(
			version,
			&protopb.ProtoVersion{},
			converter.WithHandleEmbeddedStructs(true),
			grpchelpers.ProtoCodecs,
		),
	}), nil
}

// SetEnabled enables or disables a source.
func (h *Handler) SetEnabled(
	ctx context.Context,
	req *connect.Request[sourcespb.SetEnabledRequest],
) (*connect.Response[sourcespb.SetEnabledResponse], error) {
	source, err := h.protoService.SetEnabled(ctx, req.Msg.SourceId, req.Msg.Enabled)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&sourcespb.SetEnabledResponse{Source: h.sourceToProto(source)}), nil
}

// SetWatcher enables or disables file watcher for a local directory source.
func (h *Handler) SetWatcher(
	ctx context.Context,
	req *connect.Request[sourcespb.SetWatcherRequest],
) (*connect.Response[sourcespb.SetWatcherResponse], error) {
	source, err := h.protoService.SetWatcher(ctx, req.Msg.SourceId, req.Msg.Enabled)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&sourcespb.SetWatcherResponse{Source: h.sourceToProto(source)}), nil
}

// CompileLocal triggers manual compilation of a local directory source.
func (h *Handler) CompileLocal(
	ctx context.Context,
	req *connect.Request[sourcespb.CompileLocalRequest],
) (*connect.Response[sourcespb.CompileLocalResponse], error) {
	result, diags, err := h.protoService.CompileLocal(ctx, req.Msg.SourceId)
	if err != nil {
		return nil, err
	}
	resp := &sourcespb.CompileLocalResponse{
		Diagnostics: diagnosticsToProto(diags),
	}
	if result != nil {
		resp.Valid = true
		resp.MessageTypes = int32(result.MessageTypes)
		resp.FileDescriptors = int32(result.FileDescriptors)
	}
	return connect.NewResponse(resp), nil
}

// ValidateFiles validates a Files-type source without persisting anything.
func (h *Handler) ValidateFiles(
	ctx context.Context,
	req *connect.Request[sourcespb.ValidateFilesRequest],
) (*connect.Response[sourcespb.ValidateFilesResponse], error) {
	in := req.Msg
	result, diags, err := h.protoService.ValidateFiles(ctx, in.SourceId, in.Files, in.IncludeDirs)
	if err != nil {
		return nil, err
	}
	resp := &sourcespb.ValidateFilesResponse{
		Diagnostics: diagnosticsToProto(diags),
	}
	if result != nil {
		resp.Valid = true
		resp.MessageTypes = int32(result.MessageTypes)
		resp.FileDescriptors = int32(result.FileDescriptors)
	}
	return connect.NewResponse(resp), nil
}

// CompileFiles compiles a Files-type source and persists its descriptor.
func (h *Handler) CompileFiles(
	ctx context.Context,
	req *connect.Request[sourcespb.CompileFilesRequest],
) (*connect.Response[sourcespb.CompileFilesResponse], error) {
	result, diags, err := h.protoService.CompileFiles(ctx, req.Msg.SourceId)
	if err != nil {
		return nil, err
	}
	resp := &sourcespb.CompileFilesResponse{
		Diagnostics: diagnosticsToProto(diags),
	}
	if result != nil {
		resp.Valid = true
		resp.MessageTypes = int32(result.MessageTypes)
		resp.FileDescriptors = int32(result.FileDescriptors)
	}
	return connect.NewResponse(resp), nil
}
