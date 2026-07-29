// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package workspace defines the section-registry contract for export/import;
// the Service iterates the registry and never imports domains directly.
package workspace

import (
	"context"
	"encoding/json"

	"github.com/dmit-4884/natscope/internal/entities"
)

// Section is one exportable/importable slice of a workspace; implementations
// exclude secrets from Export by design.
type Section interface {
	// Key is the stable unique identifier (e.g. "mappings").
	Key() string

	// Describe returns the title + current item count for the UI.
	Describe(ctx context.Context) (entities.WorkspaceSectionInfo, error)

	// Export serializes this section to a self-versioned, secret-free JSON
	// payload.
	Export(ctx context.Context) (json.RawMessage, error)

	// Validate dry-runs an import, returning counts without mutating anything.
	Validate(
		ctx context.Context,
		raw json.RawMessage,
		strategy entities.WorkspaceStrategy,
	) (entities.WorkspaceSectionReport, error)

	// Import applies raw under the strategy and returns what changed.
	Import(
		ctx context.Context,
		raw json.RawMessage,
		strategy entities.WorkspaceStrategy,
	) (entities.WorkspaceSectionResult, error)
}

// Service is the domain-agnostic coordinator over registered sections.
type Service interface {
	// ListSections returns every registered section's self-description.
	ListSections(ctx context.Context) ([]entities.WorkspaceSectionInfo, error)

	// Export builds a workspace file from the given keys (empty = all).
	Export(ctx context.Context, keys []string) ([]byte, error)

	// Validate dry-runs an import (empty keys = all sections in the file); unknown
	// sections are reported, never fatal.
	Validate(
		ctx context.Context,
		payload []byte,
		keys []string,
		strategy entities.WorkspaceStrategy,
	) ([]entities.WorkspaceSectionReport, error)

	// Import applies the given keys (empty = all sections in the file).
	Import(
		ctx context.Context,
		payload []byte,
		keys []string,
		strategy entities.WorkspaceStrategy,
	) ([]entities.WorkspaceSectionResult, error)
}
