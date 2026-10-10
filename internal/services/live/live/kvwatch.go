// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package live

import (
	"context"
	"time"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"

	corecontext "github.com/altessa-s/go-atlas/core/context"
)

const (
	// kvWatchValueBytes caps the value a watch sends with a change; the key itself holds the rest.
	kvWatchValueBytes = 1024

	// kvHealthInterval is how often a watch checks that its bucket's stream is reachable: a watch over a stream whose
	// server is down just waits, so without the check it would look live while nothing can arrive.
	kvHealthInterval = 5 * time.Second

	// kvHealthTimeout bounds one check.
	kvHealthTimeout = 3 * time.Second
)

// WatchKV sends the changes of a bucket in batches until ctx ends, emit fails or the connection drops the watch.
// The first frame is empty and confirms the watch is running; a frame without changes also reports the bucket's
// stream going offline and coming back.
func (s *Service) WatchKV(ctx context.Context, in *entities.KVWatchRequest, emit func(entities.KVWatchEvent) error) error {
	changes, err := s.natsService.WatchKV(ctx, in.ConnectionID, in.Bucket, in.Filter)
	if err != nil {
		return err
	}
	if err := emit(entities.KVWatchEvent{Changes: []entities.KVChange{}}); err != nil {
		return err
	}

	ticker := time.NewTicker(batchInterval)
	defer ticker.Stop()
	health := time.NewTicker(kvHealthInterval)
	defer health.Stop()

	var batch []entities.KVChange
	offline := false
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		out := batch
		batch = nil
		return emit(entities.KVWatchEvent{Changes: out})
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
			offline = false
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
		case <-health.C:
			now, known := s.bucketOffline(ctx, in)
			if !known || now == offline {
				continue
			}
			offline = now
			if err := flush(); err != nil {
				return err
			}
			if err := emit(entities.KVWatchEvent{Changes: []entities.KVChange{}, Offline: offline}); err != nil {
				return err
			}
		}
	}
}

// bucketOffline reports whether the server says the bucket's stream is offline; known is false when the check
// failed for another reason, which leaves the watch's state as it is.
func (s *Service) bucketOffline(ctx context.Context, in *entities.KVWatchRequest) (offline, known bool) {
	ctx, cancel := corecontext.WithMaxTimeout(ctx, kvHealthTimeout)
	defer cancel()
	_, err := s.natsService.GetKVBucket(ctx, in.ConnectionID, in.Bucket)
	switch {
	case err == nil:
		return false, true
	case errs.IsOffline(err):
		return true, true
	default:
		return false, false
	}
}
