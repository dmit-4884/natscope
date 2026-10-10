// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package connections

import (
	"context"
	"net/http"

	"connectrpc.com/connect"

	"github.com/altessa-s/go-atlas/core/collections/slices"
	"github.com/altessa-s/go-atlas/core/runtime/panics"
	"github.com/altessa-s/go-atlas/domain/converter"
	"github.com/altessa-s/go-atlas/domain/proto/fieldbehavior"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/natsutil"
	"github.com/dmit-4884/natscope/internal/transports/grpc/helpers"

	connectionssvc "github.com/dmit-4884/natscope/internal/services/connections"
	connectionspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/connections"
	connectionsconnect "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/connections/grpc_nats_connectionsconnect"
	natspb "github.com/dmit-4884/natscope/proto/gen/types/nats"
)

// Handler implements the Connect ConnectionsServiceHandler interface.
type Handler struct {
	connService connectionssvc.Service
}

// New creates a new ConnectionsService handler.
func New(connService connectionssvc.Service) *Handler {
	return &Handler{connService: connService}
}

// HTTPHandler returns the Connect route + handler, registering
// StatusErrorConvert as an interceptor to map domain errors before the wire.
func (h *Handler) HTTPHandler(opts ...connect.HandlerOption) (string, http.Handler) {
	opts = append(opts, connect.WithInterceptors(
		grpchelpers.NewHandlerErrorInterceptor(h.StatusErrorConvert),
	))
	return connectionsconnect.NewConnectionsServiceHandler(h, opts...)
}

// CreateConnection creates a new saved connection.
func (h *Handler) CreateConnection(
	ctx context.Context,
	req *connect.Request[connectionspb.CreateConnectionRequest],
) (*connect.Response[connectionspb.CreateConnectionResponse], error) {
	createReq := converter.Convert(req.Msg, &entities.SavedConnectionCreate{}, grpchelpers.ProtoCodecs)

	conn, err := h.connService.Create(ctx, createReq)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&connectionspb.CreateConnectionResponse{
		Connection: toProtoConnection(conn),
	}), nil
}

// ListConnections returns saved connections with AIP-158 pagination (page_token
// maps onto the storage cursor, page_size onto the limit).
func (h *Handler) ListConnections(
	ctx context.Context,
	req *connect.Request[connectionspb.ListConnectionsRequest],
) (*connect.Response[connectionspb.ListConnectionsResponse], error) {
	listReq := &entities.SavedConnectionsList{}
	if ps := req.Msg.GetPageSize(); ps > 0 {
		listReq.Limit = new(int64(ps))
	}
	listReq.Cursor = req.Msg.GetPageToken()

	list, err := h.connService.List(ctx, listReq)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&connectionspb.ListConnectionsResponse{
		Connections:   slices.To(list.Items, toProtoConnection),
		NextPageToken: list.NextCursor,
	}), nil
}

// UpdateConnection updates a saved connection.
func (h *Handler) UpdateConnection(
	ctx context.Context,
	req *connect.Request[connectionspb.UpdateConnectionRequest],
) (*connect.Response[connectionspb.UpdateConnectionResponse], error) {
	updateReq := converter.Convert(req.Msg, &entities.SavedConnectionUpdate{}, grpchelpers.ProtoCodecs)

	conn, err := h.connService.Update(ctx, updateReq)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&connectionspb.UpdateConnectionResponse{
		Connection: toProtoConnection(conn),
	}), nil
}

// DeleteConnection deletes a saved connection.
func (h *Handler) DeleteConnection(
	ctx context.Context,
	req *connect.Request[connectionspb.DeleteConnectionRequest],
) (*connect.Response[connectionspb.DeleteConnectionResponse], error) {
	if err := h.connService.Delete(ctx, req.Msg.Id); err != nil {
		return nil, err
	}
	return connect.NewResponse(&connectionspb.DeleteConnectionResponse{}), nil
}

// TestConnection probes a NATS connection (saved or ad-hoc).
func (h *Handler) TestConnection(
	ctx context.Context,
	req *connect.Request[connectionspb.TestConnectionRequest],
) (*connect.Response[connectionspb.TestConnectionResponse], error) {
	testReq := converter.Convert(req.Msg, &entities.TestConnectionRequest{}, grpchelpers.ProtoCodecs)
	result, err := h.connService.TestConnection(ctx, testReq)
	if err != nil {
		return nil, err
	}
	resp := converter.Convert(result, &connectionspb.TestConnectionResponse{})
	if !result.Success {
		return connect.NewResponse(&connectionspb.TestConnectionResponse{Error: &result.Error, Checks: resp.Checks}), nil
	}
	resp.Error = nil
	return connect.NewResponse(resp), nil
}

// DuplicateConnection duplicates a saved connection.
func (h *Handler) DuplicateConnection(
	ctx context.Context,
	req *connect.Request[connectionspb.DuplicateConnectionRequest],
) (*connect.Response[connectionspb.DuplicateConnectionResponse], error) {
	conn, err := h.connService.Duplicate(ctx, req.Msg.Id, req.Msg.Name)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&connectionspb.DuplicateConnectionResponse{
		Connection: toProtoConnection(conn),
	}), nil
}

// GetSidebarLayout returns how the sidebar arranges a connection's resources.
func (h *Handler) GetSidebarLayout(
	ctx context.Context,
	req *connect.Request[connectionspb.GetSidebarLayoutRequest],
) (*connect.Response[connectionspb.GetSidebarLayoutResponse], error) {
	layout, err := h.connService.GetSidebarLayout(ctx, req.Msg.ConnectionId)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&connectionspb.GetSidebarLayoutResponse{
		Layout: converter.Convert(layout, &connectionspb.SidebarLayout{}),
	}), nil
}

// UpdateSidebarLayout replaces the sidebar sections set in the request.
func (h *Handler) UpdateSidebarLayout(
	ctx context.Context,
	req *connect.Request[connectionspb.UpdateSidebarLayoutRequest],
) (*connect.Response[connectionspb.UpdateSidebarLayoutResponse], error) {
	layout, err := h.connService.UpdateSidebarLayout(ctx, converter.Convert(req.Msg, &entities.SidebarLayoutUpdate{}))
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&connectionspb.UpdateSidebarLayoutResponse{
		Layout: converter.Convert(layout, &connectionspb.SidebarLayout{}),
	}), nil
}

// toProtoConnection converts a saved connection to its wire form and redacts
// secrets: the secret values (NATS auth, TLS client key) are input-only and
// never echoed back — only boolean has_* presence flags are exposed so the UI
// can tell that a secret is stored.
func toProtoConnection(conn *entities.SavedConnection) *natspb.SavedConnection {
	pb := converter.Convert(
		conn,
		&natspb.SavedConnection{},
		converter.WithHandleEmbeddedStructs(true),
		grpchelpers.ProtoCodecs,
	)
	redactSecrets(pb)
	return pb
}

// redactSecrets sets the presence flag of every stored secret, then strips the
// INPUT_ONLY secret values from the wire message.
func redactSecrets(pb *natspb.SavedConnection) {
	if a := pb.GetAuth(); a != nil {
		a.HasPassword = a.Password != nil
		a.HasToken = a.Token != nil
		a.HasNkeySeed = a.NkeySeed != nil
		a.HasCredentials = a.Credentials != nil
		a.HasJwt = a.Jwt != nil
	}
	if t := pb.GetTls(); t != nil {
		t.HasClientKey = t.ClientKey != nil
	}
	panics.MustError(fieldbehavior.StripResponse(pb))
}

// ListCliContexts lists nats CLI contexts on this host or in uploaded files, without their secrets.
func (h *Handler) ListCliContexts(
	ctx context.Context,
	req *connect.Request[connectionspb.ListCliContextsRequest],
) (*connect.Response[connectionspb.ListCliContextsResponse], error) {
	if len(req.Msg.GetFiles()) == 0 && proxied(req.Header()) {
		return nil, errs.ErrCliContextsHostDisabled
	}
	found, err := h.connService.ListCliContexts(ctx, toCliContextFiles(req.Msg.GetFiles()))
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&connectionspb.ListCliContextsResponse{
		Directory: found.Dir,
		Contexts:  slices.To(found.Contexts, toProtoCliContext),
	}), nil
}

// ImportCliContexts creates connections from nats CLI contexts.
func (h *Handler) ImportCliContexts(
	ctx context.Context,
	req *connect.Request[connectionspb.ImportCliContextsRequest],
) (*connect.Response[connectionspb.ImportCliContextsResponse], error) {
	if len(req.Msg.GetFiles()) == 0 && proxied(req.Header()) {
		return nil, errs.ErrCliContextsHostDisabled
	}
	res, err := h.connService.ImportCliContexts(ctx, req.Msg.GetNames(), toCliContextFiles(req.Msg.GetFiles()))
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&connectionspb.ImportCliContextsResponse{
		Connections: slices.To(res.Created, toProtoConnection),
		Skipped: slices.To(res.Skipped, func(s entities.CliContextSkip) *connectionspb.SkippedCliContext {
			return &connectionspb.SkippedCliContext{Name: s.Name, Reason: s.Reason}
		}),
	}), nil
}

// proxied reports whether a reverse proxy forwarded the request, so it may come from another machine.
func proxied(header http.Header) bool {
	for _, name := range []string{"Forwarded", "X-Forwarded-For", "X-Real-Ip"} {
		if header.Get(name) != "" {
			return true
		}
	}
	return false
}

func toCliContextFiles(files []*connectionspb.CliContextFile) []entities.CliContextFile {
	return slices.To(files, func(f *connectionspb.CliContextFile) entities.CliContextFile {
		return entities.CliContextFile{Name: f.GetName(), Content: f.GetContent()}
	})
}

// toProtoCliContext describes the connection a context becomes; its secrets stay on the server.
func toProtoCliContext(c entities.CliContext) *connectionspb.CliContext {
	pb := &connectionspb.CliContext{
		Name:       c.Name,
		Selected:   c.Selected,
		Exists:     c.Exists,
		Importable: c.Connection != nil,
		Warnings:   c.Warnings,
	}
	conn := c.Connection
	if conn == nil {
		return pb
	}
	pb.Description = conn.Description
	pb.Urls = natsutil.StripCredentials(conn.URLs)
	if conn.Auth != nil {
		pb.AuthMethod = natspb.AuthMethod(conn.Auth.Method)
	}
	pb.Tls = !conn.TLS.IsEmpty()
	if cfg := conn.Connection; cfg != nil {
		pb.JetstreamDomain = cfg.JetstreamDomain
		pb.JetstreamApiPrefix = cfg.JetstreamAPIPrefix
		pb.InboxPrefix = cfg.InboxPrefix
	}
	return pb
}
