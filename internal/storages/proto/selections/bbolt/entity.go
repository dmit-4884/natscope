// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package bbolt

import "github.com/dmit-4884/natscope/internal/pkg/bbstore"

// selectionDoc is the persistence model for a source's active-tag selection.
type selectionDoc struct {
	bbstore.Base

	SourceID string `json:"sourceId"`
	Tag      string `json:"tag"`
}
