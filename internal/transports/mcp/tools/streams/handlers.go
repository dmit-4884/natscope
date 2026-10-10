// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package streams

import (
	"context"
	"sort"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/altessa-s/go-atlas/core/collections/slices"
	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/transports/grpc/helpers"

	mcptransport "github.com/dmit-4884/natscope/internal/transports/mcp"
)

func (t *Toolset) listStreams(ctx context.Context, _ *mcp.CallToolRequest, in mcptransport.ConnectionArg) (*mcp.CallToolResult, listStreamsOutput, error) {
	connID, err := t.conns.Resolve(ctx, in.Connection)
	if err != nil {
		return nil, listStreamsOutput{}, err
	}
	list, err := t.streams.ListStreams(ctx, connID)
	if err != nil {
		return nil, listStreamsOutput{}, err
	}
	return nil, listStreamsOutput{Streams: mcptransport.Items(slices.To(list, func(s entities.StreamInfo) streamSummary {
		return *converter.Convert(&s, &streamSummary{}, mcptransport.ViewCodecs)
	}))}, nil
}

func (t *Toolset) getStream(ctx context.Context, _ *mcp.CallToolRequest, in streamInput) (*mcp.CallToolResult, streamView, error) {
	connID, err := t.conns.Resolve(ctx, in.Connection)
	if err != nil {
		return nil, streamView{}, err
	}
	info, err := t.streams.GetStreamInfo(ctx, connID, in.Stream)
	if err != nil {
		return nil, streamView{}, err
	}
	return nil, *converter.Convert(info, &streamView{}, mcptransport.ViewCodecs), nil
}

func (t *Toolset) getStreamRelations(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	in relationsInput,
) (*mcp.CallToolResult, relationsOutput, error) {
	connID, err := t.conns.Resolve(ctx, in.Connection)
	if err != nil {
		return nil, relationsOutput{}, err
	}
	relations, err := t.streams.GetStreamRelations(ctx, connID)
	if err != nil {
		return nil, relationsOutput{}, err
	}
	return nil, relationsView(relations, in.Stream), nil
}

// relationsView keeps the links touching stream (all of them when it is empty) and the nodes they reach.
func relationsView(relations *entities.StreamRelations, stream string) relationsOutput {
	touches := func(e entities.StreamRelationEdge) bool { return stream == "" || e.From == stream || e.To == stream }
	reached := map[string]bool{}
	for _, e := range relations.Edges {
		if touches(e) {
			reached[e.From], reached[e.To] = true, true
		}
	}
	return relationsOutput{
		Nodes: mcptransport.Items(slices.ToWithFilter(relations.Nodes,
			func(n entities.StreamRelationNode) bool { return reached[n.ID] },
			func(n entities.StreamRelationNode) relationNodeView {
				return *converter.Convert(&n, &relationNodeView{}, mcptransport.ViewCodecs)
			},
		)),
		Relations: mcptransport.Items(slices.ToWithFilter(relations.Edges, touches,
			func(e entities.StreamRelationEdge) relationView {
				return *converter.Convert(&e, &relationView{}, mcptransport.ViewCodecs)
			},
		)),
	}
}

func (t *Toolset) listConsumers(ctx context.Context, _ *mcp.CallToolRequest, in listConsumersInput) (*mcp.CallToolResult, listConsumersOutput, error) {
	connID, err := t.conns.Resolve(ctx, in.Connection)
	if err != nil {
		return nil, listConsumersOutput{}, err
	}

	views, unreadable, err := t.consumers(ctx, connID, in.Stream)
	if err != nil {
		return nil, listConsumersOutput{}, err
	}
	sort.SliceStable(views, func(i, j int) bool { return views[i].NumPending > views[j].NumPending })
	return nil, listConsumersOutput{
		Consumers:         mcptransport.Items(views[:min(len(views), mcptransport.Limit(in.Limit, defaultConsumersLimit, maxConsumersLimit))]),
		Total:             len(views),
		UnreadableStreams: unreadable,
	}, nil
}

func (t *Toolset) consumers(ctx context.Context, connID, stream string) ([]consumerView, []unreadableStreamView, error) {
	if stream != "" {
		infos, err := t.streams.GetStreamConsumers(ctx, connID, stream)
		if err != nil {
			return nil, nil, err
		}
		return slices.To(infos, consumerViewOf), nil, nil
	}
	overview, err := t.stats.GetConsumersOverview(ctx, connID)
	if err != nil {
		return nil, nil, err
	}
	unreadable := slices.To(overview.UnreadableStreams, func(u entities.UnreadableStream) unreadableStreamView {
		return unreadableStreamViewOf(ctx, u)
	})
	return slices.To(overview.Consumers, consumerViewOf), unreadable, nil
}

// unreadableStreamViewOf names the refused permission, or the failure, that hid a stream's consumers.
func unreadableStreamViewOf(ctx context.Context, u entities.UnreadableStream) unreadableStreamView {
	if u.Access != nil {
		return unreadableStreamView{Stream: u.Stream, Reason: "no permission to " + u.Access.Operation + " to " + u.Access.Subject}
	}
	return unreadableStreamView{Stream: u.Stream, Reason: grpchelpers.ErrorMessage(ctx, u.Err)}
}

// consumerViewOf merges a consumer's state and config into one view.
func consumerViewOf(c entities.ConsumerInfo) consumerView {
	v := converter.Convert(&c, &consumerView{}, mcptransport.ViewCodecs, converter.WithIgnoreFields("PriorityGroups"))
	if c.Config != nil {
		v = converter.Convert(c.Config, v, mcptransport.ViewCodecs, converter.WithIgnoreFields("Name"))
	}
	for _, state := range c.PriorityGroups {
		if state.PinnedClientID != "" {
			v.Pinned = append(v.Pinned, *converter.Convert(&state, &priorityGroupView{}))
		}
	}
	return *v
}
