// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package history

import (
	"context"
	"net/http"

	"connectrpc.com/connect"

	"github.com/altessa-s/go-atlas/core/collections/slices"
	"github.com/altessa-s/go-atlas/core/types/ptr"
	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/transports/grpc/helpers"

	historysvc "github.com/dmit-4884/natscope/internal/services/history"
	historypb "github.com/dmit-4884/natscope/proto/gen/services/grpc/history/v1/history"
	historyconnect "github.com/dmit-4884/natscope/proto/gen/services/grpc/history/v1/history/grpc_historyconnect"
	historytypespb "github.com/dmit-4884/natscope/proto/gen/types/history"
)

// Handler implements the Connect HistoryServiceHandler interface.
type Handler struct {
	historyService historysvc.Service
}

// New creates a new HistoryService handler.
func New(historyService historysvc.Service) *Handler {
	return &Handler{historyService: historyService}
}

// HTTPHandler returns the Connect route + handler; uses the shared error
// interceptor since history surfaces only common storage errors.
func (h *Handler) HTTPHandler(opts ...connect.HandlerOption) (string, http.Handler) {
	opts = append(opts, connect.WithInterceptors(
		grpchelpers.NewHandlerErrorInterceptor(nil),
	))
	return historyconnect.NewHistoryServiceHandler(h, opts...)
}

// ListPublishHistory returns publish history with AIP-158 pagination
// (page_token→cursor, page_size→limit).
func (h *Handler) ListPublishHistory(
	ctx context.Context,
	req *connect.Request[historypb.ListPublishHistoryRequest],
) (*connect.Response[historypb.ListPublishHistoryResponse], error) {
	in := req.Msg
	listReq := &entities.PublishHistoryList{
		ConnectionURL: in.ConnectionUrl,
		Stream:        in.Stream,
	}
	if ps := in.GetPageSize(); ps > 0 {
		listReq.Limit = ptr.Wrap(int64(ps))
	}
	listReq.Cursor = in.GetPageToken()

	list, err := h.historyService.List(ctx, listReq)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&historypb.ListPublishHistoryResponse{
		Entries: slices.To(list.Items, func(item *entities.PublishHistory) *historytypespb.PublishHistory {
			return converter.Convert(
				item,
				&historytypespb.PublishHistory{},
				converter.WithHandleEmbeddedStructs(true),
				grpchelpers.ProtoCodecs,
			)
		}),
		NextPageToken: list.NextCursor,
	}), nil
}
