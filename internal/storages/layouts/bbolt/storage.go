// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

// Package bbolt is the bbolt (doc-model) implementation of sidebar layout
// storage: one document per saved connection, keyed by the connection id.
package bbolt

import (
	"context"
	"errors"

	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/bbstore"

	storage "github.com/dmit-4884/natscope/internal/storages/layouts"
)

// Storage is the doc-model bbolt sidebar layout store.
type Storage struct {
	store *bbstore.Store[layoutDoc, *layoutDoc]
}

// New opens the sidebar_layouts bucket (creating it if needed) and returns the
// store.
func New(ctx context.Context, db *bbstore.DB) (*Storage, error) {
	store, err := bbstore.Open[layoutDoc, *layoutDoc](ctx, db, bbstore.Spec{
		Bucket:   "sidebar_layouts",
		NotFound: errs.ErrSidebarLayoutNotFound,
	})
	if err != nil {
		return nil, err
	}
	return &Storage{store: store}, nil
}

func (s *Storage) Get(ctx context.Context, connectionID string) (*entities.SidebarLayout, error) {
	d, err := s.store.Get(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	out := converter.Convert(d, &entities.SidebarLayout{}, bbstore.Opts()...)
	out.ConnectionID = d.ID
	return out, nil
}

// Update loads the layout (or an empty one), applies mutate and persists the
// result in one transaction.
func (s *Storage) Update(
	ctx context.Context,
	connectionID string,
	mutate func(existing *entities.SidebarLayout),
) (*entities.SidebarLayout, error) {
	var result *entities.SidebarLayout
	err := s.store.WithTransaction(ctx, func(ctx context.Context) error {
		existing, err := s.Get(ctx, connectionID)
		if err != nil {
			if !errors.Is(err, errs.ErrSidebarLayoutNotFound) {
				return err
			}
			existing = &entities.SidebarLayout{ConnectionID: connectionID}
		}

		mutate(existing)

		d := converter.Convert(existing, &layoutDoc{}, bbstore.Opts()...)
		d.ID = connectionID
		if err := s.store.Upsert(ctx, d); err != nil {
			return err
		}
		result = existing
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Storage) Delete(ctx context.Context, connectionID string) error {
	return s.store.Delete(ctx, connectionID)
}

var _ storage.Storage = (*Storage)(nil)
