// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package messages

import (
	"bytes"
	"context"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/altessa-s/go-atlas/core/collections/slices"
	"github.com/altessa-s/go-atlas/core/types/ptr"
	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"

	mcptransport "github.com/dmit-4884/natscope/internal/transports/mcp"
)

func newMessageView(m *entities.Message, limit int) messageView {
	v := converter.Convert(m, &messageView{})
	switch {
	case len(v.Decoded) == 0:
		v.Body, v.Truncated = mcptransport.NewBodyFromBase64(m.DataBase64, limit, m.Truncated)
	case len(v.Decoded) > limit:
		v.Body, v.Truncated = mcptransport.NewBody(v.Decoded, limit, true)
		v.Decoded = nil
	default:
		v.Truncated = !bytes.HasPrefix(bytes.TrimSpace(v.Decoded), []byte("{"))
	}
	return *v
}

func (t *Toolset) findMessages(ctx context.Context, _ *mcp.CallToolRequest, in findMessagesInput) (*mcp.CallToolResult, findMessagesOutput, error) {
	req, err := listRequest(in, time.Now())
	if err != nil {
		return nil, findMessagesOutput{}, err
	}
	if req.ConnectionID, err = t.conns.Resolve(ctx, in.Connection); err != nil {
		return nil, findMessagesOutput{}, err
	}

	resp, err := t.messages.List(ctx, req)
	if err != nil {
		return nil, findMessagesOutput{}, err
	}
	limit := int(*req.MaxPayloadBytes)
	return nil, findMessagesOutput{
		Messages: mcptransport.Items(slices.To(resp.Messages, func(m *entities.Message) messageView { return newMessageView(m, limit) })),
		HasMore:  resp.HasMore,
		NextSeq:  resp.NextSeq,
	}, nil
}

func listRequest(in findMessagesInput, now time.Time) (*entities.MessageListRequest, error) {
	since, err := mcptransport.Since(in.Since, now)
	if err != nil {
		return nil, err
	}
	if since != nil && in.StartSeq > 0 {
		return nil, mcptransport.Errorf("pass either startSeq or since, not both")
	}

	direction := strings.ToLower(strings.TrimSpace(in.Direction))
	switch direction {
	case "":
		direction = directionBackward
		if since != nil || in.StartSeq > 0 {
			direction = directionForward
		}
	case directionForward, directionBackward:
	default:
		return nil, mcptransport.Errorf("direction must be %q or %q", directionForward, directionBackward)
	}

	pageSize := mcptransport.Limit(in.Limit, defaultPageSize, maxPageSize)
	req := &entities.MessageListRequest{
		StreamName:      in.Stream,
		StartTime:       since,
		Direction:       direction,
		Limit:           ptr.Wrap(int64(pageSize)),
		MaxPayloadBytes: ptr.Wrap(int32(payloadLimit(in.MaxPayloadBytes, pageSize))),
	}
	if in.StartSeq > 0 {
		req.StartSeq = ptr.Wrap(in.StartSeq)
	}
	if subject := strings.TrimSpace(in.Subject); subject != "" {
		req.SubjectFilter = &subject
	}
	if contains := strings.TrimSpace(in.Contains); contains != "" {
		req.ContentFilter = &contains
	}
	return req, nil
}

func (t *Toolset) getMessage(ctx context.Context, _ *mcp.CallToolRequest, in getMessageInput) (*mcp.CallToolResult, messageView, error) {
	if in.Seq == 0 {
		return nil, messageView{}, mcptransport.Errorf("seq must be a positive stream sequence")
	}
	connID, err := t.conns.Resolve(ctx, in.Connection)
	if err != nil {
		return nil, messageView{}, err
	}
	msg, err := t.messages.Get(ctx, &entities.MessageGetRequest{ConnectionID: connID, StreamName: in.Stream, Sequence: in.Seq})
	if err != nil {
		return nil, messageView{}, err
	}
	return nil, newMessageView(msg, mcptransport.Limit(in.MaxPayloadBytes, defaultMessagePayload, maxPayload)), nil
}

func payloadLimit(requested, count int) int {
	return min(mcptransport.Limit(requested, defaultPagePayload, maxPayload), max(responsePayloadBudget/count, minMessagePayload))
}
