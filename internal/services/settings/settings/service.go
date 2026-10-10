// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package settings

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	"github.com/altessa-s/go-atlas/domain/normalizer"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"

	slogx "github.com/altessa-s/go-atlas/observability/slog"
	settingssvc "github.com/dmit-4884/natscope/internal/services/settings"
	storage "github.com/dmit-4884/natscope/internal/storages/settings"
)

// Service implements settings.Service, caching the current settings in memory.
type Service struct {
	storage storage.Storage
	logger  *slog.Logger

	mu     sync.Mutex
	cached *entities.UserSettings
}

// New creates a new settings service.
func New(storage storage.Storage) *Service {
	return &Service{
		storage: storage,
		logger:  slog.Default().With(slogx.Module("service:settings")),
	}
}

// Get returns settings, or default (empty) settings if none are saved. The
// result is shared and must not be modified.
func (s *Service) Get(ctx context.Context) (*entities.UserSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cached != nil {
		return s.cached, nil
	}
	result, err := s.storage.Get(ctx)
	if err != nil {
		if !errors.Is(err, errs.ErrSettingsNotFound) {
			return nil, err
		}
		result = entities.UserSettingsNew()
	}
	s.cached = result
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

	s.mu.Lock()
	defer s.mu.Unlock()

	updated, err := s.storage.Update(ctx, func(existing *entities.UserSettings) {
		existing.ApplyUpdate(in)
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to save settings",
			slogx.Error(err))
		return nil, err
	}
	s.cached = updated

	s.logger.InfoContext(ctx, "settings updated")

	return updated, nil
}

// Reset deletes settings, returning default (empty) settings.
func (s *Service) Reset(ctx context.Context) (*entities.UserSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cached = nil
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
