// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package proto

import (
	"context"
	"log/slog"
	"slices"

	"github.com/altessa-s/go-atlas/core/runtime/panics"

	"github.com/dmit-4884/natscope/internal/entities"

	corecontext "github.com/altessa-s/go-atlas/core/context"
	slogx "github.com/altessa-s/go-atlas/observability/slog"
)

// Start drops sources of retired types, restores file watchers, then builds missing schemas and refreshes tracked
// branches in the background.
func (s *Service) Start(ctx context.Context) {
	sources, err := s.allSources(ctx)
	if err != nil {
		s.logger.WarnContext(ctx, "startup: list sources failed", slogx.Error(err))
		return
	}
	sources = slices.DeleteFunc(sources, func(src *entities.ProtoSource) bool { return s.dropRetired(ctx, src) })
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

func (s *Service) dropRetired(ctx context.Context, src *entities.ProtoSource) bool {
	switch src.SourceType {
	case entities.SourceTypeGit, entities.SourceTypeLocal, entities.SourceTypeUpload, entities.SourceTypeBSR:
		return false
	default:
	}
	if err := s.DeleteSource(ctx, src.Id); err != nil {
		s.logger.WarnContext(ctx, "startup: delete retired source failed", slog.String("source_id", src.Id), slogx.Error(err))
		return true
	}
	s.logger.WarnContext(ctx, "startup: deleted a proto source of a retired type",
		slog.String("name", src.Name), slog.String("type", string(src.SourceType)))
	return true
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

// watchLocal starts watching a local source's directory and, for an enabled source without a schema, builds one in
// the background, so a watched source decodes without waiting for a file change, a refresh or a restart.
func (s *Service) watchLocal(ctx context.Context, src *entities.ProtoSource) {
	if s.fileWatcher != nil {
		_ = s.fileWatcher.Watch(src.Id, *src.LocalPath) //nolint:errcheck // best-effort: the caller can re-toggle the watcher
	}
	if !src.Enabled {
		return
	}
	bg := context.WithoutCancel(ctx)
	go func() {
		defer panics.Handle(bg)
		s.warmUp(bg, src)
	}()
}

func (s *Service) logRefresh(ctx context.Context, src *entities.ProtoSource) {
	if _, _, err := s.RefreshSource(ctx, src.Id); err != nil {
		s.logger.WarnContext(ctx, "startup: refresh source failed", slog.String("source_id", src.Id), slogx.Error(err))
	}
}
