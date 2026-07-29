// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package bbstore

// Base holds the fields every document shares. JSON tags match the old
// SQLite docstore format so stored documents move over without reshaping.
type Base struct {
	ID        string `json:"id"`
	Etag      string `json:"etag,omitempty"`
	CreatedAt int64  `json:"createdAt"`
	UpdatedAt int64  `json:"updatedAt"`
	DeletedAt *int64 `json:"deletedAt,omitempty"`
}

// DocBase exposes the embedded base to the generic store. It is promoted to any
// model that embeds Base, so models need no boilerplate.
func (b *Base) DocBase() *Base { return b }

// Doc constrains a persistence model: a pointer to M that carries a Base.
type Doc[M any] interface {
	*M
	DocBase() *Base
}
