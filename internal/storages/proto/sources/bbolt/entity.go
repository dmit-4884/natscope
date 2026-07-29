// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package bbolt

import "github.com/dmit-4884/natscope/internal/pkg/bbstore"

// sourceDoc is the persistence model for a proto source. Token is transient: it
// is carried from the entity so secretsplit can lift it into the keychain vault,
// and json:"-" keeps it out of the persisted document (see mapper.go).
type sourceDoc struct {
	bbstore.Base

	Name            string            `json:"name"`
	SourceType      string            `json:"sourceType,omitempty"`
	Enabled         bool              `json:"enabled,omitempty"`
	Repository      string            `json:"repository,omitempty"`
	Token           *string           `json:"-" behavior:"input_only" secret:"git.token"`
	LocalPath       *string           `json:"localPath,omitempty"`
	WatcherEnabled  bool              `json:"watcherEnabled,omitempty"`
	Files           []string          `json:"files,omitempty"`
	IncludeDirs     []string          `json:"includeDirs,omitempty"`
	ImportRoots     []string          `json:"importRoots,omitempty"`
	ExcludePrefixes []string          `json:"excludePrefixes,omitempty"`
	LastCompile     *compileResultDoc `json:"lastCompile,omitempty"`
}

type compileResultDoc struct {
	At           int64                  `json:"at,omitempty"`
	Ok           bool                   `json:"ok,omitempty"`
	Error        *string                `json:"error,omitempty"`
	MessageCount int32                  `json:"messageCount,omitempty"`
	FileCount    int32                  `json:"fileCount,omitempty"`
	Diagnostics  []compileDiagnosticDoc `json:"diagnostics,omitempty"`
	Roots        []string               `json:"roots,omitempty"`
	RootsOrigin  string                 `json:"rootsOrigin,omitempty"`
}

type compileDiagnosticDoc struct {
	Severity      string `json:"severity,omitempty"`
	File          string `json:"file,omitempty"`
	Line          int    `json:"line,omitempty"`
	Column        int    `json:"column,omitempty"`
	Message       string `json:"message,omitempty"`
	MissingImport string `json:"missingImport,omitempty"`
	Hint          string `json:"hint,omitempty"`
}
