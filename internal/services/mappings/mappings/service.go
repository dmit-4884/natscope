// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package mappings

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/altessa-s/go-atlas/domain/converter"
	"github.com/altessa-s/go-atlas/domain/normalizer"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/natsutil"

	slogx "github.com/altessa-s/go-atlas/observability/slog"
	mappingsvc "github.com/dmit-4884/natscope/internal/services/mappings"
	storage "github.com/dmit-4884/natscope/internal/storages/mappings"
)

// Service implements mappings.Service. Maintains a cached MappingResolver
// callers can read lock-free; swapped atomically after every mutation.
type Service struct {
	storage  storage.Storage
	logger   *slog.Logger
	resolver atomic.Pointer[natsutil.MappingResolver]
}

// New creates a new mappings service. Loads existing mappings from storage and
// builds the initial resolver eagerly.
func New(st storage.Storage) *Service {
	s := &Service{
		storage: st,
		logger:  slog.Default().With(slogx.Module("service:mappings")),
	}
	// Best-effort initial load; empty storage still yields a working resolver.
	s.rebuildResolver(context.Background())
	return s
}

// Create creates a new subject mapping.
func (s *Service) Create(
	ctx context.Context,
	in *entities.SubjectMappingCreate,
) (*entities.SubjectMapping, error) {
	_ = normalizer.Normalize(in) //nolint:errcheck // canonical: normalize tags can't fail on a well-formed DTO

	if in.SourceID == "" {
		return nil, errs.ErrMappingSourceIDRequired
	}

	mapping := converter.Convert(in, entities.SubjectMappingNew())

	if err := s.storage.Save(ctx, mapping); err != nil {
		s.logger.ErrorContext(ctx, "failed to save mapping",
			slog.String("pattern", in.Pattern),
			slog.String("source_id", in.SourceID),
			slogx.Error(err))
		return nil, err
	}

	s.logger.InfoContext(ctx, "mapping created",
		slog.String("id", mapping.Id),
		slog.String("pattern", in.Pattern),
		slog.String("message_type", in.MessageType),
		slog.String("source_id", in.SourceID))

	s.rebuildResolver(ctx)
	return mapping, nil
}

// Get retrieves a mapping by Id.
func (s *Service) Get(ctx context.Context, id string) (*entities.SubjectMapping, error) {
	return s.storage.Get(ctx, id)
}

// Update applies partial updates to an existing mapping.
func (s *Service) Update(
	ctx context.Context,
	in *entities.SubjectMappingUpdate,
) (*entities.SubjectMapping, error) {
	_ = normalizer.Normalize(in) //nolint:errcheck // canonical: normalize tags can't fail on a well-formed DTO

	existing, err := s.storage.Get(ctx, in.Id)
	if err != nil {
		return nil, err
	}

	existing.ApplyUpdate(in)

	// Validate after merge: SourceID must remain non-empty.
	if existing.SourceID == "" {
		return nil, errs.ErrMappingSourceIDRequired
	}

	if err := s.storage.Save(ctx, existing); err != nil {
		s.logger.ErrorContext(ctx, "failed to update mapping",
			slog.String("id", in.Id),
			slogx.Error(err))
		return nil, err
	}

	s.rebuildResolver(ctx)
	return existing, nil
}

// List returns mappings with pagination.
func (s *Service) List(
	ctx context.Context,
	in *entities.SubjectMappingsList,
) (*entities.List[entities.SubjectMappings], error) {
	return s.storage.List(ctx, in)
}

// Delete deletes a mapping.
func (s *Service) Delete(ctx context.Context, id string) error {
	if err := s.storage.Delete(ctx, id); err != nil {
		s.logger.ErrorContext(ctx, "failed to delete mapping",
			slog.String("id", id),
			slogx.Error(err))
		return err
	}

	s.logger.InfoContext(ctx, "mapping deleted",
		slog.String("id", id))

	s.rebuildResolver(ctx)
	return nil
}

// BulkSave replaces the full (pattern, source_id) set atomically, without
// mutating the input; mappings absent from the given set are deleted.
func (s *Service) BulkSave(
	ctx context.Context,
	mappings entities.SubjectMappings,
) (*entities.SubjectMappingBulkSaveResult, error) {
	now := time.Now().UTC()

	prepared := make(entities.SubjectMappings, len(mappings))
	for i, m := range mappings {
		if m.SourceID == "" {
			return nil, errs.ErrMappingSourceIDRequired
		}
		cp := *m
		if cp.Id == "" {
			cp.BaseEntity = *entities.New()
		}
		if cp.CreatedAt.IsZero() {
			cp.CreatedAt = now
		}
		prepared[i] = &cp
	}

	result, err := s.storage.BulkSave(ctx, prepared)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to bulk save mappings",
			slog.Int("count", len(prepared)),
			slogx.Error(err))
		return nil, err
	}

	s.logger.InfoContext(ctx, "mappings bulk saved",
		slog.Int("created", result.Created),
		slog.Int("updated", result.Updated),
		slog.Int("deleted", result.Deleted))

	s.rebuildResolver(ctx)
	return result, nil
}

// GetAll returns all mappings from storage.
func (s *Service) GetAll(ctx context.Context) (entities.SubjectMappings, error) {
	return s.storage.ListAll(ctx)
}

// Resolver returns the current resolver. The first call after construction or
// after a mutation triggers a rebuild if the cache is empty.
func (s *Service) Resolver(ctx context.Context) *natsutil.MappingResolver {
	if r := s.resolver.Load(); r != nil {
		return r
	}
	s.rebuildResolver(ctx)
	return s.resolver.Load()
}

// rebuildResolver reloads all mappings and atomically swaps the cached resolver.
func (s *Service) rebuildResolver(ctx context.Context) {
	all, err := s.storage.ListAll(ctx)
	if err != nil {
		s.logger.WarnContext(ctx, "failed to load mappings for resolver rebuild",
			slogx.Error(err))
		// Install an empty resolver so callers get deterministic nil-results
		// instead of a stale set.
		s.resolver.Store(natsutil.NewMappingResolver(nil))
		return
	}
	s.resolver.Store(natsutil.NewMappingResolver(all))
}

// Compile-time interface check.
var _ mappingsvc.Service = (*Service)(nil)
