// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package bbolt

import "github.com/dmit-4884/natscope/internal/pkg/bbstore"

type fileSetDoc struct {
	bbstore.Base

	SourceID  string         `json:"sourceId"`
	Revision  string         `json:"revision"`
	Files     []fileEntryDoc `json:"files,omitempty"`
	Configs   []fileEntryDoc `json:"configs,omitempty"`
	FetchedAt int64          `json:"fetchedAt,omitempty"`
}

type fileEntryDoc struct {
	Path    string `json:"path"`
	Content string `json:"content,omitempty"`
	Size    int64  `json:"size,omitempty"`
}
