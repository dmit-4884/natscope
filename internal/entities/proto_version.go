// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

import "time"

// ProtoVersion holds the raw .proto files of a tag fetched from Git, ready for
// compilation.
type ProtoVersion struct {
	BaseEntity

	SourceID string

	// Tag is the Git tag (e.g., "v0.72.0").
	Tag string

	// Files holds the raw .proto file contents.
	Files []ProtoFileEntry

	// Configs holds captured compile-affecting files (buf.yaml/work.yaml/lock);
	// empty for old versions → inference fallback.
	Configs []ProtoFileEntry

	// FetchedAt is when this version was fetched from Git.
	FetchedAt time.Time

	// FetchedBy is the user Id who triggered the fetch (optional).
	FetchedBy *string
}

// ProtoVersionNew creates a new ProtoVersion with generated Id and timestamps.
func ProtoVersionNew(init ...func(*ProtoVersion)) *ProtoVersion {
	v := &ProtoVersion{
		BaseEntity: *New(),
	}

	if len(init) > 0 && init[0] != nil {
		init[0](v)
	}

	return v
}

// ProtoVersions is a slice of ProtoVersion pointers.
type ProtoVersions []*ProtoVersion

// ProtoFileEntry is a single .proto file's content.
type ProtoFileEntry struct {
	// Path is the repo-relative path (e.g., "proto/orders/v1/order.proto").
	Path string

	Content string

	// Size is the file size in bytes.
	Size int64
}

// ProtoVersionsList is the listing criteria for versions.
type ProtoVersionsList struct {
	ListBase
	SourceID string
}
