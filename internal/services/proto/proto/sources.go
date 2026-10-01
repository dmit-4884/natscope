// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package proto

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/altessa-s/go-atlas/core/types/ptr"
	"github.com/altessa-s/go-atlas/domain/converter"
	"github.com/altessa-s/go-atlas/domain/normalizer"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"

	slogx "github.com/altessa-s/go-atlas/observability/slog"
)

// listAllSourcesLimit pulls every source in one List call; bbolt holds the whole
// set in memory, so cursor pagination buys nothing here.
const listAllSourcesLimit int64 = 1_000_000

// allSources returns every source, bypassing the default 50-item page cap.
func (s *Service) allSources(ctx context.Context) (entities.ProtoSources, error) {
	list, err := s.sourcesStorage.List(ctx, &entities.ProtoSourcesList{
		ListBase: entities.ListBase{Limit: ptr.Wrap(listAllSourcesLimit)},
	})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// RestoreWatchers re-registers file watchers for enabled local sources; call on
// startup so .proto edits resume triggering recompiles after a restart.
func (s *Service) RestoreWatchers(ctx context.Context) {
	if s.fileWatcher == nil {
		return
	}
	sources, err := s.allSources(ctx)
	if err != nil {
		s.logger.WarnContext(ctx, "restore watchers: list sources failed", slogx.Error(err))
		return
	}
	for _, src := range sources {
		if src.SourceType != entities.SourceTypeLocal || !src.Enabled || !src.WatcherEnabled || src.LocalPath == nil {
			continue
		}
		if werr := s.fileWatcher.Watch(src.Id, *src.LocalPath); werr != nil {
			s.logger.WarnContext(ctx, "restore watchers: watch failed",
				slog.String("source_id", src.Id), slogx.Error(werr))
		}
	}
}

// CreateSource creates a new proto source for a user.
func (s *Service) CreateSource(ctx context.Context, in *entities.ProtoSourceCreate) (*entities.ProtoSource, error) {
	_ = normalizer.Normalize(in) //nolint:errcheck // canonical: normalize tags can't fail on a well-formed DTO

	source := converter.Convert(in, entities.ProtoSourceNew())

	if err := s.sourcesStorage.Save(ctx, source); err != nil {
		return nil, err
	}

	// Start filewatcher for local sources with watcher enabled.
	if s.fileWatcher != nil && source.SourceType == entities.SourceTypeLocal && source.WatcherEnabled && source.LocalPath != nil {
		_ = s.fileWatcher.Watch(source.Id, *source.LocalPath) //nolint:errcheck // best-effort
	}

	s.logger.InfoContext(ctx, "created proto source",
		slog.String("id", source.Id),
		slog.String("name", source.Name),
		slog.String("type", string(source.SourceType)))

	return source, nil
}

// GetSource retrieves a source by Id.
func (s *Service) GetSource(ctx context.Context, id string) (*entities.ProtoSource, error) {
	return s.sourcesStorage.Get(ctx, id, false)
}

// ListSources returns sources for a user with pagination.
func (s *Service) ListSources(
	ctx context.Context,
	in *entities.ProtoSourcesList,
) (*entities.List[entities.ProtoSources], error) {
	return s.sourcesStorage.List(ctx, in)
}

// UpdateSource updates a proto source.
func (s *Service) UpdateSource(ctx context.Context, in *entities.ProtoSourceUpdate) (*entities.ProtoSource, error) {
	_ = normalizer.Normalize(in) //nolint:errcheck // canonical: normalize tags can't fail on a well-formed DTO

	current, err := s.sourcesStorage.Get(ctx, in.Id, false)
	if err != nil {
		return nil, err
	}

	// Token wire semantics: nil = no change, empty string = remove.
	if in.Token != nil && *in.Token == "" {
		current.Token = nil
		in.Token = nil // Don't copy empty string into the entity's Token.
	}

	current.ApplyUpdate(in)

	if err := s.sourcesStorage.Update(ctx, current); err != nil {
		return nil, err
	}

	s.logger.InfoContext(ctx, "updated proto source",
		slog.String("id", current.Id),
		slog.String("name", current.Name))

	return current, nil
}

// DeleteSource soft-deletes a proto source and invalidates its cached snapshots
// so subsequent decodes fail-closed.
func (s *Service) DeleteSource(ctx context.Context, id string) error {
	now := time.Now().UTC()

	if err := s.sourcesStorage.SoftDelete(ctx, &entities.SoftDelete{
		Id:           id,
		NewUpdatedAt: now,
	}); err != nil {
		return err
	}

	if s.fileWatcher != nil {
		s.fileWatcher.Unwatch(id)
	}
	if _, err := s.descriptorsStorage.DeleteBySource(ctx, id); err != nil {
		s.logger.WarnContext(ctx, "delete source schemas failed", slog.String("id", id), slogx.Error(err))
	}
	if _, err := s.fileSetsStorage.DeleteBySource(ctx, id); err != nil {
		s.logger.WarnContext(ctx, "delete source files failed", slog.String("id", id), slogx.Error(err))
	}
	s.registryCache.InvalidateSource(id)
	s.notifyReload(ctx)

	s.logger.InfoContext(ctx, "deleted proto source", slog.String("id", id))

	return nil
}

// SetEnabled enables or disables a source.
func (s *Service) SetEnabled(ctx context.Context, sourceID string, enabled bool) (*entities.ProtoSource, error) {
	source, err := s.sourcesStorage.Get(ctx, sourceID, false)
	if err != nil {
		return nil, err
	}

	source.Enabled = enabled
	source.BeforeUpdate()

	if err := s.sourcesStorage.Update(ctx, source); err != nil {
		return nil, err
	}

	// Manage file watcher for local sources based on enabled state.
	if s.fileWatcher != nil && source.SourceType == entities.SourceTypeLocal {
		if !enabled {
			s.fileWatcher.Unwatch(sourceID)
		} else if source.WatcherEnabled && source.LocalPath != nil {
			//nolint:errcheck // best-effort resume; user can toggle watcher to retry
			_ = s.fileWatcher.Watch(sourceID, *source.LocalPath)
		}
	}

	s.registryCache.InvalidateSource(sourceID)
	s.notifyReload(ctx)

	s.logger.InfoContext(ctx, "source enabled state changed",
		slog.String("id", sourceID),
		slog.Bool("enabled", enabled))

	return source, nil
}

// SetWatcher enables or disables file watcher for a local directory source.
func (s *Service) SetWatcher(ctx context.Context, sourceID string, enabled bool) (*entities.ProtoSource, error) {
	source, err := s.sourcesStorage.Get(ctx, sourceID, false)
	if err != nil {
		return nil, err
	}

	if source.SourceType != entities.SourceTypeLocal {
		return nil, fmt.Errorf("%w: file watcher is only available for local directory sources", errs.ErrInvalidRequest)
	}

	source.WatcherEnabled = enabled
	source.BeforeUpdate()

	if err := s.sourcesStorage.Update(ctx, source); err != nil {
		return nil, err
	}

	if s.fileWatcher != nil {
		if enabled && source.LocalPath != nil {
			_ = s.fileWatcher.Watch(sourceID, *source.LocalPath) //nolint:errcheck // best-effort: caller can re-toggle watcher
		} else {
			s.fileWatcher.Unwatch(sourceID)
		}
	}

	s.logger.InfoContext(ctx, "watcher state changed",
		slog.String("id", sourceID),
		slog.Bool("watcher_enabled", enabled))

	return source, nil
}

// ValidateRepository probes a Git repo under the token; outcome is on the
// returned entity, error reserved for genuine internal failures.
func (s *Service) ValidateRepository(
	ctx context.Context,
	repository string,
	token *string,
) (*entities.RepositoryValidation, error) {
	source := &entities.ProtoSource{Repository: repository, Token: token}
	if err := s.gitFetcher.ValidateRepository(ctx, source.AuthenticatedURL()); err != nil {
		msg := "Repository is not accessible: " + err.Error()
		return &entities.RepositoryValidation{Valid: false, Error: &msg}, nil
	}
	return &entities.RepositoryValidation{Valid: true}, nil
}
