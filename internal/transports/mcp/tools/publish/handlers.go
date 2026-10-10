// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package publish

import (
	"context"
	"encoding/json"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"

	mcptransport "github.com/dmit-4884/natscope/internal/transports/mcp"
)

func (t *Toolset) publishMessage(ctx context.Context, _ *mcp.CallToolRequest, in publishInput) (*mcp.CallToolResult, publishOutput, error) {
	connID, err := t.conns.Resolve(ctx, in.Connection)
	if err != nil {
		return nil, publishOutput{}, err
	}
	req, out, err := t.request(ctx, in)
	if err != nil {
		return nil, publishOutput{}, err
	}
	req.ConnectionID = connID

	res, err := t.publish.Publish(ctx, req)
	if err != nil {
		return nil, publishOutput{}, err
	}
	if res.Error != nil {
		return nil, publishOutput{}, mcptransport.Errorf("%s", *res.Error)
	}
	out.Stream, out.Sequence, out.Duplicate = res.Stream, res.Sequence, res.Duplicate
	return nil, out, nil
}

func (t *Toolset) requestMessage(ctx context.Context, _ *mcp.CallToolRequest, in requestInput) (*mcp.CallToolResult, requestOutput, error) {
	if in.TimeoutMs < 0 || in.TimeoutMs > maxRequestTimeoutMs {
		return nil, requestOutput{}, mcptransport.Errorf("timeoutMs must be between 1 and %d", maxRequestTimeoutMs)
	}
	connID, err := t.conns.Resolve(ctx, in.Connection)
	if err != nil {
		return nil, requestOutput{}, err
	}

	req := &entities.PublishRequest{Subject: in.Subject, Headers: in.Headers}
	enc := publishOutput{Encoding: encodingRaw}
	if in.JSON != nil || in.Text != "" || in.Type != "" {
		req, enc, err = t.request(ctx, publishInput{
			Subject:  in.Subject,
			JSON:     in.JSON,
			Text:     in.Text,
			Type:     in.Type,
			SourceID: in.SourceID,
			Raw:      in.Raw,
			Headers:  in.Headers,
		})
		if err != nil {
			return nil, requestOutput{}, err
		}
	}
	req.ConnectionID = connID

	reply, err := t.publish.Request(ctx, &entities.RequestMessage{
		PublishRequest: *req,
		Timeout:        time.Duration(in.TimeoutMs) * time.Millisecond,
	})
	if err != nil {
		return nil, requestOutput{}, err
	}

	out := converter.Convert(reply, &requestOutput{})
	out.Size = len(reply.Data)
	out.DurationMs = float64(reply.Duration) / float64(time.Millisecond)
	out.Body, out.Truncated = mcptransport.NewBody(reply.Data, replyPayloadBudget, false)
	out.Encoding, out.MessageType = enc.Encoding, enc.MessageType
	return nil, *out, nil
}

func (t *Toolset) request(ctx context.Context, in publishInput) (*entities.PublishRequest, publishOutput, error) {
	subject := in.Subject
	req := &entities.PublishRequest{Subject: subject, Headers: in.Headers}
	out := publishOutput{Encoding: encodingRaw}

	switch {
	case in.Raw && in.Type != "":
		return nil, out, mcptransport.Errorf("raw and type contradict each other; drop one")
	case in.JSON != nil && in.Text != "":
		return nil, out, mcptransport.Errorf("pass either json or text, not both")
	case in.Text != "":
		if in.Type != "" {
			return nil, out, mcptransport.Errorf("type applies to a json payload; pass the message as json")
		}
		req.Data = in.Text
		return req, out, nil
	case in.JSON == nil:
		return nil, out, mcptransport.Errorf("pass the payload as json or text")
	}

	data, err := json.Marshal(in.JSON)
	if err != nil {
		return nil, out, mcptransport.Errorf("json payload does not serialize: %v", err)
	}
	req.Data = string(data)

	switch {
	case in.Type != "":
		info, lookupErr := mcptransport.LookupType(ctx, t.registry, in.Type, in.SourceID)
		if lookupErr != nil {
			return nil, out, lookupErr
		}
		req.MessageType, req.SourceID = &info.FullName, &info.SourceID
	case !in.Raw:
		if m := t.mappings.Resolver(ctx).Resolve(subject); m != nil {
			req.MessageType, req.SourceID, req.SchemaFingerprint = &m.MessageType, &m.SourceID, m.PinnedFingerprint
			req.Framing = m.Framing
		}
	}
	if req.MessageType != nil {
		out.Encoding, out.MessageType = encodingProtobuf, *req.MessageType
	}
	return req, out, nil
}
