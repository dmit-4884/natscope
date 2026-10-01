// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package bbolt

import "github.com/dmit-4884/natscope/internal/pkg/bbstore"

type conflictDoc struct {
	bbstore.Base

	Kind     string       `json:"kind,omitempty"`
	Severity string       `json:"severity,omitempty"`
	Symbol   string       `json:"symbol,omitempty"`
	First    schemaRefDoc `json:"first"`
	Second   schemaRefDoc `json:"second"`
	Reason   string       `json:"reason,omitempty"`
}

type schemaRefDoc struct {
	SourceID string `json:"sourceId,omitempty"`
	Revision string `json:"revision,omitempty"`
	File     string `json:"file,omitempty"`
}
