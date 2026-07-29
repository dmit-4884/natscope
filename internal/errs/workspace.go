// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package errs

import "errors"

// Workspace export/import domain errors.
var (
	// ErrWorkspaceInvalidFile means the import payload is not a valid workspace
	// file (bad JSON or missing envelope).
	ErrWorkspaceInvalidFile = errors.New("workspace: invalid workspace file")

	// ErrWorkspaceSectionInvalid means a section payload could not be parsed into
	// the expected shape.
	ErrWorkspaceSectionInvalid = errors.New("workspace: invalid section payload")

	// ErrWorkspaceUnknownSection means export requested an unregistered section
	// key, surfaced instead of silently omitting it.
	ErrWorkspaceUnknownSection = errors.New("workspace: unknown section key")
)
