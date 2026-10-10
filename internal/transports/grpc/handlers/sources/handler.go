// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package sources

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"

	"github.com/altessa-s/go-atlas/core/collections/slices"
	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
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
	pb := converter.Convert(s, &protopb.ProtoSource{},
		converter.WithHandleEmbeddedStructs(true), grpchelpers.ProtoCodecs, converter.WithIgnoreFields("SelectedRef"))
	pb.SourceType = sourceTypeToProto(s.SourceType)
	if s.SelectedRef != nil {
		pb.SelectedRef = refToProto(*s.SelectedRef)
	}
	return pb
}

func refToProto(r entities.ProtoRef) *protopb.ProtoRef {
	pb := &protopb.ProtoRef{Name: r.Name, Revision: r.Revision}
	switch r.Kind {
	case entities.RefKindTag:
		pb.Kind = protopb.RefKind_REF_KIND_TAG
	case entities.RefKindBranch:
		pb.Kind = protopb.RefKind_REF_KIND_BRANCH
	case entities.RefKindCommit:
		pb.Kind = protopb.RefKind_REF_KIND_COMMIT
	case entities.RefKindLabel:
		pb.Kind = protopb.RefKind_REF_KIND_LABEL
	}
	return pb
}

func outcomeToProto(o *entities.CompileOutcome) *sourcespb.CompileOutcome {
	if o == nil {
		return nil
	}
	return &sourcespb.CompileOutcome{
		Valid:           o.Valid,
		MessageTypes:    int32(o.MessageTypes),    //nolint:gosec // bounded by descriptor size
		FileDescriptors: int32(o.FileDescriptors), //nolint:gosec // bounded by descriptor size
		Diagnostics:     diagnosticsToProto(o.Diagnostics),
	}
}

func sourceTypeToProto(st entities.SourceType) protopb.SourceType {
	switch st {
	case entities.SourceTypeGit:
		return protopb.SourceType_SOURCE_TYPE_GIT
	case entities.SourceTypeLocal:
		return protopb.SourceType_SOURCE_TYPE_LOCAL
	case entities.SourceTypeUpload:
		return protopb.SourceType_SOURCE_TYPE_UPLOAD
	case entities.SourceTypeBSR:
		return protopb.SourceType_SOURCE_TYPE_BSR
	default:
		return protopb.SourceType_SOURCE_TYPE_GIT
	}
}

// sourceTypeFromProto maps a wire SourceType to the domain type; ok is false for unspecified or unknown values.
func sourceTypeFromProto(st protopb.SourceType) (_ entities.SourceType, ok bool) {
	switch st {
	case protopb.SourceType_SOURCE_TYPE_GIT:
		return entities.SourceTypeGit, true
	case protopb.SourceType_SOURCE_TYPE_LOCAL:
		return entities.SourceTypeLocal, true
	case protopb.SourceType_SOURCE_TYPE_UPLOAD:
		return entities.SourceTypeUpload, true
	case protopb.SourceType_SOURCE_TYPE_BSR:
		return entities.SourceTypeBSR, true
	default:
		return "", false
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
	sourceType, ok := sourceTypeFromProto(in.SourceType)
	if !ok {
		return nil, fmt.Errorf("%w: unrecognized source_type", errs.ErrInvalidRequest)
	}
	createReq := converter.Convert(in, &entities.ProtoSourceCreate{},
		converter.WithIgnoreFields("SourceType"),
	)
	createReq.SourceType = sourceType

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
		listReq.Limit = new(int64(ps))
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
	sourceType := entities.SourceTypeGit
	if in.SourceType == protopb.SourceType_SOURCE_TYPE_BSR {
		sourceType = entities.SourceTypeBSR
	}
	result, err := h.protoService.ValidateRepository(ctx, sourceType, in.Repository, in.Token)
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

// ListRefs returns the tags and branches of a git source.
func (h *Handler) ListRefs(
	ctx context.Context,
	req *connect.Request[sourcespb.ListRefsRequest],
) (*connect.Response[sourcespb.ListRefsResponse], error) {
	refs, err := h.protoService.ListRefs(ctx, req.Msg.SourceId)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&sourcespb.ListRefsResponse{Refs: slices.To(refs, refToProto)}), nil
}

// SelectRef points a git source at a tag, branch or commit.
func (h *Handler) SelectRef(
	ctx context.Context,
	req *connect.Request[sourcespb.SelectRefRequest],
) (*connect.Response[sourcespb.SelectRefResponse], error) {
	source, outcome, err := h.protoService.SelectRef(ctx, req.Msg.SourceId, req.Msg.Ref)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&sourcespb.SelectRefResponse{
		Source:  h.sourceToProto(source),
		Outcome: outcomeToProto(outcome),
	}), nil
}

// RefreshSource rebuilds the active schema of a source.
func (h *Handler) RefreshSource(
	ctx context.Context,
	req *connect.Request[sourcespb.RefreshSourceRequest],
) (*connect.Response[sourcespb.RefreshSourceResponse], error) {
	source, outcome, err := h.protoService.RefreshSource(ctx, req.Msg.SourceId)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&sourcespb.RefreshSourceResponse{
		Source:  h.sourceToProto(source),
		Outcome: outcomeToProto(outcome),
	}), nil
}

// ListRevisions returns the stored schemas of a source.
func (h *Handler) ListRevisions(
	ctx context.Context,
	req *connect.Request[sourcespb.ListRevisionsRequest],
) (*connect.Response[sourcespb.ListRevisionsResponse], error) {
	revisions, err := h.protoService.ListRevisions(ctx, req.Msg.SourceId)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&sourcespb.ListRevisionsResponse{
		Revisions: slices.To(revisions, func(r entities.SchemaRevision) *protopb.SchemaRevision {
			return converter.Convert(r, &protopb.SchemaRevision{})
		}),
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

// UploadSchema makes uploaded .proto files, or a compiled descriptor set, the schema of an upload source.
func (h *Handler) UploadSchema(
	ctx context.Context,
	req *connect.Request[sourcespb.UploadSchemaRequest],
) (*connect.Response[sourcespb.UploadSchemaResponse], error) {
	upload := entities.SchemaUpload{DescriptorSet: req.Msg.GetDescriptorSet()}
	for _, f := range req.Msg.GetFiles().GetFiles() {
		upload.Files = append(upload.Files, entities.ProtoFileEntry{Path: f.GetPath(), Content: f.GetContent()})
	}
	source, outcome, err := h.protoService.UploadSchema(ctx, req.Msg.SourceId, upload)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&sourcespb.UploadSchemaResponse{
		Source:  h.sourceToProto(source),
		Outcome: outcomeToProto(outcome),
	}), nil
}
