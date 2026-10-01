// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package proto

import (
	"context"
	"log/slog"

	"github.com/altessa-s/go-atlas/core/runtime/panics"

	"github.com/dmit-4884/natscope/internal/entities"

	corecontext "github.com/altessa-s/go-atlas/core/context"
	slogx "github.com/altessa-s/go-atlas/observability/slog"
)

// Start restores file watchers, then builds missing schemas and refreshes tracked branches in the background.
func (s *Service) Start(ctx context.Context) {
	sources, err := s.allSources(ctx)
	if err != nil {
		s.logger.WarnContext(ctx, "startup: list sources failed", slogx.Error(err))
		return
	}
	s.RestoreWatchers(ctx)

	bg := context.WithoutCancel(ctx)
	go func() {
		defer panics.Handle(bg)
		for _, src := range sources {
			if src.Enabled {
				s.warmUp(bg, src)
			}
		}
	}()
}

func (s *Service) warmUp(ctx context.Context, src *entities.ProtoSource) {
	ctx, cancel := corecontext.ApplyTimeout(ctx, startupCompileTimeout)
	defer cancel()

	tracksRef := src.SourceType == entities.SourceTypeGit || src.SourceType == entities.SourceTypeBSR
	if tracksRef && src.SelectedRef != nil && src.SelectedRef.Kind.Movable() {
		s.logRefresh(ctx, src)
		return
	}
	revision, err := activeRevision(src)
	if err != nil {
		return
	}
	if ok, err := s.hasSchema(ctx, src.Id, revision); err != nil || ok {
		return
	}
	if tracksRef {
		if _, _, err := s.activateRef(ctx, src.Id, *src.SelectedRef); err != nil {
			s.logger.WarnContext(ctx, "startup: build source schema failed", slog.String("source_id", src.Id), slogx.Error(err))
		}
		return
	}
	s.logRefresh(ctx, src)
}

func (s *Service) logRefresh(ctx context.Context, src *entities.ProtoSource) {
	if _, _, err := s.RefreshSource(ctx, src.Id); err != nil {
		s.logger.WarnContext(ctx, "startup: refresh source failed", slog.String("source_id", src.Id), slogx.Error(err))
	}
}
