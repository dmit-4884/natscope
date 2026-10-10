// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

// Package registry provides a per-snapshot proto descriptor cache;
// decode/encode use the cached snapshot directly with no cross-source merging
// or per-message re-parse.
package registry

import (
	"time"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/pkg/protoutils"
)

// Snapshot is an immutable parsed view of one stored schema.
type Snapshot struct {
	SourceID    string
	Revision    string
	Descriptor  *entities.ProtoDescriptor
	Schema      *protoutils.Schema
	ParsedAt    time.Time
	Fingerprint string
}
