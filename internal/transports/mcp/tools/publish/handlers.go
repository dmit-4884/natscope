// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package publish

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

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

func (t *Toolset) request(ctx context.Context, in publishInput) (*entities.PublishRequest, publishOutput, error) {
	subject := strings.TrimSpace(in.Subject)
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
			req.MessageType, req.SourceID, req.SourceTag = &m.MessageType, &m.SourceID, m.PinnedTag
		}
	}
	if req.MessageType != nil {
		out.Encoding, out.MessageType = encodingProtobuf, *req.MessageType
	}
	return req, out, nil
}
