// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package connections

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/altessa-s/go-atlas/core/collections/slices"
	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/pkg/natsutil"

	mcptransport "github.com/dmit-4884/natscope/internal/transports/mcp"
)

func (t *Toolset) listConnections(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, listConnectionsOutput, error) {
	all, err := t.conns.List(ctx)
	if err != nil {
		return nil, listConnectionsOutput{}, err
	}
	return nil, listConnectionsOutput{Connections: mcptransport.Items(slices.To(all, redact))}, nil
}

func redact(c *entities.SavedConnection) connectionView {
	v := converter.Convert(c, &connectionView{}, mcptransport.ViewCodecs, converter.WithHandleEmbeddedStructs(true))
	v.URLs = slices.To(natsutil.StripCredentials(c.URLs), natsutil.MaskURL)
	if v.Meta != nil && v.Meta.LastError != nil {
		v.Meta.LastError = new(natsutil.MaskURL(*v.Meta.LastError))
	}
	return *v
}

func (t *Toolset) serverInfo(ctx context.Context, _ *mcp.CallToolRequest, in mcptransport.ConnectionArg) (*mcp.CallToolResult, serverView, error) {
	connID, err := t.conns.Resolve(ctx, in.Connection)
	if err != nil {
		return nil, serverView{}, err
	}
	info, err := t.stats.GetServerInfo(ctx, connID)
	if err != nil {
		return nil, serverView{}, err
	}
	return nil, *converter.Convert(info, &serverView{}), nil
}
