// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package connections

import (
	"context"
	"net/http"

	"connectrpc.com/connect"

	"github.com/altessa-s/go-atlas/core/collections/slices"
	"github.com/altessa-s/go-atlas/core/types/ptr"
	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"
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
		listReq.Limit = ptr.Wrap(int64(ps))
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
	return connect.NewResponse(converter.Convert(result, &connectionspb.TestConnectionResponse{})), nil
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

// redactSecrets clears every secret value on the wire message and sets the
// matching presence flag when a value was stored.
func redactSecrets(pb *natspb.SavedConnection) {
	if a := pb.GetAuth(); a != nil {
		a.HasPassword = a.Password != nil
		a.HasToken = a.Token != nil
		a.HasNkeySeed = a.NkeySeed != nil
		a.HasCredentials = a.Credentials != nil
		a.HasJwt = a.Jwt != nil
		a.Password, a.Token, a.NkeySeed, a.Credentials, a.Jwt = nil, nil, nil, nil, nil
	}
	if t := pb.GetTls(); t != nil {
		t.HasClientKey = t.ClientKey != nil
		t.ClientKey = nil
	}
}
