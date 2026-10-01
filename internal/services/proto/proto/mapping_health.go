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
		h, err := s.computeOneHealth(ctx, id)
		if err != nil {
			return nil, err
		}
		out[i] = h
	}
	return out, nil
}

func (s *Service) computeOneHealth(ctx context.Context, id string) (entities.SubjectMappingHealth, error) {
	res := entities.SubjectMappingHealth{Id: id, Health: entities.MappingHealthOK}

	m, err := s.mappingsService.Get(ctx, id)
	if err != nil {
		if !errors.Is(err, errs.ErrMappingNotFound) {
			return res, err
		}
		res.Health = entities.MappingHealthMappingMissing
		res.Detail = "mapping not found"
		return res, nil
	}

	if m.SourceID == "" {
		res.Health = entities.MappingHealthSourceMissing
		res.Detail = "mapping has no source_id"
		return res, nil
	}

	src, err := s.sourcesStorage.Get(ctx, m.SourceID, false)
	if err != nil {
		res.Health = entities.MappingHealthSourceMissing
		res.Detail = fmt.Sprintf("source %q not found", m.SourceID)
		return res, nil
	}
	if !src.Enabled {
		res.Health = entities.MappingHealthSourceDisabled
		res.Detail = fmt.Sprintf("source %q is disabled", src.Name)
		return res, nil
	}

	version := mappingVersionLabel(m)
	snap, err := s.resolveDescriptorForMapping(ctx, m)
	if err != nil {
		switch {
		case errors.Is(err, errs.ErrMappingSelectionMissing):
			res.Health = entities.MappingHealthSelectionMissing
			res.Detail = fmt.Sprintf("source %q has no selected version", src.Name)
		case errors.Is(err, errs.ErrMappingSourceDisabled):
			res.Health = entities.MappingHealthSourceDisabled
			res.Detail = fmt.Sprintf("source %q is disabled", src.Name)
		case errors.Is(err, errs.ErrMappingSourceNotFound):
			res.Health = entities.MappingHealthSourceMissing
			res.Detail = fmt.Sprintf("source %q not found", m.SourceID)
		default:
			res.Health = entities.MappingHealthDescriptorMissing
			res.Detail = fmt.Sprintf("no compiled descriptor for source %q at %s", src.Name, version)
		}
		return res, nil
	}
	if _, ok := snap.Schema.Messages[m.MessageType]; !ok {
		res.Health = entities.MappingHealthTypeMissing
		res.Detail = fmt.Sprintf("type %q not present in source %q at %s", m.MessageType, src.Name, version)
	}
	return res, nil
}

// mappingVersionLabel describes the version a mapping decodes with, matching
// resolveDescriptorForMapping's fingerprint -> tag -> active order.
func mappingVersionLabel(m *entities.SubjectMapping) string {
	switch {
	case m.PinnedFingerprint != nil && *m.PinnedFingerprint != "":
		return fmt.Sprintf("pinned fingerprint %q", *m.PinnedFingerprint)
	case m.PinnedTag != nil && *m.PinnedTag != "":
		return fmt.Sprintf("pinned tag %q", *m.PinnedTag)
	default:
		return "the active version"
	}
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
