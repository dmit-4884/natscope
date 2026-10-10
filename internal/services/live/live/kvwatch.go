// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package live

import (
	"context"
	"time"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
)

// kvWatchValueBytes caps the value a watch sends with a change; the key itself holds the rest.
const kvWatchValueBytes = 1024

// WatchKV sends the changes of a bucket in batches until ctx ends, emit fails or the connection drops the watch.
// The first batch is empty and confirms the watch is running.
func (s *Service) WatchKV(ctx context.Context, in *entities.KVWatchRequest, emit func([]entities.KVChange) error) error {
	changes, err := s.natsService.WatchKV(ctx, in.ConnectionID, in.Bucket, in.Filter)
	if err != nil {
		return err
	}
	if err := emit([]entities.KVChange{}); err != nil {
		return err
	}

	ticker := time.NewTicker(batchInterval)
	defer ticker.Stop()

	var batch []entities.KVChange
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		out := batch
		batch = nil
		return emit(out)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case change, ok := <-changes:
			if !ok {
				if ctx.Err() != nil {
					return nil
				}
				if err := flush(); err != nil {
					return err
				}
				return errs.ErrNATSConnectionClosed
			}
			if len(change.Value) > kvWatchValueBytes {
				change.Value = change.Value[:kvWatchValueBytes]
			}
			batch = append(batch, change)
			if len(batch) >= maxBatchSize {
				if err := flush(); err != nil {
					return err
				}
			}
		case <-ticker.C:
			if err := flush(); err != nil {
				return err
			}
		}
	}
}
