// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package bbolt

import "github.com/dmit-4884/natscope/internal/pkg/bbstore"

// mappingDoc is the persistence model for a subject mapping.
type mappingDoc struct {
	bbstore.Base

	Pattern           string  `json:"pattern"`
	MessageType       string  `json:"messageType,omitempty"`
	SourceID          string  `json:"sourceId"`
	PinnedTag         *string `json:"pinnedTag,omitempty"`
	PinnedFingerprint *string `json:"pinnedFingerprint,omitempty"`
}
