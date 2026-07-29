// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package bbolt is the bbolt (doc-model) implementation of proto-source storage.
// One JSON document per source; the git token lives in the keychain vault, never
// in the database.
package bbolt

import (
	"context"
	"errors"

	"github.com/altessa-s/go-atlas/core/types/ptr"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/bbstore"
	"github.com/dmit-4884/natscope/internal/pkg/secrets"

	storage "github.com/dmit-4884/natscope/internal/storages/proto/sources"
)

// namespace addresses this domain's secrets in the vault.
const namespace = "proto_sources"

// Storage is the doc-model bbolt proto-source store.
type Storage struct {
	store *bbstore.Store[sourceDoc, *sourceDoc]
	vault secrets.Vault
}

// New opens the proto_sources bucket (creating it if needed) and returns the
// store.
func New(ctx context.Context, db *bbstore.DB, vault secrets.Vault) (*Storage, error) {
	store, err := bbstore.Open[sourceDoc, *sourceDoc](ctx, db, bbstore.Spec{
		Bucket:   "proto_sources",
		NotFound: errs.ErrProtoSourceNotFound,
		Indexes: []bbstore.Index{
			{Path: "name", Unique: true, ActiveOnly: true},
		},
	})
	if err != nil {
		return nil, err
	}
	return &Storage{store: store, vault: vault}, nil
}

func (s *Storage) Save(ctx context.Context, in *entities.ProtoSource) error {
	doc, secs, err := toDoc(in)
	if err != nil {
		return err
	}
	return s.store.WithTransaction(ctx, func(ctx context.Context) error {
		if err := s.store.Save(ctx, doc); err != nil {
			return mapUnique(err)
		}
		return s.vault.Put(ctx, namespace, in.Id, secs)
	})
}

func (s *Storage) Update(ctx context.Context, in *entities.ProtoSource) error {
	doc, secs, err := toDoc(in)
	if err != nil {
		return err
	}
	return s.store.WithTransaction(ctx, func(ctx context.Context) error {
		if err := s.store.Update(ctx, doc); err != nil {
			return mapUnique(err)
		}
		return s.vault.Put(ctx, namespace, in.Id, secs)
	})
}

func (s *Storage) Get(ctx context.Context, id string, includeDeleted ...bool) (*entities.ProtoSource, error) {
	d, err := s.store.Get(ctx, id, includeDeleted...)
	if err != nil {
		return nil, err
	}
	return s.hydrate(ctx, d)
}

func (s *Storage) GetByName(ctx context.Context, name string, includeDeleted ...bool) (*entities.ProtoSource, error) {
	d, err := s.store.GetBy(ctx, "$.name", name, includeDeleted...)
	if err != nil {
		return nil, err
	}
	return s.hydrate(ctx, d)
}

func (s *Storage) List(
	ctx context.Context,
	in *entities.ProtoSourcesList,
) (*entities.List[entities.ProtoSources], error) {
	docs, next, total, err := s.store.List(ctx, in.Cursor, in.GetLimit(), in.IncludeTotalCount)
	if err != nil {
		return nil, err
	}
	items := make(entities.ProtoSources, 0, len(docs))
	for _, d := range docs {
		e, herr := s.hydrate(ctx, d)
		if herr != nil {
			return nil, herr
		}
		items = append(items, e)
	}
	out := &entities.List[entities.ProtoSources]{Items: items, Total: total}
	if next != "" {
		out.NextCursor = ptr.Wrap(next)
	}
	return out, nil
}

// SoftDelete hides the source but keeps its token so it can be restored.
func (s *Storage) SoftDelete(ctx context.Context, in *entities.SoftDelete) error {
	return s.store.SoftDelete(ctx, in.Id, in.NewUpdatedAt.UnixMilli())
}

func (s *Storage) Exists(ctx context.Context, id string, includeDeleted ...bool) (bool, error) {
	return s.store.Exists(ctx, id, includeDeleted...)
}

func (s *Storage) hydrate(ctx context.Context, d *sourceDoc) (*entities.ProtoSource, error) {
	secs, err := s.vault.Get(ctx, namespace, d.ID)
	if err != nil {
		return nil, err
	}
	return fromDoc(d, secs)
}

// mapUnique turns the engine's unique-violation into this domain's sentinel so
// callers see a stable error regardless of which guard tripped.
func mapUnique(err error) error {
	if uv, _ := errors.AsType[*bbstore.ErrUniqueViolation](err); uv != nil {
		return errs.ErrProtoSourceNameAlreadyInUse
	}
	return err
}

var _ storage.Storage = (*Storage)(nil)
