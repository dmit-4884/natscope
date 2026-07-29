// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package bbolt is the bbolt implementation of saved-connections storage. The
// whole connection is stored as one JSON document; secret fields (NATS auth, TLS
// client key) live in the keychain vault, never in the database.
package bbolt

import (
	"context"
	"errors"
	"maps"

	"github.com/altessa-s/go-atlas/core/types/ptr"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/bbstore"
	"github.com/dmit-4884/natscope/internal/pkg/secrets"

	storage "github.com/dmit-4884/natscope/internal/storages/connections"
)

// namespace addresses this domain's secrets in the vault.
const namespace = "connections"

// Storage is the doc-model bbolt connections store.
type Storage struct {
	store *bbstore.Store[connectionDoc, *connectionDoc]
	vault secrets.Vault
}

// New opens the connections bucket (creating it if needed) and returns the store.
func New(ctx context.Context, db *bbstore.DB, vault secrets.Vault) (*Storage, error) {
	store, err := bbstore.Open[connectionDoc, *connectionDoc](ctx, db, bbstore.Spec{
		Bucket:   "connections",
		NotFound: errs.ErrSavedConnectionNotFound,
		Indexes: []bbstore.Index{
			{Path: "name", Unique: true, ActiveOnly: true},
		},
	})
	if err != nil {
		return nil, err
	}
	return &Storage{store: store, vault: vault}, nil
}

func (s *Storage) Save(ctx context.Context, in *entities.SavedConnection) error {
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

func (s *Storage) Update(ctx context.Context, in *entities.SavedConnection) error {
	doc, secs, err := toDoc(in)
	if err != nil {
		return err
	}
	return s.store.WithTransaction(ctx, func(ctx context.Context) error {
		if err := s.store.Update(ctx, doc); err != nil {
			return mapUnique(err)
		}
		// Secrets omitted from the request are preserved: the API never returns
		// stored secret values, so an edit that leaves a secret field blank means
		// "keep the existing one" rather than "clear it".
		merged, err := s.mergeSecrets(ctx, in.Id, secs)
		if err != nil {
			return err
		}
		return s.vault.Put(ctx, namespace, in.Id, merged)
	})
}

// mergeSecrets overlays the incoming secrets onto the ones already in the vault,
// so unspecified secret fields retain their stored value on update.
func (s *Storage) mergeSecrets(ctx context.Context, id string, incoming map[string]string) (map[string]string, error) {
	existing, err := s.vault.Get(ctx, namespace, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		existing = make(map[string]string, len(incoming))
	}
	maps.Copy(existing, incoming)
	return existing, nil
}

func (s *Storage) Get(ctx context.Context, id string, includeDeleted ...bool) (*entities.SavedConnection, error) {
	doc, err := s.store.Get(ctx, id, includeDeleted...)
	if err != nil {
		return nil, err
	}
	return s.hydrate(ctx, doc)
}

func (s *Storage) List(
	ctx context.Context,
	in *entities.SavedConnectionsList,
) (*entities.List[entities.SavedConnections], error) {
	docs, next, total, err := s.store.List(ctx, in.Cursor, in.GetLimit(), in.IncludeTotalCount)
	if err != nil {
		return nil, err
	}
	items := make(entities.SavedConnections, 0, len(docs))
	for _, d := range docs {
		e, herr := s.hydrate(ctx, d)
		if herr != nil {
			return nil, herr
		}
		items = append(items, e)
	}
	out := &entities.List[entities.SavedConnections]{Items: items, Total: total}
	if next != "" {
		out.NextCursor = ptr.Wrap(next)
	}
	return out, nil
}

// SoftDelete hides the connection but keeps its secrets so it can be restored.
func (s *Storage) SoftDelete(ctx context.Context, in *entities.SoftDelete) error {
	return s.store.SoftDelete(ctx, in.Id, in.NewUpdatedAt.UnixMilli())
}

// Delete removes the connection and purges its secrets from the vault.
func (s *Storage) Delete(ctx context.Context, id string) error {
	return s.store.WithTransaction(ctx, func(ctx context.Context) error {
		if err := s.store.Delete(ctx, id); err != nil {
			return err
		}
		return s.vault.Delete(ctx, namespace, id)
	})
}

func (s *Storage) hydrate(ctx context.Context, doc *connectionDoc) (*entities.SavedConnection, error) {
	secs, err := s.vault.Get(ctx, namespace, doc.ID)
	if err != nil {
		return nil, err
	}
	return fromDoc(doc, secs)
}

// mapUnique turns the engine's unique-violation into this domain's sentinel so
// callers see a stable error regardless of which guard tripped.
func mapUnique(err error) error {
	if uv, _ := errors.AsType[*bbstore.ErrUniqueViolation](err); uv != nil {
		return errs.ErrConnectionNameAlreadyInUse
	}
	return err
}

var _ storage.Storage = (*Storage)(nil)
