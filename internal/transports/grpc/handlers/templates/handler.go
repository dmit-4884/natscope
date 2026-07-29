// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package templates

import (
	"context"
	"net/http"

	"connectrpc.com/connect"

	"github.com/altessa-s/go-atlas/core/collections/slices"
	"github.com/altessa-s/go-atlas/core/types/ptr"
	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/transports/grpc/helpers"

	templatessvc "github.com/dmit-4884/natscope/internal/services/templates"
	templatespb "github.com/dmit-4884/natscope/proto/gen/services/grpc/templates/v1/templates"
	templatesconnect "github.com/dmit-4884/natscope/proto/gen/services/grpc/templates/v1/templates/grpc_templatesconnect"
	templatestypespb "github.com/dmit-4884/natscope/proto/gen/types/templates"
)

// Handler implements the Connect TemplatesServiceHandler interface.
type Handler struct {
	service templatessvc.Service
}

// New creates a new TemplatesService handler.
func New(service templatessvc.Service) *Handler {
	return &Handler{service: service}
}

// HTTPHandler returns the Connect route + handler, registering
// StatusErrorConvert as an interceptor.
func (h *Handler) HTTPHandler(opts ...connect.HandlerOption) (string, http.Handler) {
	opts = append(opts, connect.WithInterceptors(
		grpchelpers.NewHandlerErrorInterceptor(h.StatusErrorConvert),
	))
	return templatesconnect.NewTemplatesServiceHandler(h, opts...)
}

func toProto(t *entities.MessageTemplate) *templatestypespb.MessageTemplate {
	if t == nil {
		return nil
	}
	return converter.Convert(
		t,
		&templatestypespb.MessageTemplate{},
		converter.WithHandleEmbeddedStructs(true),
		grpchelpers.ProtoCodecs,
	)
}

// CreateTemplate creates a new template.
func (h *Handler) CreateTemplate(
	ctx context.Context,
	req *connect.Request[templatespb.CreateTemplateRequest],
) (*connect.Response[templatespb.CreateTemplateResponse], error) {
	in := req.Msg
	createReq := converter.Convert(in, &entities.MessageTemplateCreate{})
	t, err := h.service.Create(ctx, createReq)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&templatespb.CreateTemplateResponse{Template: toProto(t)}), nil
}

// UpdateTemplate applies a partial update to an existing template.
func (h *Handler) UpdateTemplate(
	ctx context.Context,
	req *connect.Request[templatespb.UpdateTemplateRequest],
) (*connect.Response[templatespb.UpdateTemplateResponse], error) {
	in := req.Msg
	updateReq := converter.Convert(in, &entities.MessageTemplateUpdate{}, converter.WithIgnoreFields("Headers", "Wildcards"))
	if in.Headers != nil {
		if in.Headers.Values == nil {
			updateReq.Headers = map[string]string{}
		} else {
			updateReq.Headers = in.Headers.Values
		}
	}
	if in.Wildcards != nil {
		if in.Wildcards.Values == nil {
			updateReq.Wildcards = []string{}
		} else {
			updateReq.Wildcards = in.Wildcards.Values
		}
	}

	t, err := h.service.Update(ctx, updateReq)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&templatespb.UpdateTemplateResponse{Template: toProto(t)}), nil
}

// GetTemplate retrieves a template by id.
func (h *Handler) GetTemplate(
	ctx context.Context,
	req *connect.Request[templatespb.GetTemplateRequest],
) (*connect.Response[templatespb.GetTemplateResponse], error) {
	t, err := h.service.Get(ctx, req.Msg.Id)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&templatespb.GetTemplateResponse{Template: toProto(t)}), nil
}

// ListTemplates returns templates with AIP-158 pagination (page_token→cursor,
// page_size→limit).
func (h *Handler) ListTemplates(
	ctx context.Context,
	req *connect.Request[templatespb.ListTemplatesRequest],
) (*connect.Response[templatespb.ListTemplatesResponse], error) {
	in := req.Msg
	listReq := &entities.MessageTemplatesList{}
	if ps := in.GetPageSize(); ps > 0 {
		listReq.Limit = ptr.Wrap(int64(ps))
	}
	listReq.Cursor = in.GetPageToken()
	listReq.IncludeTotalCount = in.GetIncludeTotalCount()

	list, err := h.service.List(ctx, listReq)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&templatespb.ListTemplatesResponse{
		Templates:     slices.To(list.Items, toProto),
		NextPageToken: list.NextCursor,
		TotalSize:     list.Total,
	}), nil
}

// DeleteTemplate removes a template.
func (h *Handler) DeleteTemplate(
	ctx context.Context,
	req *connect.Request[templatespb.DeleteTemplateRequest],
) (*connect.Response[templatespb.DeleteTemplateResponse], error) {
	if err := h.service.Delete(ctx, req.Msg.Id); err != nil {
		return nil, err
	}
	return connect.NewResponse(&templatespb.DeleteTemplateResponse{}), nil
}

// BatchCreateTemplates inserts each item as a new template.
func (h *Handler) BatchCreateTemplates(
	ctx context.Context,
	req *connect.Request[templatespb.BatchCreateTemplatesRequest],
) (*connect.Response[templatespb.BatchCreateTemplatesResponse], error) {
	in := req.Msg
	items := slices.To(in.Templates, func(t *templatespb.TemplateBulkCreateItem) *entities.MessageTemplateCreate {
		return converter.Convert(t, &entities.MessageTemplateCreate{})
	})
	created, err := h.service.BulkCreate(ctx, items)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&templatespb.BatchCreateTemplatesResponse{Created: int32(created)}), nil
}

// DeleteAllTemplates removes every stored template.
func (h *Handler) DeleteAllTemplates(
	ctx context.Context,
	_ *connect.Request[templatespb.DeleteAllTemplatesRequest],
) (*connect.Response[templatespb.DeleteAllTemplatesResponse], error) {
	n, err := h.service.DeleteAll(ctx)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&templatespb.DeleteAllTemplatesResponse{Deleted: int32(n)}), nil
}
