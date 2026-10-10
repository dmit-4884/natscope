// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package entities

import "time"

// ProtoFileSet holds the raw .proto files of a source at one revision.
type ProtoFileSet struct {
	BaseEntity

	SourceID string
	Revision string
	Files    []ProtoFileEntry
	// Configs are buf.yaml, buf.work.yaml and buf.lock files.
	Configs   []ProtoFileEntry
	FetchedAt time.Time
}

// ProtoFileSetNew creates a ProtoFileSet with a generated Id and timestamps.
func ProtoFileSetNew(init ...func(*ProtoFileSet)) *ProtoFileSet {
	v := &ProtoFileSet{
		BaseEntity: *New(),
	}

	if len(init) > 0 && init[0] != nil {
		init[0](v)
	}

	return v
}

// ProtoFileEntry is a single .proto file's content.
type ProtoFileEntry struct {
	// Path is the repo-relative path (e.g., "proto/orders/v1/order.proto").
	Path string

	Content string

	// Size is the file size in bytes.
	Size int64
}
