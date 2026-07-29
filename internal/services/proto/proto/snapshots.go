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

// activeTagFor returns the active tag: LocalTag for local, FilesTag for files,
// the ProtoSelection tag for git (ErrMappingSelectionMissing if absent).
func (s *Service) activeTagFor(ctx context.Context, sourceID string) (string, error) {
	src, err := s.sourcesStorage.Get(ctx, sourceID, false)
	if err != nil {
		if errors.Is(err, errs.ErrProtoSourceNotFound) {
			return "", errs.ErrMappingSourceNotFound
		}
		return "", err
	}
	if !src.Enabled {
		return "", errs.ErrMappingSourceDisabled
	}

	if src.SourceType == entities.SourceTypeLocal {
		return LocalTag, nil
	}
	if src.SourceType == entities.SourceTypeFiles {
		return FilesTag, nil
	}

	sel, err := s.selectionsStorage.GetBySource(ctx, sourceID)
	if err != nil {
		if errors.Is(err, errs.ErrProtoSelectionNotFound) {
			return "", errs.ErrMappingSelectionMissing
		}
		return "", err
	}
	return sel.Tag, nil
}

// snapshotForSource returns the cached snapshot for the active tag of a source.
func (s *Service) snapshotForSource(ctx context.Context, sourceID string) (*registry.Snapshot, error) {
	tag, err := s.activeTagFor(ctx, sourceID)
	if err != nil {
		return nil, err
	}
	return s.registryCache.GetOrBuild(ctx, sourceID, tag)
}

// snapshotForRequest resolves a snapshot for a CodecRequest; empty Tag falls
// back to the active selection.
func (s *Service) snapshotForRequest(ctx context.Context, req entities.CodecRequest) (*registry.Snapshot, error) {
	if req.SourceID == "" {
		return nil, errs.ErrMappingSourceIDRequired
	}
	if req.Tag != "" {
		return s.registryCache.GetOrBuild(ctx, req.SourceID, req.Tag)
	}
	return s.snapshotForSource(ctx, req.SourceID)
}

// resolveDescriptorForMapping resolves a snapshot bound to the mapping's
// source: PinnedFingerprint -> PinnedTag -> active selection, in that order.
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
	if m.PinnedFingerprint != nil && *m.PinnedFingerprint != "" {
		// Always scope by mapping.SourceID — never resolve a fingerprint match from
		// another source (see cache.go GetByFingerprint).
		return s.registryCache.GetByFingerprint(ctx, m.SourceID, *m.PinnedFingerprint)
	}
	if m.PinnedTag != nil && *m.PinnedTag != "" {
		return s.registryCache.GetOrBuild(ctx, m.SourceID, *m.PinnedTag)
	}
	return s.snapshotForSource(ctx, m.SourceID)
}

// activeSnapshots returns one cached snapshot per enabled source; per-source
// errors are logged, never abort the aggregate.
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
