// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package live

import (
	"context"

	"github.com/dmit-4884/natscope/internal/entities"
)

// Service runs live NATS subscription sessions and broadcasts global
// proto-reload signals to all in-flight sessions.
type Service interface {
	// Subscribe runs a session, calling emit per event; aborts on emit error, ctx
	// done, or when the initial subscription set could not be established.
	Subscribe(ctx context.Context, in *entities.LiveSubscribeRequest, emit func(*entities.LiveEvent) error) error

	// BroadcastProtoReload makes every in-flight session reflect reloaded proto
	// descriptors.
	BroadcastProtoReload()
}
