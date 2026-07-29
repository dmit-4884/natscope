// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package proto

import (
	"context"
	"log/slog"

	"github.com/altessa-s/go-atlas/domain/converter"
	"github.com/altessa-s/go-atlas/domain/normalizer"

	"github.com/dmit-4884/natscope/internal/entities"
)

// Select selects a proto version (creates or updates); invalidates the registry
// cache so subsequent decodes pick up the new active selection.
func (s *Service) Select(ctx context.Context, in *entities.ProtoSelectionCreate) (*entities.ProtoSelection, error) {
	_ = normalizer.Normalize(in) //nolint:errcheck // canonical: normalize tags can't fail on a well-formed DTO

	if _, err := s.GetSource(ctx, in.SourceID); err != nil {
		return nil, err
	}

	_, err := s.FetchAndCompile(ctx, in.SourceID, in.Tag)
	if err != nil {
		return nil, err
	}

	existing, err := s.selectionsStorage.GetBySource(ctx, in.SourceID)
	if err == nil {
		// Update existing selection
		existing.Tag = in.Tag
		existing.BeforeUpdate()

		if err := s.selectionsStorage.Save(ctx, existing); err != nil {
			return nil, err
		}

		s.registryCache.InvalidateSource(in.SourceID)
		s.notifyReload(ctx)

		s.logger.InfoContext(ctx, "updated proto selection",
			slog.String("source_id", in.SourceID),
			slog.String("tag", in.Tag))

		return existing, nil
	}

	// Create new selection
	selection := converter.Convert(in, entities.ProtoSelectionNew())

	if err := s.selectionsStorage.Save(ctx, selection); err != nil {
		return nil, err
	}

	s.registryCache.InvalidateSource(in.SourceID)
	s.notifyReload(ctx)

	s.logger.InfoContext(ctx, "created proto selection",
		slog.String("source_id", in.SourceID),
		slog.String("tag", in.Tag))

	return selection, nil
}

// ListSelections returns all selections.
func (s *Service) ListSelections(ctx context.Context) (entities.ProtoSelections, error) {
	return s.selectionsStorage.GetAll(ctx)
}

// LoadAllSelections runs FetchAndCompile on every stored selection; per-
// selection failures are logged and counted, never abort (partial load is OK).
func (s *Service) LoadAllSelections(ctx context.Context) (*entities.ProtoLoadResult, error) {
	selections, err := s.ListSelections(ctx)
	if err != nil {
		return nil, err
	}

	result := &entities.ProtoLoadResult{}
	for _, selection := range selections {
		if _, compErr := s.FetchAndCompile(ctx, selection.SourceID, selection.Tag); compErr != nil {
			s.logger.WarnContext(ctx, "fetch/compile failed during selections load",
				slog.String("source_id", selection.SourceID),
				slog.String("tag", selection.Tag),
				slog.String("error", compErr.Error()))
			result.FailedCount++
			continue
		}
		result.CompiledCount++
	}

	result.MessageCount = len(s.ListMessages(ctx))
	return result, nil
}

// DeleteSelection removes a selection and invalidates the source's cached
// snapshot so subsequent decodes fail-closed with "selection missing".
func (s *Service) DeleteSelection(ctx context.Context, selectionID string) error {
	sel, getErr := s.selectionsStorage.Get(ctx, selectionID)
	if err := s.selectionsStorage.Delete(ctx, selectionID); err != nil {
		return err
	}
	if getErr == nil && sel != nil {
		s.registryCache.InvalidateSource(sel.SourceID)
	}
	s.notifyReload(ctx)
	return nil
}
