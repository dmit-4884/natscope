// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package bbolt

import "github.com/dmit-4884/natscope/internal/pkg/bbstore"

// layoutDoc is the persistence model of one connection's sidebar layout; the
// document id is the connection id.
type layoutDoc struct {
	bbstore.Base

	Streams *sectionDoc `json:"streams,omitempty"`
	KV      *sectionDoc `json:"kv,omitempty"`
	Objects *sectionDoc `json:"objects,omitempty"`
}

type sectionDoc struct {
	Pinned []string `json:"pinned,omitempty"`
	Order  []string `json:"order,omitempty"`
}
