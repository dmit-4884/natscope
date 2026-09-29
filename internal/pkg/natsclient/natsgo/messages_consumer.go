// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
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

// backwardFetchMultiplier sizes the initial backward window as limit*backwardFetchMultiplier sequences.
const backwardFetchMultiplier = 2

// backwardWidenFactor multiplies the backward window on each retry.
const backwardWidenFactor = 4

// backwardWidenAttempts bounds how many times the backward window widens.
const backwardWidenAttempts = 6

// getMessagesViaConsumer fetches via an ephemeral consumer (best for
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
	// An empty stream has no valid start sequence, so return an empty page.
	if info != nil && info.State.Msgs == 0 {
		return &entities.MessagesResponse{Messages: []*entities.Message{}, HasMore: false}, nil
	}

	var filterSubjects []string
	if subjectFilter != "" {
		filterSubjects = []string{subjectFilter}
	}

	if direction == "forward" {
		optStartSeq := startSeq
		if optStartSeq == 0 {
			optStartSeq = info.State.FirstSeq
		}
		return c.consumeBrowseBatch(ctx, stream, info, optStartSeq, startSeq, filterSubjects, limit, direction)
	}

	endSeq := startSeq
	if endSeq == 0 {
		endSeq = info.State.LastSeq
	}

	maxWindow := uint64(DefaultSearchRange - 1)
	window := min(uint64(limit+1)*backwardFetchMultiplier, maxWindow)

	var resp *entities.MessagesResponse
	var fetchStart uint64
	for attempt := 0; ; attempt++ {
		fetchStart = max(info.State.FirstSeq, 1)
		if endSeq > window && endSeq-window > fetchStart {
			fetchStart = endSeq - window
		}

		resp, err = c.consumeBrowseBatch(ctx, stream, info, fetchStart, startSeq, filterSubjects, limit, direction)
		if err != nil {
			return nil, err
		}

		if resp.HasMore || fetchStart <= info.State.FirstSeq {
			return resp, nil
		}
		if attempt+1 >= backwardWidenAttempts || window >= maxWindow {
			break
		}
		window = min(window*backwardWidenFactor, maxWindow)
	}

	resp.HasMore = true
	resp.NextSeq = fetchStart - 1
	return resp, nil
}

// consumeBrowseBatch pulls one window from an ephemeral consumer at optStartSeq and shapes it for direction.
// startSeq is the request's upper bound (0 = LastSeq); backward results past it are dropped.
func (c *Client) consumeBrowseBatch(
	ctx context.Context,
	stream jetstream.Stream,
	info *jetstream.StreamInfo,
	optStartSeq uint64,
	startSeq uint64,
	filterSubjects []string,
	limit int,
	direction string,
) (*entities.MessagesResponse, error) {
	ephCfg := jetstream.ConsumerConfig{
		DeliverPolicy:     jetstream.DeliverByStartSequencePolicy,
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

	endSeq := startSeq
	fetchLimit := limit + 1 // +1 for hasMore detection
	if direction == DefaultDirection {
		if endSeq == 0 {
			endSeq = info.State.LastSeq
		}
		// Size the fetch cap to the whole window so a widened window reaches endSeq.
		if endSeq >= optStartSeq {
			span := endSeq - optStartSeq + 1
			fetchLimit = int(min(span, uint64(DefaultSearchRange)))
		}
		fetchLimit = max(fetchLimit, limit+1)
	}

	batch, err := consumer.FetchNoWait(fetchLimit)
	if err != nil {
		return nil, wrapErr(coreerrs.Wrap(err, "fetch messages"))
	}

	messages := make([]*entities.Message, 0, min(fetchLimit, limit+1))
	for msg := range batch.Messages() {
		meta, metaErr := msg.Metadata()
		if metaErr != nil || meta == nil {
			continue
		}

		// For backward direction, skip messages after the requested upper bound.
		if direction == DefaultDirection && endSeq > 0 && meta.Sequence.Stream > endSeq {
			continue
		}

		headers := make(map[string]string)
		hdrs := msg.Headers()
		for k, v := range hdrs {
			headers[k] = strings.Join(v, ", ")
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
