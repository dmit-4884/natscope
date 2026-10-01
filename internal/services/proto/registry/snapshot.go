// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package registry provides a per-snapshot proto descriptor cache;
// decode/encode use the cached snapshot directly with no cross-source merging
// or per-message re-parse.
package registry

import (
	"time"

	"github.com/altessa-s/go-atlas/core/encoding/hash"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/pkg/protoutils"
)

// Snapshot is an immutable parsed view of a single ProtoDescriptor. Equal
// Fingerprint means interchangeable schemas, surviving no-op recompiles.
type Snapshot struct {
	SourceID    string
	Tag         string
	Descriptor  *entities.ProtoDescriptor
	Schema      *protoutils.Schema
	ParsedAt    time.Time
	Fingerprint string
}

// fingerprint computes a content hash for a serialized FileDescriptorSet; empty
// input yields "".
func fingerprint(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	return hash.SHA256HexBytes(data)
}
