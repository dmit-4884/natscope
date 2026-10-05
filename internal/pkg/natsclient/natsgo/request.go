// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/convcodecs"
)

// replyOpts maps a nats.Msg reply → entities.Reply. IgnoreZeroValues keeps a
// header-less reply's Headers nil; multi-value headers are joined.
var replyOpts = []converter.Option{
	converter.WithCodecs(convcodecs.StringSliceJoin),
	converter.WithFieldMappings(map[string]string{"Header": "Headers"}),
	converter.WithIgnoreZeroValues(),
}

// Request sends a core NATS request and returns the first reply. A permissions
// violation on the subject or on the reply inbox fails it at once instead of
// letting it run into ctx's deadline.
func (c *Client) Request(
	ctx context.Context,
	subject string,
	data []byte,
	headers map[string]string,
) (*entities.Reply, error) {
	if err := validateNATSSubjectLength("subject", subject); err != nil {
		return nil, wrapErr(err)
	}

	msg := &nats.Msg{Subject: subject, Data: data}
	if len(headers) > 0 {
		msg.Header = make(nats.Header, len(headers))
		for k, v := range headers {
			msg.Header.Set(k, v)
		}
	}

	var (
		reply *nats.Msg
		took  time.Duration
	)
	err := c.permWatch.WatchRequest(ctx, []string{subject}, func(ctx context.Context) error {
		start := time.Now()
		var reqErr error
		reply, reqErr = c.conn.RequestMsgWithContext(ctx, msg)
		took = time.Since(start)
		return reqErr
	})
	switch {
	case errors.Is(err, nats.ErrNoResponders):
		return nil, errors.Join(errs.ErrNATSNoResponders, err)
	case err != nil:
		return nil, widenInboxDenial(wrapErr(err), c.inboxPrefix())
	}

	result := converter.Convert(reply, &entities.Reply{}, replyOpts...)
	result.Duration = took
	return result, nil
}

// replySubscription is the subject of the reply subscription nats.go shares between a connection's requests.
func replySubscription(conn *nats.Conn) string {
	inbox := conn.NewRespInbox()
	return inbox[:strings.LastIndexByte(inbox, '.')+1] + "*"
}

// inboxPrefix returns the subject prefix of this connection's reply inboxes,
// ending in "." so the permission watcher matches it as a prefix.
func (c *Client) inboxPrefix() string {
	if p := c.conn.Opts.InboxPrefix; p != "" {
		return p + "."
	}
	return nats.InboxPrefix
}
