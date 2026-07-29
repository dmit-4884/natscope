// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package proto

import (
	"context"
	"log/slog"

	"github.com/altessa-s/go-atlas/core/collections/slices"
	"github.com/altessa-s/go-atlas/core/runtime/panics"

	"github.com/dmit-4884/natscope/internal/entities"

	corecontext "github.com/altessa-s/go-atlas/core/context"
	slogx "github.com/altessa-s/go-atlas/observability/slog"
)

// onFileWatcherChange is the debounced local-source change handler; the
// per-source lock stops overlapping bursts from persisting stale snapshots.
func (s *Service) onFileWatcherChange(ctx context.Context, sourceID string) {
	ctx, cancel := corecontext.ApplyTimeout(ctx, fileWatcherDebounceTimeout)
	defer cancel()
	defer panics.Handle(ctx)

	source, err := s.sourcesStorage.Get(ctx, sourceID, false)
	if err != nil {
		s.logger.Error("filewatcher: failed to get source",
			slog.String("source_id", sourceID), slogx.Error(err))
		return
	}
	if !source.Enabled || !source.WatcherEnabled {
		return
	}
	if source.LocalPath == nil || *source.LocalPath == "" {
		return
	}

	if _, diags, err := s.compileLocalLocked(ctx, source); err != nil {
		s.logger.Error("filewatcher: recompile failed",
			slog.String("source_id", sourceID), slogx.Error(err))
	} else if slices.Any(diags, func(d entities.CompileDiagnostic) bool {
		return d.Severity == entities.DiagnosticError
	}) {
		s.logger.Warn("filewatcher: recompile produced diagnostics",
			slog.String("source_id", sourceID))
	}
}
