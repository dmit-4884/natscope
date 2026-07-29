// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package bbolt is the bbolt (doc-model) implementation of subject-mapping
// storage. Each mapping is one JSON document; uniqueness is (pattern, sourceId).
package bbolt

import (
	"context"

	"github.com/altessa-s/go-atlas/core/types/ptr"
	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/bbstore"

	storage "github.com/dmit-4884/natscope/internal/storages/mappings"
)

// Storage is the doc-model bbolt subject-mapping store.
type Storage struct {
	store *bbstore.Store[mappingDoc, *mappingDoc]
}

// New opens the subject_mappings bucket (creating it if needed) and returns the
// store.
func New(ctx context.Context, db *bbstore.DB) (*Storage, error) {
	store, err := bbstore.Open[mappingDoc, *mappingDoc](ctx, db, bbstore.Spec{
		Bucket:   "subject_mappings",
		NotFound: errs.ErrMappingNotFound,
	})
	if err != nil {
		return nil, err
	}
	return &Storage{store: store}, nil
}

// Save upserts the mapping by id, guarding (pattern, sourceId) uniqueness.
func (s *Storage) Save(ctx context.Context, in *entities.SubjectMapping) error {
	return s.store.WithTransaction(ctx, func(ctx context.Context) error {
		if err := s.assertPatternFree(ctx, in.Pattern, in.SourceID, in.Id); err != nil {
			return err
		}
		return s.store.Upsert(ctx, converter.Convert(in, &mappingDoc{}, bbstore.Opts()...))
	})
}

func (s *Storage) Get(ctx context.Context, id string) (*entities.SubjectMapping, error) {
	d, err := s.store.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return converter.Convert(d, &entities.SubjectMapping{}, bbstore.Opts()...), nil
}

func (s *Storage) List(
	ctx context.Context,
	in *entities.SubjectMappingsList,
) (*entities.List[entities.SubjectMappings], error) {
	docs, next, total, err := s.store.List(ctx, in.Cursor, in.GetLimit(), in.IncludeTotalCount)
	if err != nil {
		return nil, err
	}
	out := &entities.List[entities.SubjectMappings]{
		Items: bbstore.ToEntities[entities.SubjectMapping](docs),
		Total: total,
	}
	if next != "" {
		out.NextCursor = ptr.Wrap(next)
	}
	return out, nil
}

func (s *Storage) ListAll(ctx context.Context) (entities.SubjectMappings, error) {
	docs, err := s.store.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	return bbstore.ToEntities[entities.SubjectMapping](docs), nil
}

func (s *Storage) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

func (s *Storage) Exists(ctx context.Context, id string) (bool, error) {
	return s.store.Exists(ctx, id)
}

// BulkSave replaces the full (pattern, sourceId) set atomically: merges by
// key (keeps id/created_at), then deletes rows absent from the new set.
func (s *Storage) BulkSave(
	ctx context.Context,
	mappings entities.SubjectMappings,
) (*entities.SubjectMappingBulkSaveResult, error) {
	result := &entities.SubjectMappingBulkSaveResult{}
	err := s.store.WithTransaction(ctx, func(ctx context.Context) error {
		existing, err := s.store.ListAll(ctx)
		if err != nil {
			return err
		}
		// Non-nil map required: entries are added below, so maps.FromSlice
		// (nil on empty) is unsafe here.
		byKey := make(map[string]*mappingDoc, len(existing))
		for _, d := range existing {
			byKey[patternSourceKey(d.Pattern, d.SourceID)] = d
		}

		keep := make(map[string]bool, len(mappings))
		for _, m := range mappings {
			key := patternSourceKey(m.Pattern, m.SourceID)
			keep[key] = true
			d := converter.Convert(m, &mappingDoc{}, bbstore.Opts()...)
			if prev, ok := byKey[key]; ok {
				d.ID = prev.ID
				d.CreatedAt = prev.CreatedAt
				result.Updated++
			} else {
				result.Created++
			}
			if err := s.store.Upsert(ctx, d); err != nil {
				return err
			}
			byKey[key] = d
		}

		for key, d := range byKey {
			if keep[key] {
				continue
			}
			if err := s.store.Delete(ctx, d.ID); err != nil {
				return err
			}
			result.Deleted++
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// patternSourceKey is the (pattern, sourceId) uniqueness key.
func patternSourceKey(pattern, sourceID string) string {
	return pattern + "\x00" + sourceID
}

func (s *Storage) assertPatternFree(ctx context.Context, pattern, sourceID, excludeID string) error {
	existing, err := s.findByPatternSource(ctx, pattern, sourceID)
	if err != nil {
		return err
	}
	if existing != nil && existing.ID != excludeID {
		return errs.ErrMappingPatternAlreadyInUse
	}
	return nil
}

// findByPatternSource returns the mapping for (pattern, sourceId), or nil.
func (s *Storage) findByPatternSource(ctx context.Context, pattern, sourceID string) (*mappingDoc, error) {
	same, err := s.store.ListBy(ctx, "$.pattern", pattern)
	if err != nil {
		return nil, err
	}
	for _, m := range same {
		if m.SourceID == sourceID {
			return m, nil
		}
	}
	return nil, nil //nolint:nilnil // (nil, nil) is the documented "not found" result
}

var _ storage.Storage = (*Storage)(nil)
