// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package proto

import (
	"context"
	"errors"
	"log/slog"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/services/proto/registry"

	slogx "github.com/altessa-s/go-atlas/observability/slog"
)

func (s *Service) mappingSource(ctx context.Context, sourceID string) (*entities.ProtoSource, error) {
	src, err := s.sourcesStorage.Get(ctx, sourceID, false)
	if errors.Is(err, errs.ErrProtoSourceNotFound) {
		return nil, errs.ErrMappingSourceNotFound
	}
	return src, err
}

func (s *Service) snapshotForSource(ctx context.Context, sourceID string) (*registry.Snapshot, error) {
	return s.snapshotAt(ctx, sourceID, "")
}

func (s *Service) snapshotAt(ctx context.Context, sourceID, fingerprint string) (*registry.Snapshot, error) {
	src, err := s.mappingSource(ctx, sourceID)
	if err != nil {
		return nil, err
	}
	if !src.Enabled {
		return nil, errs.ErrMappingSourceDisabled
	}
	if fingerprint != "" {
		return s.registryCache.GetByFingerprint(ctx, sourceID, fingerprint)
	}
	revision, err := activeRevision(src)
	if err != nil {
		return nil, err
	}
	return s.registryCache.GetOrBuild(ctx, sourceID, revision)
}

func (s *Service) snapshotForRequest(ctx context.Context, req entities.CodecRequest) (*registry.Snapshot, error) {
	if req.SourceID == "" {
		return nil, errs.ErrMappingSourceIDRequired
	}
	return s.snapshotAt(ctx, req.SourceID, req.Fingerprint)
}

func (s *Service) resolveDescriptorForMapping(
	ctx context.Context,
	m *entities.SubjectMapping,
) (*registry.Snapshot, error) {
	if m == nil {
		return nil, errs.ErrMappingNotFound
	}
	if m.SourceID == "" {
		return nil, errs.ErrMappingSourceIDRequired
	}
	pin := ""
	if m.PinnedFingerprint != nil {
		pin = *m.PinnedFingerprint
	}
	return s.snapshotAt(ctx, m.SourceID, pin)
}

func (s *Service) activeSnapshots(ctx context.Context) []*registry.Snapshot {
	sources, err := s.allSources(ctx)
	if err != nil {
		s.logger.WarnContext(ctx, "list sources failed", slogx.Error(err))
		return nil
	}

	out := make([]*registry.Snapshot, 0, len(sources))
	for _, src := range sources {
		if !src.Enabled {
			continue
		}
		snap, err := s.snapshotForSource(ctx, src.Id)
		if err != nil {
			s.logger.DebugContext(ctx, "snapshot unavailable for source",
				slog.String("source_id", src.Id),
				slogx.Error(err))
			continue
		}
		out = append(out, snap)
	}
	return out
}
