// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

// Package bbolt is the bbolt (doc-model) implementation of user-settings
// storage. Settings are a single document keyed by a fixed id; no secrets.
package bbolt

import (
	"context"
	"errors"

	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/bbstore"

	storage "github.com/dmit-4884/natscope/internal/storages/settings"
)

// settingsID is the fixed key of the settings document (entities.UserSettingsID).
const settingsID = entities.UserSettingsID

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

// Update loads settings (or a fresh default), applies mutate and persists the result in one transaction.
func (s *Storage) Update(
	ctx context.Context,
	mutate func(existing *entities.UserSettings),
) (*entities.UserSettings, error) {
	var result *entities.UserSettings
	err := s.store.WithTransaction(ctx, func(ctx context.Context) error {
		existing, err := s.Get(ctx)
		if err != nil {
			if !errors.Is(err, errs.ErrSettingsNotFound) {
				return err
			}
			existing = entities.UserSettingsNew()
		}

		mutate(existing)

		if err := s.Save(ctx, existing); err != nil {
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

func (s *Storage) Delete(ctx context.Context) error {
	return s.store.Delete(ctx, settingsID)
}

var _ storage.Storage = (*Storage)(nil)
