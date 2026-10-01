// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package bbolt is the bbolt implementation of raw proto file storage.
package bbolt

import (
	"context"

	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/bbstore"

	storage "github.com/dmit-4884/natscope/internal/storages/proto/filesets"
)

// Storage is the bbolt raw proto file store.
type Storage struct {
	store *bbstore.Store[fileSetDoc, *fileSetDoc]
}

// New opens the proto_file_sets bucket.
func New(ctx context.Context, db *bbstore.DB) (*Storage, error) {
	store, err := bbstore.Open[fileSetDoc, *fileSetDoc](ctx, db, bbstore.Spec{
		Bucket:   "proto_file_sets",
		NotFound: errs.ErrProtoFileSetNotFound,
		Indexes:  []bbstore.Index{{Path: "sourceId"}},
	})
	if err != nil {
		return nil, err
	}
	return &Storage{store: store}, nil
}

// Save stores a file set, replacing the one for the same source and revision.
func (s *Storage) Save(ctx context.Context, in *entities.ProtoFileSet) error {
	return s.store.WithTransaction(ctx, func(ctx context.Context) error {
		existing, err := s.find(ctx, in.SourceID, in.Revision)
		if err != nil {
			return err
		}
		if existing != nil {
			if err := s.store.Delete(ctx, existing.ID); err != nil {
				return err
			}
		}
		return s.store.Save(ctx, converter.Convert(in, &fileSetDoc{}, bbstore.Opts()...))
	})
}

// GetBySourceRevision returns errs.ErrProtoFileSetNotFound when absent.
func (s *Storage) GetBySourceRevision(ctx context.Context, sourceID, revision string) (*entities.ProtoFileSet, error) {
	d, err := s.find(ctx, sourceID, revision)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, errs.ErrProtoFileSetNotFound
	}
	return converter.Convert(d, &entities.ProtoFileSet{}, bbstore.Opts()...), nil
}

// DeleteBySource deletes every file set of a source.
func (s *Storage) DeleteBySource(ctx context.Context, sourceID string) (int64, error) {
	return s.store.DeleteBy(ctx, "$.sourceId", sourceID)
}

func (s *Storage) find(ctx context.Context, sourceID, revision string) (*fileSetDoc, error) {
	docs, err := s.store.ListBy(ctx, "$.sourceId", sourceID)
	if err != nil {
		return nil, err
	}
	for _, d := range docs {
		if d.Revision == revision {
			return d, nil
		}
	}
	return nil, nil //nolint:nilnil // nil, nil means not found
}

var _ storage.Storage = (*Storage)(nil)
