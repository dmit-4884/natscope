// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package bbolt

import (
	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/pkg/bbstore"
	"github.com/dmit-4884/natscope/internal/pkg/secretsplit"
)

// toDoc converts the entity to its document and lifts the secret fields (NATS
// auth, TLS client key) out for the vault; they are blanked on the document.
func toDoc(in *entities.SavedConnection) (*connectionDoc, map[string]string, error) {
	d := converter.Convert(in, &connectionDoc{}, bbstore.Opts()...)
	secs, err := secretsplit.Split(d)
	if err != nil {
		return nil, nil, err
	}
	return d, secs, nil
}

// fromDoc refills the secret fields from the vault and converts back to the
// domain entity.
func fromDoc(d *connectionDoc, secs map[string]string) (*entities.SavedConnection, error) {
	if err := secretsplit.Merge(d, secs); err != nil {
		return nil, err
	}
	return converter.Convert(d, &entities.SavedConnection{}, bbstore.Opts()...), nil
}
