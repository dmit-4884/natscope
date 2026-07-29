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
	// List fetches a page, applies settings defaults, decodes, and applies the
	// optional case-insensitive substring content filter.
	List(ctx context.Context, in *entities.MessageListRequest) (*entities.MessagesResponse, error)

	// Get fetches and decodes a single message by sequence number.
	Get(ctx context.Context, in *entities.MessageGetRequest) (*entities.Message, error)
}
