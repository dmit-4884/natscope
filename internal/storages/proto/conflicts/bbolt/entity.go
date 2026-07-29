// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package bbolt

import "github.com/dmit-4884/natscope/internal/pkg/bbstore"

// conflictDoc is the persistence model for one schema conflict. Winner and Loser
// refs are nested instead of flattened into winner_*/loser_* columns.
type conflictDoc struct {
	bbstore.Base

	Kind     string       `json:"kind,omitempty"`
	Severity string       `json:"severity,omitempty"`
	Symbol   string       `json:"symbol,omitempty"`
	Winner   schemaRefDoc `json:"winner"`
	Loser    schemaRefDoc `json:"loser"`
	Reason   string       `json:"reason,omitempty"`
	Policy   string       `json:"policy,omitempty"`
}

type schemaRefDoc struct {
	SourceID string `json:"sourceId,omitempty"`
	Tag      string `json:"tag,omitempty"`
	File     string `json:"file,omitempty"`
}
