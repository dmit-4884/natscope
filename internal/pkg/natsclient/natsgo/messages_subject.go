// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package natsgo

import (
	"context"
	"errors"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/pkg/natsutil"
)

// exactSubjectScanRange bounds the backward probe for exact-subject matches on
// huge streams.
const exactSubjectScanRange = 1000

// wildcardFetchBatch is the sequence-window size per wildcard-filter fetch pass.
const wildcardFetchBatch = 200

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
	// An empty stream has no sequence range to search.
	if info.State.Msgs == 0 {
		return &entities.MessagesResponse{Messages: []*entities.Message{}, HasMore: false}, nil
	}

	if !containsWildcard(subjectFilter) {
		if direction == "forward" {
			return c.getMessagesForExactSubjectForward(ctx, stream, info, subjectFilter, startSeq, limit)
		}
		return c.getMessagesForExactSubjectBackward(ctx, stream, info, subjectFilter, startSeq, limit)
	}
	return c.getMessagesForWildcardSubject(ctx, stream, info, subjectFilter, startSeq, limit, direction)
}

// getMessagesForExactSubjectForward walks forward from the oldest match using
// JetStream's server-side subject index (next_by_subj).
func (c *Client) getMessagesForExactSubjectForward(
	ctx context.Context,
	stream jetstream.Stream,
	info *jetstream.StreamInfo,
	subjectFilter string,
	startSeq uint64,
	limit int,
) (*entities.MessagesResponse, error) {
	cursor := startSeq
	if cursor == 0 {
		cursor = info.State.FirstSeq
	}

	messages := make([]*entities.Message, 0, limit)
	lastSeq := cursor
	for len(messages) < limit+1 {
		msg, err := stream.GetMsg(ctx, cursor, jetstream.WithGetMsgSubject(subjectFilter))
		if err != nil {
			if errors.Is(err, jetstream.ErrMsgNotFound) {
				break
			}
			return nil, wrapErr(err)
		}
		messages = append(messages, toMessage(msg))
		lastSeq = msg.Sequence
		cursor = msg.Sequence + 1
	}

	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[:limit]
		lastSeq = messages[limit-1].Sequence
	}

	nextSeq := calculateNextSeq(hasMore, "forward", lastSeq, info)

	return &entities.MessagesResponse{
		Messages: messages,
		HasMore:  hasMore,
		NextSeq:  nextSeq,
	}, nil
}

// getMessagesForExactSubjectBackward walks backward from the latest match, capped at exactSubjectScanRange.
func (c *Client) getMessagesForExactSubjectBackward(
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
			if errors.Is(err, jetstream.ErrMsgNotFound) {
				return &entities.MessagesResponse{
					Messages: []*entities.Message{},
					HasMore:  false,
				}, nil
			}
			return nil, wrapErr(err)
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
	switch {
	case len(messages) > 0:
		nextSeq = messages[len(messages)-1].Sequence - 1
	case hasMore:
		// The scan cap stopped short of FirstSeq, so resume past this window instead of leaving nextSeq at 0.
		nextSeq = startSeq + 1
	}

	return &entities.MessagesResponse{
		Messages: messages,
		HasMore:  hasMore,
		NextSeq:  nextSeq,
	}, nil
}

// getMessagesForWildcardSubject scans bounded batches from the direction's boundary until enough messages match.
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
		if direction == DefaultDirection {
			startSeq = info.State.LastSeq
		} else {
			startSeq = info.State.FirstSeq
		}
	}

	var messages []*entities.Message //nolint:prealloc
	var lastProcessedSeq uint64
	hasMore := false
	cursor := startSeq

	for scanned := 0; scanned < DefaultSearchRange; {
		batchSize := min(wildcardFetchBatch, DefaultSearchRange-scanned)
		seqsToFetch := buildSequenceList(cursor, info, direction, batchSize)
		if len(seqsToFetch) == 0 {
			break
		}
		scanned += len(seqsToFetch)

		msgMap, err := c.fetchMessagesParallel(ctx, stream, seqsToFetch)
		if err != nil {
			return nil, wrapErr(err)
		}

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
		if hasMore {
			break
		}

		last := seqsToFetch[len(seqsToFetch)-1]
		lastProcessedSeq = last
		if len(seqsToFetch) < batchSize {
			// A short batch means buildSequenceList hit the stream boundary.
			break
		}
		if direction == DefaultDirection {
			if last == 0 {
				break
			}
			cursor = last - 1
		} else {
			cursor = last + 1
		}
	}

	cursorSeq := lastProcessedSeq
	if len(messages) > 0 {
		cursorSeq = messages[len(messages)-1].Sequence
	}
	if !hasMore {
		hasMore = hasMoreMessages(direction, cursorSeq, info)
	}
	nextSeq := calculateNextSeq(hasMore, direction, cursorSeq, info)

	return &entities.MessagesResponse{
		Messages: messages,
		HasMore:  hasMore,
		NextSeq:  nextSeq,
	}, nil
}

// GetNextMessage asks the server for the next message on each subject (next_by_subj) and keeps the earliest.
func (c *Client) GetNextMessage(ctx context.Context, streamName string, startSeq uint64, subjects []string) (*entities.Message, error) {
	if err := validateNATSNameLength("stream name", streamName); err != nil {
		return nil, wrapErr(err)
	}
	for _, subject := range subjects {
		if err := validateNATSSubjectLength("subject", subject); err != nil {
			return nil, wrapErr(err)
		}
		if err := natsutil.ValidateSubjectPattern(subject); err != nil {
			return nil, wrapErr(err)
		}
	}
	if len(subjects) == 0 {
		subjects = []string{">"}
	}

	stream, err := c.jetStream.Stream(ctx, streamName)
	if err != nil {
		return nil, wrapErr(err)
	}

	var next *jetstream.RawStreamMsg
	for _, subject := range subjects {
		msg, err := stream.GetMsg(ctx, startSeq, jetstream.WithGetMsgSubject(subject))
		if errors.Is(err, jetstream.ErrMsgNotFound) {
			continue
		}
		if err != nil {
			return nil, wrapErr(err)
		}
		if next == nil || msg.Sequence < next.Sequence {
			next = msg
		}
	}
	if next == nil {
		return nil, nil //nolint:nilnil // nil, nil means no message matches
	}
	return toMessage(next), nil
}
