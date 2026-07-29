// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package gitfetcher

import (
	"context"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/pkg/protoutils"
)

// FetchResult is everything extracted from a repository at a tag.
type FetchResult struct {
	// Files are the .proto files, slash-relative to the repo root.
	Files []entities.ProtoFileEntry
	// Configs are buf.yaml / buf.work.yaml / buf.lock files, same path rules.
	Configs []entities.ProtoFileEntry
	// Skipped reports oversize/unreadable entries (fetch does not abort).
	Skipped []protoutils.SkippedFile
}

// Service defines operations for fetching proto files from Git repositories.
type Service interface {
	// ListTags returns available tags from a Git repository.
	ListTags(ctx context.Context, repositoryURL string) ([]string, error)

	// FetchVersion fetches proto files and buf configs from a specific tag.
	FetchVersion(ctx context.Context, repositoryURL, tag string) (*FetchResult, error)

	// ValidateRepository checks if a repository URL is valid and accessible.
	ValidateRepository(ctx context.Context, repositoryURL string) error
}
