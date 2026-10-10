// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package bbolt

import "github.com/dmit-4884/natscope/internal/pkg/bbstore"

// mappingDoc is the persistence model for a subject mapping.
type mappingDoc struct {
	bbstore.Base

	Pattern           string     `json:"pattern"`
	MessageType       string     `json:"messageType,omitempty"`
	SourceID          string     `json:"sourceId"`
	PinnedFingerprint *string    `json:"pinnedFingerprint,omitempty"`
	Framing           framingDoc `json:"framing,omitzero"`
}

type framingDoc struct {
	Kind     string `json:"kind,omitempty"`
	SchemaID int32  `json:"schemaId,omitempty"`
	Prefix   []byte `json:"prefix,omitempty"`
	Suffix   []byte `json:"suffix,omitempty"`
}
