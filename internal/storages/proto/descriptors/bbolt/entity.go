// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package bbolt

import "github.com/dmit-4884/natscope/internal/pkg/bbstore"

// descriptorDoc is the persistence model for a compiled descriptor.
// Fingerprint is the sha256 of the set, stored so pinned mappings can resolve by content hash.
type descriptorDoc struct {
	bbstore.Base

	SourceID      string   `json:"sourceId"`
	Tag           string   `json:"tag"`
	DescriptorSet []byte   `json:"descriptorSet,omitempty"`
	MessageTypes  []string `json:"messageTypes,omitempty"`
	CompiledAt    int64    `json:"compiledAt,omitempty"`
	Fingerprint   string   `json:"fingerprint,omitempty"`
}
