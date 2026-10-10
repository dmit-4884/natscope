// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

// Package management provides Connect handlers for JetStream CRUD (streams,
// consumers, KV, object stores); methods split by resource type.
package management

import (
	"net/http"

	"connectrpc.com/connect"

	"github.com/dmit-4884/natscope/internal/transports/grpc/helpers"

	natssvc "github.com/dmit-4884/natscope/internal/services/nats"
	protosvc "github.com/dmit-4884/natscope/internal/services/proto"
	settingssvc "github.com/dmit-4884/natscope/internal/services/settings"
	managementconnect "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/management/grpc_nats_managementconnect"
)

// natsDeps bundles the narrow NATS roles the management handler needs: stream
// and consumer CRUD plus KV and object-store management. The roles are injected
// separately so fx can resolve each, then embedded here for internal use.
type natsDeps struct {
	natssvc.StreamReader
	natssvc.StreamManager
	natssvc.ConsumerManager
	natssvc.KVStore
	natssvc.ObjectStore
}

// Handler implements the Connect ManagementServiceHandler interface.
type Handler struct {
	natsService natsDeps
	codec       protosvc.Codec
	settings    settingssvc.Service
}

// New creates a new ManagementService handler; codec and settings decode and encode KV values.
func New(
	streamReader natssvc.StreamReader,
	streamManager natssvc.StreamManager,
	consumerManager natssvc.ConsumerManager,
	kvStore natssvc.KVStore,
	objectStore natssvc.ObjectStore,
	codec protosvc.Codec,
	settings settingssvc.Service,
) *Handler {
	return &Handler{
		natsService: natsDeps{streamReader, streamManager, consumerManager, kvStore, objectStore},
		codec:       codec,
		settings:    settings,
	}
}

// HTTPHandler returns the Connect route + handler; registers the shared error
// interceptor (all errors are NATS-domain).
func (h *Handler) HTTPHandler(opts ...connect.HandlerOption) (string, http.Handler) {
	opts = append(opts, connect.WithInterceptors(
		grpchelpers.NewHandlerErrorInterceptor(nil),
	))
	return managementconnect.NewManagementServiceHandler(h, opts...)
}
