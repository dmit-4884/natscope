// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package messages

import (
	"context"

	"github.com/dmit-4884/natscope/internal/entities"
)

// Service coordinates message listing and single-message fetch across
// NATS, proto and settings. See package doc for fall-through semantics.
type Service interface {
	// List fetches a page, applies settings defaults and decodes it; content search is Search.
	List(ctx context.Context, in *entities.MessageListRequest) (*entities.MessagesResponse, error)

	// Get fetches and decodes a single message by sequence number.
	Get(ctx context.Context, in *entities.MessageGetRequest) (*entities.Message, error)

	// Next fetches and decodes the first message at or after a sequence that matches the filters; nil when none does.
	Next(ctx context.Context, in *entities.MessageNextRequest) (*entities.Message, error)

	// Search reads the stream in budgeted runs and streams progress, batches of matches and a summary through emit.
	Search(ctx context.Context, in *entities.MessageSearchRequest, emit func(*entities.MessageSearchEvent) error) error
}
