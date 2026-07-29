// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package bbolt

import (
	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/pkg/bbstore"
	"github.com/dmit-4884/natscope/internal/pkg/secretsplit"
)

// toDoc converts the entity to its document and lifts the git token out for the
// vault; it is blanked on the document.
func toDoc(in *entities.ProtoSource) (*sourceDoc, map[string]string, error) {
	d := converter.Convert(in, &sourceDoc{}, bbstore.Opts()...)
	secs, err := secretsplit.Split(d)
	if err != nil {
		return nil, nil, err
	}
	return d, secs, nil
}

// fromDoc converts the document back and refills the git token from the vault.
func fromDoc(d *sourceDoc, secs map[string]string) (*entities.ProtoSource, error) {
	if err := secretsplit.Merge(d, secs); err != nil {
		return nil, err
	}
	return converter.Convert(d, &entities.ProtoSource{}, bbstore.Opts()...), nil
}
