// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

import "encoding/json"

// WorkspaceFileVersion is the top-level version of the workspace export format.
const WorkspaceFileVersion = 1

// WorkspaceStrategy selects how an import resolves collisions.
type WorkspaceStrategy string

const (
	// WorkspaceStrategyMerge upserts items by name (update existing, create new).
	WorkspaceStrategyMerge WorkspaceStrategy = "merge"
	// WorkspaceStrategyReplace clears the section, then creates from the file.
	WorkspaceStrategyReplace WorkspaceStrategy = "replace"
)

// WorkspaceFile is the on-disk shape of an exported workspace; each section
// carries its own sub-version so unknown sections skip with a warning.
type WorkspaceFile struct {
	Version    int                        `json:"version"`
	ExportedAt int64                      `json:"exported_at,omitempty"`
	Sections   map[string]json.RawMessage `json:"sections"`
}

// WorkspaceSectionInfo is a section's self-description for the UI.
type WorkspaceSectionInfo struct {
	Key   string
	Title string
	Count int32
}

// WorkspaceSectionReport is the dry-run outcome for one section.
type WorkspaceSectionReport struct {
	Key     string
	Created int32
	Updated int32
	// Deleted counts existing items a REPLACE import would remove before
	// recreating (0 under MERGE); surfaced for a pre-import destructive warning.
	Deleted   int32
	Conflicts []string
	Warnings  []string
	// Unknown is set when the file has this section but the server has no
	// matching registered Section (forward-compatibility).
	Unknown bool
}

// WorkspaceSectionResult is the applied outcome for one section.
type WorkspaceSectionResult struct {
	Key      string
	Created  int32
	Updated  int32
	Deleted  int32
	Warnings []string
}
