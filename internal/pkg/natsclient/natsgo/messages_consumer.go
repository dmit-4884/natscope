// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"

	coreerrs "github.com/altessa-s/go-atlas/core/errors"
)

// browseConsumerInactiveThreshold is the server-side safety net keeping the
// browse consumer alive if our defer DeleteConsumer gets killed mid-flight.
const browseConsumerInactiveThreshold = 10 * time.Second

// backwardFetchMultiplier widens the fetch when paginating backward, since
// consumers can't address by sequence (over-pull and trim client-side).
const backwardFetchMultiplier = 2

// getMessagesViaConsumer fetches via an ordered consumer (best for
// filtered/sparse streams); rejects WorkQueue since AckNone would drain it.
func (c *Client) getMessagesViaConsumer(
	ctx context.Context,
	streamName string,
	subjectFilter string,
	startSeq uint64,
	limit int,
	direction string,
) (*entities.MessagesResponse, error) {
	stream, err := c.jetStream.Stream(ctx, streamName)
	if err != nil {
		return nil, wrapErr(coreerrs.WrapOperation(err, "get stream"))
	}

	info := stream.CachedInfo()

	if info != nil && info.Config.Retention == jetstream.WorkQueuePolicy {
		return nil, errs.ErrWorkQueueConsumerNotAllowed
	}

	deliverPolicy := jetstream.DeliverByStartSequencePolicy
	var filterSubjects []string
	if subjectFilter != "" {
		filterSubjects = []string{subjectFilter}
	}

	var optStartSeq uint64

	if direction == "forward" {
		if startSeq > 0 {
			optStartSeq = startSeq
		} else {
			optStartSeq = info.State.FirstSeq
		}
	} else {
		endSeq := startSeq
		if endSeq == 0 {
			endSeq = info.State.LastSeq
		}

		fetchStart := uint64(1)
		if endSeq > uint64(limit)*backwardFetchMultiplier {
			fetchStart = endSeq - uint64(limit)*backwardFetchMultiplier
		}
		if fetchStart < info.State.FirstSeq {
			fetchStart = info.State.FirstSeq
		}

		optStartSeq = fetchStart
	}

	ephCfg := jetstream.ConsumerConfig{
		DeliverPolicy:     deliverPolicy,
		OptStartSeq:       optStartSeq,
		AckPolicy:         jetstream.AckNonePolicy,
		InactiveThreshold: browseConsumerInactiveThreshold,
		Name:              fmt.Sprintf("natscope-browse-%s", nats.NewInbox()[7:]),
	}

	if len(filterSubjects) == 1 {
		ephCfg.FilterSubject = filterSubjects[0]
	} else if len(filterSubjects) > 1 {
		ephCfg.FilterSubjects = filterSubjects
	}

	consumer, err := stream.CreateConsumer(ctx, ephCfg)
	if err != nil {
		return nil, wrapErr(err)
	}
	defer func() {
		_ = stream.DeleteConsumer(ctx, ephCfg.Name) //nolint:errcheck // best-effort cleanup
	}()

	fetchLimit := limit + 1 // +1 for hasMore detection
	if direction == DefaultDirection {
		// For backward, over-fetch to collect enough messages up to startSeq.
		fetchLimit = (limit + 1) * backwardFetchMultiplier
	}

	batch, err := consumer.FetchNoWait(fetchLimit)
	if err != nil {
		return nil, wrapErr(coreerrs.Wrap(err, "fetch messages"))
	}

	messages := make([]*entities.Message, 0, fetchLimit)
	for msg := range batch.Messages() {
		meta, metaErr := msg.Metadata()
		if metaErr != nil || meta == nil {
			continue
		}

		// For backward direction, skip messages after startSeq.
		if direction == DefaultDirection && startSeq > 0 && meta.Sequence.Stream > startSeq {
			continue
		}

		headers := make(map[string]string)
		hdrs := msg.Headers()
		for k := range hdrs {
			headers[k] = hdrs.Get(k)
		}

		messages = append(messages, &entities.Message{
			Sequence:    meta.Sequence.Stream,
			Subject:     msg.Subject(),
			Timestamp:   meta.Timestamp,
			DataBase64:  base64.StdEncoding.EncodeToString(msg.Data()),
			DataSize:    len(msg.Data()),
			ContentType: entities.DetectContentType(msg.Data()),
			Headers:     headers,
		})
	}

	// A plain fetch deadline just means the batch is exhausted; anything else
	// (including a permissions violation recovered by the watched client) is a
	// real error.
	if batchErr := batch.Error(); batchErr != nil && !errors.Is(batchErr, context.DeadlineExceeded) {
		return nil, batchErr
	}

	// Reverse for backward direction so newest messages come first.
	if direction == DefaultDirection {
		for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
			messages[i], messages[j] = messages[j], messages[i]
		}
	}

	// Trim to limit + detect hasMore.
	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[:limit]
	}

	var nextSeq uint64
	if hasMore && len(messages) > 0 {
		lastMsg := messages[len(messages)-1]
		if direction == DefaultDirection {
			if lastMsg.Sequence > info.State.FirstSeq {
				nextSeq = lastMsg.Sequence - 1
			}
		} else {
			if lastMsg.Sequence < info.State.LastSeq {
				nextSeq = lastMsg.Sequence + 1
			}
		}
	}

	return &entities.MessagesResponse{
		Messages: messages,
		HasMore:  hasMore,
		NextSeq:  nextSeq,
	}, nil
}
