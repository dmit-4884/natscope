// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

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

// pullRetryDelay is the wait before the first repeated pull.
const pullRetryDelay = 25 * time.Millisecond

// pullRetryFactor multiplies the wait before each next repeated pull.
const pullRetryFactor = 2

// pullRetry repeats a pull while JetStream has no responder for it yet.
var pullRetry = retry.NewPolicy(
	retry.WithMaxAttempts(pullRetries),
	retry.WithNextDelay(retry.Exponential(retry.ExponentialConfig{BaseDelay: pullRetryDelay, Factor: pullRetryFactor})),
	retry.WithShouldRetry(func(err error) bool { return errors.Is(err, nats.ErrNoResponders) }),
)

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
	err := pullRetry.Do(ctx, func(context.Context) error {
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
	})
	return pulled, noAnswer(err)
}
