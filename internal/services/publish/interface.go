// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package publish

import (
	"context"

	"github.com/dmit-4884/natscope/internal/entities"
)

// Service coordinates the publish flow; see package doc for soft-failure
// semantics.
type Service interface {
	// Publish runs the pipeline and always returns a non-nil result; the error
	// return is reserved for unexpected internal failures (user-actionable ones go
	// through PublishResult.Error).
	Publish(ctx context.Context, in *entities.PublishRequest) (*entities.PublishResult, error)

	// Request sends a core NATS request and returns the first reply. Unlike
	// Publish, every failure is an error (errs.ErrNATSNoResponders,
	// errs.ErrNATSTimeout, *errs.ProtoEncodeError, ...) and nothing is recorded to
	// history.
	Request(ctx context.Context, in *entities.RequestMessage) (*entities.Reply, error)
}
