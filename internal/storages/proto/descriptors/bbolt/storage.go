// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

// Package bbolt is the bbolt implementation of compiled proto schema storage.
package bbolt

import (
	"context"

	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/bbstore"

	storage "github.com/dmit-4884/natscope/internal/storages/proto/descriptors"
)

// Storage is the bbolt proto schema store.
type Storage struct {
	store *bbstore.Store[descriptorDoc, *descriptorDoc]
}

// New opens the proto_schemas bucket.
func New(ctx context.Context, db *bbstore.DB) (*Storage, error) {
	store, err := bbstore.Open[descriptorDoc, *descriptorDoc](ctx, db, bbstore.Spec{
		Bucket:   "proto_schemas",
		NotFound: errs.ErrProtoDescriptorNotFound,
		Indexes: []bbstore.Index{
			{Path: "sourceId"},
			{Path: "fingerprint"},
		},
	})
	if err != nil {
		return nil, err
	}
	return &Storage{store: store}, nil
}

// Save stores a schema, replacing the one for the same source and revision.
func (s *Storage) Save(ctx context.Context, in *entities.ProtoDescriptor) error {
	return s.store.WithTransaction(ctx, func(ctx context.Context) error {
		existing, err := s.find(ctx, in.SourceID, func(d *descriptorDoc) bool { return d.Revision == in.Revision })
		if err != nil {
			return err
		}
		if existing != nil {
			if err := s.store.Delete(ctx, existing.ID); err != nil {
				return err
			}
		}
		return s.store.Save(ctx, converter.Convert(in, &descriptorDoc{}, bbstore.Opts()...))
	})
}

// GetBySourceRevision returns errs.ErrProtoDescriptorNotFound when absent.
func (s *Storage) GetBySourceRevision(ctx context.Context, sourceID, revision string) (*entities.ProtoDescriptor, error) {
	return s.get(ctx, sourceID, func(d *descriptorDoc) bool { return d.Revision == revision })
}

// GetByFingerprint returns errs.ErrProtoDescriptorNotFound when absent.
func (s *Storage) GetByFingerprint(ctx context.Context, sourceID, fingerprint string) (*entities.ProtoDescriptor, error) {
	if fingerprint == "" {
		return nil, errs.ErrProtoDescriptorNotFound
	}
	docs, err := s.store.ListBy(ctx, "$.fingerprint", fingerprint)
	if err != nil {
		return nil, err
	}
	for _, d := range docs {
		if d.SourceID == sourceID {
			return bbstore.ToEntity[entities.ProtoDescriptor](d), nil
		}
	}
	return nil, errs.ErrProtoDescriptorNotFound
}

// DeleteBySourceRevision deletes one stored schema; a missing one is not an error.
func (s *Storage) DeleteBySourceRevision(ctx context.Context, sourceID, revision string) error {
	d, err := s.find(ctx, sourceID, func(d *descriptorDoc) bool { return d.Revision == revision })
	if err != nil || d == nil {
		return err
	}
	return s.store.Delete(ctx, d.ID)
}

// ListBySource returns every stored schema of a source.
func (s *Storage) ListBySource(ctx context.Context, sourceID string) (entities.ProtoDescriptors, error) {
	docs, err := s.store.ListBy(ctx, "$.sourceId", sourceID)
	if err != nil {
		return nil, err
	}
	return bbstore.ToEntities[entities.ProtoDescriptor](docs), nil
}

// DeleteBySource deletes every stored schema of a source.
func (s *Storage) DeleteBySource(ctx context.Context, sourceID string) (int64, error) {
	return s.store.DeleteBy(ctx, "$.sourceId", sourceID)
}

func (s *Storage) get(ctx context.Context, sourceID string, match func(*descriptorDoc) bool) (*entities.ProtoDescriptor, error) {
	d, err := s.find(ctx, sourceID, match)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, errs.ErrProtoDescriptorNotFound
	}
	return bbstore.ToEntity[entities.ProtoDescriptor](d), nil
}

func (s *Storage) find(ctx context.Context, sourceID string, match func(*descriptorDoc) bool) (*descriptorDoc, error) {
	docs, err := s.store.ListBy(ctx, "$.sourceId", sourceID)
	if err != nil {
		return nil, err
	}
	for _, d := range docs {
		if match(d) {
			return d, nil
		}
	}
	return nil, nil //nolint:nilnil // nil, nil means not found
}

var _ storage.Storage = (*Storage)(nil)
