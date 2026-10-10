// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package gitfetcher

import (
	"context"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/pkg/protoutils"
)

// FetchResult is the .proto tree of a repository at one commit.
type FetchResult struct {
	// Revision is the commit SHA that was checked out.
	Revision string
	// Files are the .proto files, slash-relative to the repo root.
	Files []entities.ProtoFileEntry
	// Configs are buf.yaml / buf.work.yaml / buf.lock files, same path rules.
	Configs []entities.ProtoFileEntry
	// Skipped reports oversize/unreadable entries (fetch does not abort).
	Skipped []protoutils.SkippedFile
}

// Service defines operations for fetching proto files from Git repositories.
type Service interface {
	// ListRefs returns tags newest first, then branches, with their commits.
	ListRefs(ctx context.Context, repositoryURL string) ([]entities.ProtoRef, error)

	// ResolveRef finds a tag, branch or full commit SHA; errs.ErrProtoRefNotFound when none matches.
	ResolveRef(ctx context.Context, repositoryURL, ref string) (entities.ProtoRef, error)

	// Fetch checks out ref and collects its .proto files and buf configs.
	Fetch(ctx context.Context, repositoryURL string, ref entities.ProtoRef) (*FetchResult, error)

	// ValidateRepository checks if a repository URL is valid and accessible.
	ValidateRepository(ctx context.Context, repositoryURL string) error
}
