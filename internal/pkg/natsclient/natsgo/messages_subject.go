// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"context"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/pkg/natsutil"
)

// exactSubjectScanRange bounds the backward probe for exact-subject matches on
// huge streams.
const exactSubjectScanRange = 1000

// getMessagesWithSubjectFilter dispatches exact subjects to the per-subject
// helper, wildcards to a range walk with client-side filter.
func (c *Client) getMessagesWithSubjectFilter(
	ctx context.Context,
	stream jetstream.Stream,
	info *jetstream.StreamInfo,
	subjectFilter string,
	startSeq uint64,
	limit int,
	direction string,
) (*entities.MessagesResponse, error) {
	if !containsWildcard(subjectFilter) {
		return c.getMessagesForExactSubject(ctx, stream, info, subjectFilter, startSeq, limit)
	}
	return c.getMessagesForWildcardSubject(ctx, stream, info, subjectFilter, startSeq, limit, direction)
}

// getMessagesForExactSubject walks backward from the latest match, fetching by
// sequence and rejecting mismatches, capped at exactSubjectScanRange.
func (c *Client) getMessagesForExactSubject(
	ctx context.Context,
	stream jetstream.Stream,
	info *jetstream.StreamInfo,
	subjectFilter string,
	startSeq uint64,
	limit int,
) (*entities.MessagesResponse, error) {
	var messages []*entities.Message //nolint:prealloc

	if startSeq == 0 {
		lastMsg, err := stream.GetLastMsgForSubject(ctx, subjectFilter)
		if err != nil {
			return &entities.MessagesResponse{
				Messages: []*entities.Message{},
				HasMore:  false,
			}, nil
		}
		startSeq = lastMsg.Sequence
		messages = append(messages, toMessage(lastMsg))

		if len(messages) >= limit {
			return &entities.MessagesResponse{
				Messages: messages,
				HasMore:  true,
				NextSeq:  startSeq - 1,
			}, nil
		}
		startSeq--
	}

	scanned := 0
	for startSeq >= info.State.FirstSeq && len(messages) < limit && scanned < exactSubjectScanRange {
		msg, err := stream.GetMsg(ctx, startSeq)
		if err == nil && msg.Subject == subjectFilter {
			messages = append(messages, toMessage(msg))
		}
		startSeq--
		scanned++
	}

	hasMore := startSeq >= info.State.FirstSeq && (scanned >= exactSubjectScanRange || len(messages) >= limit)
	var nextSeq uint64
	if len(messages) > 0 {
		nextSeq = messages[len(messages)-1].Sequence - 1
	}

	return &entities.MessagesResponse{
		Messages: messages,
		HasMore:  hasMore,
		NextSeq:  nextSeq,
	}, nil
}

// getMessagesForWildcardSubject parallel-fetches a sequence list, then filters
// per-message by NATS wildcard match.
func (c *Client) getMessagesForWildcardSubject(
	ctx context.Context,
	stream jetstream.Stream,
	info *jetstream.StreamInfo,
	subjectFilter string,
	startSeq uint64,
	limit int,
	direction string,
) (*entities.MessagesResponse, error) {
	if startSeq == 0 {
		startSeq = info.State.LastSeq
	}

	seqsToFetch := buildSequenceList(startSeq, info, direction, DefaultSearchRange)
	if len(seqsToFetch) == 0 {
		return &entities.MessagesResponse{
			Messages: []*entities.Message{},
			HasMore:  false,
		}, nil
	}

	msgMap, err := c.fetchMessagesParallel(ctx, stream, seqsToFetch)
	if err != nil {
		return nil, wrapErr(err)
	}

	var messages []*entities.Message //nolint:prealloc
	var lastProcessedSeq uint64
	hasMore := false

	for _, seq := range seqsToFetch {
		if len(messages) >= limit {
			hasMore = true
			break
		}

		msg, ok := msgMap[seq]
		if !ok {
			continue
		}

		if !natsutil.MatchSubject(subjectFilter, msg.Subject) {
			continue
		}

		messages = append(messages, toMessage(msg))
		lastProcessedSeq = seq
	}

	nextSeq := calculateNextSeq(hasMore, direction, lastProcessedSeq, info)

	return &entities.MessagesResponse{
		Messages: messages,
		HasMore:  hasMore,
		NextSeq:  nextSeq,
	}, nil
}
