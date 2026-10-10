// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"context"
	"errors"
	"iter"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/altessa-s/go-atlas/core/retry"
)

// pullRetries is how many times a pull that found no responder is repeated.
const pullRetries = 5

// pullRetryDelay is the wait before the first repeated pull; each next one waits twice as long.
const pullRetryDelay = 25 * time.Millisecond

// pulledBatch is a fetched batch whose first message was already received.
type pulledBatch struct {
	first jetstream.Msg
	batch jetstream.MessageBatch
}

// All yields the batch's messages in order.
func (b pulledBatch) All() iter.Seq[jetstream.Msg] {
	return func(yield func(jetstream.Msg) bool) {
		if b.first == nil || !yield(b.first) {
			return
		}
		for msg := range b.batch.Messages() {
			if !yield(msg) {
				return
			}
		}
	}
}

// Error is the batch's error once All has yielded every message.
func (b pulledBatch) Error() error {
	return b.batch.Error()
}

// fetchPull is consumer.Fetch that pulls again while the pull request finds no responder.
func fetchPull(ctx context.Context, consumer jetstream.Consumer, n int, wait time.Duration) (pulledBatch, error) {
	var pulled pulledBatch
	err := retry.Do(ctx, func(context.Context) error {
		batch, err := consumer.Fetch(n, jetstream.FetchMaxWait(wait))
		if err != nil {
			return err
		}
		first, ok := <-batch.Messages()
		if !ok && errors.Is(batch.Error(), nats.ErrNoResponders) {
			return batch.Error()
		}
		pulled = pulledBatch{first: first, batch: batch}
		return nil
	},
		retry.WithMaxAttempts(pullRetries),
		retry.WithNextDelay(func(attempt int, _ error) time.Duration { return pullRetryDelay << attempt }),
		retry.WithShouldRetry(func(err error) bool { return errors.Is(err, nats.ErrNoResponders) }),
	)
	return pulled, noAnswer(err)
}
