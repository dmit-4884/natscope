// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package settings

import (
	"context"
	"net/http"

	"connectrpc.com/connect"

	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/transports/grpc/helpers"

	settingssvc "github.com/dmit-4884/natscope/internal/services/settings"
	settingspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/settings/v1/settings"
	settingsconnect "github.com/dmit-4884/natscope/proto/gen/services/grpc/settings/v1/settings/grpc_settingsconnect"
	settingstypespb "github.com/dmit-4884/natscope/proto/gen/types/settings"
)

// Handler implements the Connect SettingsServiceHandler interface.
type Handler struct {
	settingsService settingssvc.Service
}

// New creates a new SettingsService handler.
func New(settingsService settingssvc.Service) *Handler {
	return &Handler{settingsService: settingsService}
}

// HTTPHandler returns the Connect route + handler, registering
// StatusErrorConvert as an interceptor.
func (h *Handler) HTTPHandler(opts ...connect.HandlerOption) (string, http.Handler) {
	opts = append(opts, connect.WithInterceptors(
		grpchelpers.NewHandlerErrorInterceptor(h.StatusErrorConvert),
	))
	return settingsconnect.NewSettingsServiceHandler(h, opts...)
}

// GetSettings returns the current settings.
func (h *Handler) GetSettings(
	ctx context.Context,
	_ *connect.Request[settingspb.GetSettingsRequest],
) (*connect.Response[settingspb.GetSettingsResponse], error) {
	result, err := h.settingsService.Get(ctx)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&settingspb.GetSettingsResponse{
		Settings: converter.Convert(
			result,
			&settingstypespb.UserSettings{},
			converter.WithHandleEmbeddedStructs(true),
			grpchelpers.ProtoCodecs,
		),
	}), nil
}

// UpdateSettings updates the current settings.
func (h *Handler) UpdateSettings(
	ctx context.Context,
	req *connect.Request[settingspb.UpdateSettingsRequest],
) (*connect.Response[settingspb.UpdateSettingsResponse], error) {
	updateReq := converter.Convert(req.Msg, &entities.UserSettingsUpdate{})

	result, err := h.settingsService.Update(ctx, updateReq)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&settingspb.UpdateSettingsResponse{
		Settings: converter.Convert(
			result,
			&settingstypespb.UserSettings{},
			converter.WithHandleEmbeddedStructs(true),
			grpchelpers.ProtoCodecs,
		),
	}), nil
}

// ResetSettings resets the current settings to defaults.
func (h *Handler) ResetSettings(
	ctx context.Context,
	_ *connect.Request[settingspb.ResetSettingsRequest],
) (*connect.Response[settingspb.ResetSettingsResponse], error) {
	result, err := h.settingsService.Reset(ctx)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&settingspb.ResetSettingsResponse{
		Settings: converter.Convert(
			result,
			&settingstypespb.UserSettings{},
			converter.WithHandleEmbeddedStructs(true),
			grpchelpers.ProtoCodecs,
		),
	}), nil
}
