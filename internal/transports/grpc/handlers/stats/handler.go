// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package stats

import (
	"context"
	"net/http"

	"connectrpc.com/connect"

	"github.com/altessa-s/go-atlas/core/collections/slices"
	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/transports/grpc/helpers"

	natssvc "github.com/dmit-4884/natscope/internal/services/nats"
	statspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/stats"
	statsconnect "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/stats/grpc_nats_statsconnect"
	natspb "github.com/dmit-4884/natscope/proto/gen/types/nats"
)

// natsDeps bundles the narrow NATS roles the stats handler needs: stream/consumer
// statistics plus connection health. The roles are injected separately so fx can
// resolve each, then embedded here for internal use.
type natsDeps struct {
	natssvc.StatsReader
	natssvc.ConnectionManager
}

// Handler implements the Connect StatsServiceHandler interface.
type Handler struct {
	natsService natsDeps
}

// New creates a new StatsService handler.
func New(statsReader natssvc.StatsReader, connManager natssvc.ConnectionManager) *Handler {
	return &Handler{natsService: natsDeps{statsReader, connManager}}
}

// HTTPHandler returns the Connect route + handler; uses the shared error
// interceptor.
func (h *Handler) HTTPHandler(opts ...connect.HandlerOption) (string, http.Handler) {
	opts = append(opts, connect.WithInterceptors(
		grpchelpers.NewHandlerErrorInterceptor(nil),
	))
	return statsconnect.NewStatsServiceHandler(h, opts...)
}

// GetAllStreamsStats returns stats for all streams on a connection.
func (h *Handler) GetAllStreamsStats(
	ctx context.Context,
	req *connect.Request[statspb.GetAllStreamsStatsRequest],
) (*connect.Response[statspb.GetAllStreamsStatsResponse], error) {
	stats, err := h.natsService.GetAllStreamsStats(ctx, req.Msg.ConnectionId)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&statspb.GetAllStreamsStatsResponse{
		Streams: slices.To(stats, func(s entities.StreamStats) *natspb.StreamStats {
			return converter.Convert(s, &natspb.StreamStats{}, grpchelpers.ProtoCodecs)
		}),
	}), nil
}

// GetStreamStats returns stats for a specific stream.
func (h *Handler) GetStreamStats(
	ctx context.Context,
	req *connect.Request[statspb.GetStreamStatsRequest],
) (*connect.Response[statspb.GetStreamStatsResponse], error) {
	s, err := h.natsService.GetStreamStats(ctx, req.Msg.ConnectionId, req.Msg.StreamName)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&statspb.GetStreamStatsResponse{
		Stream: converter.Convert(s, &natspb.StreamStats{}, grpchelpers.ProtoCodecs),
	}), nil
}

// GetAllConsumers returns all consumers across all streams.
func (h *Handler) GetAllConsumers(
	ctx context.Context,
	req *connect.Request[statspb.GetAllConsumersRequest],
) (*connect.Response[statspb.GetAllConsumersResponse], error) {
	consumers, err := h.natsService.GetAllConsumers(ctx, req.Msg.ConnectionId)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&statspb.GetAllConsumersResponse{
		Consumers: slices.To(consumers, func(c entities.ConsumerStats) *natspb.ConsumerStats {
			return converter.Convert(&c, &natspb.ConsumerStats{}, grpchelpers.ProtoCodecs)
		}),
	}), nil
}

// GetHealth returns connection health status.
func (h *Handler) GetHealth(
	ctx context.Context,
	req *connect.Request[statspb.GetHealthRequest],
) (*connect.Response[statspb.GetHealthResponse], error) {
	health, err := h.natsService.GetConnectionHealth(ctx, req.Msg.ConnectionId)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&statspb.GetHealthResponse{
		Health: converter.Convert(health, &natspb.ConnectionHealth{}),
	}), nil
}

// GetServerInfo returns detailed NATS server information.
func (h *Handler) GetServerInfo(
	ctx context.Context,
	req *connect.Request[statspb.GetServerInfoRequest],
) (*connect.Response[statspb.GetServerInfoResponse], error) {
	info, err := h.natsService.GetServerInfo(ctx, req.Msg.ConnectionId)
	if err != nil {
		return nil, err
	}
	pb := converter.Convert(info, &natspb.ServerInfo{})
	return connect.NewResponse(&statspb.GetServerInfoResponse{ServerInfo: pb}), nil
}
