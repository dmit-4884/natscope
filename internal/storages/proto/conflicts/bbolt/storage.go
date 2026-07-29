// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package bbolt is the bbolt (doc-model) implementation of proto schema conflict
// storage. The whole list is rewritten on every reload, so there is no per-row
// update path.
package bbolt

import (
	"context"

	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/pkg/bbstore"

	storage "github.com/dmit-4884/natscope/internal/storages/proto/conflicts"
)

// Storage is the doc-model bbolt proto-conflict store.
type Storage struct {
	store *bbstore.Store[conflictDoc, *conflictDoc]
}

// New opens the proto_conflicts bucket (creating it if needed) and returns the
// store.
func New(ctx context.Context, db *bbstore.DB) (*Storage, error) {
	store, err := bbstore.Open[conflictDoc, *conflictDoc](ctx, db, bbstore.Spec{
		Bucket: "proto_conflicts",
	})
	if err != nil {
		return nil, err
	}
	return &Storage{store: store}, nil
}

// ReplaceAll overwrites the conflict list atomically.
func (s *Storage) ReplaceAll(ctx context.Context, conflicts entities.SchemaConflicts) error {
	return s.store.WithTransaction(ctx, func(ctx context.Context) error {
		if _, err := s.store.DeleteAll(ctx); err != nil {
			return err
		}
		for _, c := range conflicts {
			if err := s.store.Save(ctx, converter.Convert(c, &conflictDoc{}, bbstore.Opts()...)); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Storage) GetAll(ctx context.Context) (entities.SchemaConflicts, error) {
	docs, err := s.store.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	return bbstore.ToEntities[entities.SchemaConflict](docs), nil
}

var _ storage.Storage = (*Storage)(nil)
