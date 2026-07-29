// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package bbolt is the bbolt (doc-model) implementation of user-settings
// storage. Settings are a single document keyed by a fixed id; no secrets.
package bbolt

import (
	"context"

	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/bbstore"

	storage "github.com/dmit-4884/natscope/internal/storages/settings"
)

// settingsID is the fixed key of the singleton settings document.
const settingsID = "user-settings"

// Storage is the doc-model bbolt user-settings store (a single document).
type Storage struct {
	store *bbstore.Store[settingsDoc, *settingsDoc]
}

// New opens the user_settings bucket (creating it if needed) and returns the
// store.
func New(ctx context.Context, db *bbstore.DB) (*Storage, error) {
	store, err := bbstore.Open[settingsDoc, *settingsDoc](ctx, db, bbstore.Spec{
		Bucket:   "user_settings",
		NotFound: errs.ErrSettingsNotFound,
	})
	if err != nil {
		return nil, err
	}
	return &Storage{store: store}, nil
}

// Save upserts the single settings document under its fixed key.
func (s *Storage) Save(ctx context.Context, in *entities.UserSettings) error {
	d := converter.Convert(in, &settingsDoc{}, bbstore.Opts()...)
	d.ID = settingsID
	return s.store.Upsert(ctx, d)
}

func (s *Storage) Get(ctx context.Context) (*entities.UserSettings, error) {
	d, err := s.store.Get(ctx, settingsID)
	if err != nil {
		return nil, err
	}
	return converter.Convert(d, &entities.UserSettings{}, bbstore.Opts()...), nil
}

func (s *Storage) Delete(ctx context.Context) error {
	return s.store.Delete(ctx, settingsID)
}

var _ storage.Storage = (*Storage)(nil)
