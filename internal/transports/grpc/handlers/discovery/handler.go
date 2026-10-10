// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

// Package discovery is the Connect adapter for NATS Micro service discovery.
package discovery

import (
	"context"
	"net/http"

	"connectrpc.com/connect"

	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/transports/grpc/helpers"

	microsvc "github.com/dmit-4884/natscope/internal/services/micro"
	discoverypb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/discovery"
	discoveryconnect "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/discovery/grpc_nats_discoveryconnect"
)

// Handler implements the Connect DiscoveryServiceHandler interface.
type Handler struct {
	micro microsvc.Service
}

// New creates a DiscoveryService handler.
func New(micro microsvc.Service) *Handler {
	return &Handler{micro: micro}
}

// HTTPHandler returns the Connect route and handler.
func (h *Handler) HTTPHandler(opts ...connect.HandlerOption) (string, http.Handler) {
	opts = append(opts, connect.WithInterceptors(
		grpchelpers.NewHandlerErrorInterceptor(nil),
	))
	return discoveryconnect.NewDiscoveryServiceHandler(h, opts...)
}

// ListServices lists the NATS Micro services reachable through a connection.
func (h *Handler) ListServices(
	ctx context.Context,
	req *connect.Request[discoverypb.ListServicesRequest],
) (*connect.Response[discoverypb.ListServicesResponse], error) {
	discovery, err := h.micro.ListServices(ctx, req.Msg.GetConnectionId(), req.Msg.GetSkipStats())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(converter.Convert(discovery, &discoverypb.ListServicesResponse{}, grpchelpers.ProtoCodecs)), nil
}
