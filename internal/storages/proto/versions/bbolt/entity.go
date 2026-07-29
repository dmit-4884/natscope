// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package bbolt

import "github.com/dmit-4884/natscope/internal/pkg/bbstore"

// versionDoc is the persistence model for a fetched proto version. The raw files
// and captured configs are nested JSON instead of child tables.
type versionDoc struct {
	bbstore.Base

	SourceID  string              `json:"sourceId"`
	Tag       string              `json:"tag"`
	Files     []protoFileEntryDoc `json:"files,omitempty"`
	Configs   []protoFileEntryDoc `json:"configs,omitempty"`
	FetchedAt int64               `json:"fetchedAt,omitempty"`
	FetchedBy *string             `json:"fetchedBy,omitempty"`
}

type protoFileEntryDoc struct {
	Path    string `json:"path"`
	Content string `json:"content,omitempty"`
	Size    int64  `json:"size,omitempty"`
}
