// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package templates

import (
	"context"
	"log/slog"
	"strings"

	"github.com/altessa-s/go-atlas/domain/converter"
	"github.com/altessa-s/go-atlas/domain/normalizer"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"

	slogx "github.com/altessa-s/go-atlas/observability/slog"
	templatessvc "github.com/dmit-4884/natscope/internal/services/templates"
	storage "github.com/dmit-4884/natscope/internal/storages/templates"
)

// Service implements templates.Service over a generic Storage.
type Service struct {
	storage storage.Storage
	logger  *slog.Logger
}

// New creates a new templates Service.
func New(st storage.Storage) *Service {
	return &Service{
		storage: st,
		logger:  slog.Default().With(slogx.Module("service:templates")),
	}
}

// Create creates a new template.
func (s *Service) Create(
	ctx context.Context,
	in *entities.MessageTemplateCreate,
) (*entities.MessageTemplate, error) {
	_ = normalizer.Normalize(in) //nolint:errcheck // canonical: normalize tags can't fail on a well-formed DTO

	// buf.validate min_len=1 runs on the raw request; re-check after the trim
	// normalizer so a whitespace-only name can't persist as an empty name.
	if strings.TrimSpace(in.Name) == "" {
		return nil, errs.ErrMessageTemplateNameRequired
	}

	t := converter.Convert(in, entities.MessageTemplateNew())

	if err := s.storage.Save(ctx, t); err != nil {
		s.logger.ErrorContext(ctx, "failed to save template",
			slog.String("name", in.Name),
			slogx.Error(err))
		return nil, err
	}
	s.logger.InfoContext(ctx, "template created",
		slog.String("id", t.Id),
		slog.String("name", t.Name))
	return t, nil
}

// Get retrieves a template by id.
func (s *Service) Get(ctx context.Context, id string) (*entities.MessageTemplate, error) {
	return s.storage.Get(ctx, id)
}

// Update applies a partial update to an existing template.
func (s *Service) Update(
	ctx context.Context,
	in *entities.MessageTemplateUpdate,
) (*entities.MessageTemplate, error) {
	_ = normalizer.Normalize(in) //nolint:errcheck // canonical: normalize tags can't fail on a well-formed DTO

	// A provided name that trims to empty must be rejected (UpdateTemplate has
	// no buf.validate min_len on name, so this is the only guard).
	if in.Name != nil && strings.TrimSpace(*in.Name) == "" {
		return nil, errs.ErrMessageTemplateNameRequired
	}

	updated, err := s.storage.Update(ctx, in.Id, func(existing *entities.MessageTemplate) {
		existing.ApplyUpdate(in)
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to update template",
			slog.String("id", in.Id),
			slogx.Error(err))
		return nil, err
	}
	return updated, nil
}

// List returns templates with cursor-based pagination.
func (s *Service) List(
	ctx context.Context,
	in *entities.MessageTemplatesList,
) (*entities.List[entities.MessageTemplates], error) {
	return s.storage.List(ctx, in)
}

// Delete removes a template.
func (s *Service) Delete(ctx context.Context, id string) error {
	if err := s.storage.Delete(ctx, id); err != nil {
		s.logger.ErrorContext(ctx, "failed to delete template",
			slog.String("id", id),
			slogx.Error(err))
		return err
	}
	s.logger.InfoContext(ctx, "template deleted", slog.String("id", id))
	return nil
}

// BulkCreate inserts every item as a fresh template.
func (s *Service) BulkCreate(ctx context.Context, in []*entities.MessageTemplateCreate) (int, error) {
	created := 0
	for _, c := range in {
		_ = normalizer.Normalize(c) //nolint:errcheck // canonical: normalize tags can't fail on a well-formed DTO
		if strings.TrimSpace(c.Name) == "" {
			return created, errs.ErrMessageTemplateNameRequired
		}
		t := converter.Convert(c, entities.MessageTemplateNew())
		if err := s.storage.Save(ctx, t); err != nil {
			s.logger.ErrorContext(ctx, "bulk create: failed to save template",
				slog.Int("created", created),
				slogx.Error(err))
			return created, err
		}
		created++
	}
	s.logger.InfoContext(ctx, "templates bulk created", slog.Int("count", created))
	return created, nil
}

// DeleteAll removes every stored template.
func (s *Service) DeleteAll(ctx context.Context) (int64, error) {
	n, err := s.storage.DeleteAll(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to delete all templates", slogx.Error(err))
		return 0, err
	}
	s.logger.InfoContext(ctx, "templates cleared", slog.Int64("deleted", n))
	return n, nil
}

// Compile-time interface check.
var _ templatessvc.Service = (*Service)(nil)
