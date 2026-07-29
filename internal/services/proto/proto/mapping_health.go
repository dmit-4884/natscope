// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package proto

import (
	"context"
	"errors"
	"fmt"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"

	protosvc "github.com/dmit-4884/natscope/internal/services/proto"
)

// MappingHealth computes resolvability state for the given mapping ids.
func (s *Service) MappingHealth(ctx context.Context, ids []string) ([]entities.SubjectMappingHealth, error) {
	out := make([]entities.SubjectMappingHealth, len(ids))
	for i, id := range ids {
		out[i] = s.computeOneHealth(ctx, id)
	}
	return out, nil
}

func (s *Service) computeOneHealth(ctx context.Context, id string) entities.SubjectMappingHealth {
	res := entities.SubjectMappingHealth{Id: id, Health: entities.MappingHealthOK}

	m, err := s.mappingsService.Get(ctx, id)
	if err != nil {
		res.Health = entities.MappingHealthSourceMissing
		res.Detail = "mapping not found"
		return res
	}

	if m.SourceID == "" {
		res.Health = entities.MappingHealthSourceMissing
		res.Detail = "mapping has no source_id"
		return res
	}

	src, err := s.sourcesStorage.Get(ctx, m.SourceID, false)
	if err != nil {
		res.Health = entities.MappingHealthSourceMissing
		res.Detail = fmt.Sprintf("source %q not found", m.SourceID)
		return res
	}
	if !src.Enabled {
		res.Health = entities.MappingHealthSourceDisabled
		res.Detail = fmt.Sprintf("source %q is disabled", src.Name)
		return res
	}

	tag, err := s.activeTagFor(ctx, m.SourceID)
	if err != nil {
		switch {
		case errors.Is(err, errs.ErrMappingSelectionMissing):
			res.Health = entities.MappingHealthSelectionMissing
			res.Detail = fmt.Sprintf("source %q has no selected version", src.Name)
		case errors.Is(err, errs.ErrMappingSourceDisabled):
			res.Health = entities.MappingHealthSourceDisabled
			res.Detail = fmt.Sprintf("source %q is disabled", src.Name)
		default:
			res.Health = entities.MappingHealthSourceMissing
			res.Detail = err.Error()
		}
		return res
	}

	snap, err := s.registryCache.GetOrBuild(ctx, m.SourceID, tag)
	if err != nil {
		res.Health = entities.MappingHealthDescriptorMissing
		res.Detail = fmt.Sprintf("no compiled descriptor for source %q at %q", src.Name, tag)
		return res
	}
	if _, ok := snap.Messages[m.MessageType]; !ok {
		res.Health = entities.MappingHealthTypeMissing
		res.Detail = fmt.Sprintf("type %q not present in source %q at %q", m.MessageType, src.Name, tag)
		return res
	}

	// Cross-source schema conflicts aren't a mapping health concern: each
	// mapping decodes via its own SourceID snapshot, unaffected by duplicates.

	return res
}

// ListSchemaConflicts returns cross-source schema conflicts from the most
// recent compile/reload.
func (s *Service) ListSchemaConflicts(ctx context.Context) (entities.SchemaConflicts, error) {
	if s.conflictsStorage == nil {
		return nil, nil
	}
	return s.conflictsStorage.GetAll(ctx)
}

// NewLiveDecoder creates a stateful decoder for live streams.
func (s *Service) NewLiveDecoder() protosvc.LiveDecoder {
	return &liveDecoder{service: s}
}
