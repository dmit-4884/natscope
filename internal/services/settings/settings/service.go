// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package settings

import (
	"context"
	"errors"
	"log/slog"

	"github.com/altessa-s/go-atlas/domain/normalizer"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"

	slogx "github.com/altessa-s/go-atlas/observability/slog"
	settingssvc "github.com/dmit-4884/natscope/internal/services/settings"
	storage "github.com/dmit-4884/natscope/internal/storages/settings"
)

// Service implements settings.Service.
type Service struct {
	storage storage.Storage
	logger  *slog.Logger
}

// New creates a new settings service.
func New(storage storage.Storage) *Service {
	return &Service{
		storage: storage,
		logger:  slog.Default().With(slogx.Module("service:settings")),
	}
}

// Get returns settings. Returns default (empty) settings if none saved.
func (s *Service) Get(ctx context.Context) (*entities.UserSettings, error) {
	result, err := s.storage.Get(ctx)
	if err != nil {
		if errors.Is(err, errs.ErrSettingsNotFound) {
			return entities.UserSettingsNew(), nil
		}
		return nil, err
	}

	return result, nil
}

// Update partially updates settings (creates if not exists).
func (s *Service) Update(ctx context.Context, in *entities.UserSettingsUpdate) (*entities.UserSettings, error) {
	_ = normalizer.Normalize(in) //nolint:errcheck // canonical: normalize tags can't fail on a well-formed DTO

	// Fold export-range-limit 0 back to nil: 0 means "use default", never
	// "unlimited".
	if in != nil && in.Messages != nil && in.Messages.ExportRangeLimit != nil && *in.Messages.ExportRangeLimit == 0 {
		in.Messages.ExportRangeLimit = nil
	}

	existing, err := s.storage.Get(ctx)
	if err != nil {
		if !errors.Is(err, errs.ErrSettingsNotFound) {
			return nil, err
		}

		// Create new settings entity
		existing = entities.UserSettingsNew()
	}

	existing.ApplyUpdate(in)

	if err := s.storage.Save(ctx, existing); err != nil {
		s.logger.ErrorContext(ctx, "failed to save settings",
			slogx.Error(err))
		return nil, err
	}

	s.logger.InfoContext(ctx, "settings updated")

	return existing, nil
}

// Reset deletes settings, returning default (empty) settings.
func (s *Service) Reset(ctx context.Context) (*entities.UserSettings, error) {
	if err := s.storage.Delete(ctx); err != nil {
		if !errors.Is(err, errs.ErrSettingsNotFound) {
			s.logger.ErrorContext(ctx, "failed to delete settings",
				slogx.Error(err))
			return nil, err
		}
	}

	s.logger.InfoContext(ctx, "settings reset")

	return entities.UserSettingsNew(), nil
}

// Compile-time interface check.
var _ settingssvc.Service = (*Service)(nil)
