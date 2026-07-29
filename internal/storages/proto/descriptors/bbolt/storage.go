// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package bbolt is the bbolt (doc-model) implementation of proto-descriptor
// storage. The FileDescriptorSet and message types are nested JSON; a
// fingerprint (sha256 of the set) enables pinned-snapshot lookup.
package bbolt

import (
	"context"

	"github.com/altessa-s/go-atlas/core/types/ptr"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/bbstore"

	storage "github.com/dmit-4884/natscope/internal/storages/proto/descriptors"
)

// Storage is the doc-model bbolt proto-descriptor store.
type Storage struct {
	store *bbstore.Store[descriptorDoc, *descriptorDoc]
}

// New opens the proto_descriptors bucket (creating it if needed) and returns the
// store.
func New(ctx context.Context, db *bbstore.DB) (*Storage, error) {
	store, err := bbstore.Open[descriptorDoc, *descriptorDoc](ctx, db, bbstore.Spec{
		Bucket:   "proto_descriptors",
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

// Save replaces any descriptor for the same (source, tag), so the new id and
// message types fully supersede the prior snapshot.
func (s *Storage) Save(ctx context.Context, in *entities.ProtoDescriptor) error {
	return s.store.WithTransaction(ctx, func(ctx context.Context) error {
		existing, err := s.docBySourceTag(ctx, in.SourceID, in.Tag)
		if err != nil {
			return err
		}
		if existing != nil {
			if err := s.store.Delete(ctx, existing.ID); err != nil {
				return err
			}
		}
		return s.store.Save(ctx, toDoc(in))
	})
}

func (s *Storage) GetById(ctx context.Context, id string) (*entities.ProtoDescriptor, error) {
	d, err := s.store.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return bbstore.ToEntity[entities.ProtoDescriptor](d), nil
}

func (s *Storage) GetBySourceTag(ctx context.Context, sourceID, tag string) (*entities.ProtoDescriptor, error) {
	d, err := s.docBySourceTag(ctx, sourceID, tag)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, errs.ErrProtoDescriptorNotFound
	}
	return bbstore.ToEntity[entities.ProtoDescriptor](d), nil
}

func (s *Storage) FindByFingerprint(ctx context.Context, fingerprint string) (*entities.ProtoDescriptor, error) {
	if fingerprint == "" {
		return nil, errs.ErrProtoDescriptorNotFound
	}
	d, err := s.store.GetBy(ctx, "$.fingerprint", fingerprint)
	if err != nil {
		return nil, err
	}
	return bbstore.ToEntity[entities.ProtoDescriptor](d), nil
}

func (s *Storage) GetAll(ctx context.Context) (entities.ProtoDescriptors, error) {
	docs, err := s.store.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	return bbstore.ToEntities[entities.ProtoDescriptor](docs), nil
}

func (s *Storage) List(
	ctx context.Context,
	in *entities.ProtoDescriptorsList,
) (*entities.List[entities.ProtoDescriptors], error) {
	var filters []bbstore.Filter
	if in.SourceID != nil {
		filters = append(filters, bbstore.Filter{Path: "$.sourceId", Val: *in.SourceID})
	}
	if in.Tag != nil {
		filters = append(filters, bbstore.Filter{Path: "$.tag", Val: *in.Tag})
	}
	docs, next, total, err := s.store.ListFiltered(ctx, filters, in.Cursor, in.GetLimit(), in.IncludeTotalCount)
	if err != nil {
		return nil, err
	}
	out := &entities.List[entities.ProtoDescriptors]{Items: bbstore.ToEntities[entities.ProtoDescriptor](docs), Total: total}
	if next != "" {
		out.NextCursor = ptr.Wrap(next)
	}
	return out, nil
}

func (s *Storage) Update(ctx context.Context, in *entities.ProtoDescriptor) error {
	return s.store.Update(ctx, toDoc(in))
}

func (s *Storage) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

func (s *Storage) DeleteBySource(ctx context.Context, sourceID string) (int64, error) {
	return s.store.DeleteBy(ctx, "$.sourceId", sourceID)
}

func (s *Storage) Exists(ctx context.Context, id string) (bool, error) {
	return s.store.Exists(ctx, id)
}

// docBySourceTag finds the descriptor for (sourceId, tag), or nil.
func (s *Storage) docBySourceTag(ctx context.Context, sourceID, tag string) (*descriptorDoc, error) {
	docs, err := s.store.ListBy(ctx, "$.sourceId", sourceID)
	if err != nil {
		return nil, err
	}
	for _, d := range docs {
		if d.Tag == tag {
			return d, nil
		}
	}
	return nil, nil //nolint:nilnil // (nil, nil) is the documented "not found" result
}

var _ storage.Storage = (*Storage)(nil)
