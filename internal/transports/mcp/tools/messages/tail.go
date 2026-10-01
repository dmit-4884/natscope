// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package messages

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/altessa-s/go-atlas/core/types/ptr"

	"github.com/dmit-4884/natscope/internal/entities"

	corectx "github.com/altessa-s/go-atlas/core/context"
	mcptransport "github.com/dmit-4884/natscope/internal/transports/mcp"
)

var errTailFull = errors.New("tail reached maxMessages")

type tailCollector struct {
	mu    sync.Mutex
	out   tailOutput
	max   int
	limit int
}

func (c *tailCollector) emit(ev *entities.LiveEvent) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	switch {
	case ev.Batch != nil:
		for _, m := range ev.Batch.Messages {
			c.out.Messages = append(c.out.Messages, newLiveMessageView(m, c.limit))
			if len(c.out.Messages) >= c.max {
				return errTailFull
			}
		}
	case ev.Stats != nil:
		c.out.Dropped = ev.Stats.MessagesDropped
	case ev.Error != nil:
		c.out.Errors = append(c.out.Errors, ev.Error.Message)
	}
	return nil
}

func (c *tailCollector) result() tailOutput {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.out.StoppedBy = "timeout"
	if len(c.out.Messages) >= c.max {
		c.out.StoppedBy = "maxMessages"
	}
	return c.out
}

func (t *Toolset) tailSubject(ctx context.Context, _ *mcp.CallToolRequest, in tailInput) (*mcp.CallToolResult, tailOutput, error) {
	connID, err := t.conns.Resolve(ctx, in.Connection)
	if err != nil {
		return nil, tailOutput{}, err
	}

	maxMessages := mcptransport.Limit(in.MaxMessages, defaultTailMessages, maxTailMessages)
	limit := payloadLimit(in.MaxPayloadBytes, maxMessages)
	target := &entities.LiveSubscriptionTarget{Subject: strings.TrimSpace(in.Subject)}
	if stream := strings.TrimSpace(in.Stream); stream != "" {
		target.StreamName = &stream
	}
	collector := &tailCollector{
		out:   tailOutput{Messages: []liveMessageView{}},
		max:   maxMessages,
		limit: limit,
	}

	tailCtx, cancel := corectx.ApplyTimeout(ctx, time.Duration(mcptransport.Limit(in.Seconds, defaultTailSeconds, maxTailSeconds))*time.Second)
	defer cancel()

	err = t.live.Subscribe(tailCtx, &entities.LiveSubscribeRequest{
		ConnectionId:    connID,
		Subscriptions:   []*entities.LiveSubscriptionTarget{target},
		MaxPayloadBytes: ptr.Wrap(int32(limit)),
	}, collector.emit)
	switch {
	case err != nil && !errors.Is(err, errTailFull):
		return nil, tailOutput{}, err
	case err == nil && ctx.Err() != nil:
		return nil, tailOutput{}, ctx.Err()
	}
	return nil, collector.result(), nil
}

func newLiveMessageView(m *entities.LiveMessage, limit int) liveMessageView {
	nm := m.NatsMessage
	v := liveMessageView{
		Subject:     nm.Subject,
		Stream:      nm.Stream,
		Sequence:    nm.Sequence,
		Headers:     nm.Header,
		Size:        max(m.OriginalSize, len(nm.Data)),
		DecodedType: ptr.Unwrap(m.DecodedType),
		DecodedAuto: m.DecodedAuto,
		DecodeError: ptr.Unwrap(m.DecodeError),
		Truncated:   m.Truncated,
	}
	if !nm.Timestamp.IsZero() {
		v.Timestamp = &nm.Timestamp
	}
	decoded := ptr.Unwrap(m.Decoded)
	switch {
	case decoded != "" && len(decoded) <= limit && json.Valid([]byte(decoded)):
		v.Decoded = json.RawMessage(decoded)
		v.Truncated = false
	case decoded != "":
		v.Body, v.Truncated = mcptransport.NewBody([]byte(decoded), limit, true)
	default:
		v.Body, v.Truncated = mcptransport.NewBody(nm.Data, limit, m.Truncated)
	}
	return v
}
