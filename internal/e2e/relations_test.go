// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	streamspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/streams"
	natstypes "github.com/dmit-4884/natscope/proto/gen/types/nats"
)

func TestGetStreamRelations(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()
	connID := createTestConnection(t, env, "stream-relations", env.natsURL, nil)

	nc, err := nats.Connect(env.natsURL)
	require.NoError(t, err)
	defer nc.Close()
	js, err := jetstream.New(nc)
	require.NoError(t, err)

	for _, cfg := range []jetstream.StreamConfig{
		{Name: "ORDERS", Subjects: []string{"orders.>"}},
		{Name: "AUDIT", Subjects: []string{"audit.>"}},
		{Name: "LONELY", Subjects: []string{"lonely"}},
		{Name: "EVENTS", Subjects: []string{"events.>"}, RePublish: &jetstream.RePublish{Source: "events.>", Destination: "audit.events.>"}},
		{Name: "AGG", Sources: []*jetstream.StreamSource{{Name: "ORDERS", FilterSubject: "orders.eu.>"}, {Name: "GHOST"}}},
		{Name: "BACKUP", Mirror: &jetstream.StreamSource{Name: "ORDERS"}},
	} {
		_, err := js.CreateStream(ctx, cfg)
		require.NoError(t, err, cfg.Name)
	}
	_, err = js.Publish(ctx, "orders.eu.1", []byte("{}"))
	require.NoError(t, err)

	get := func() *streamspb.GetStreamRelationsResponse {
		resp, err := env.streams.GetStreamRelations(ctx, connect.NewRequest(&streamspb.GetStreamRelationsRequest{ConnectionId: connID}))
		require.NoError(t, err)
		return resp.Msg
	}
	edge := func(msg *streamspb.GetStreamRelationsResponse, kind natstypes.StreamRelationKind, from, to string) *natstypes.StreamRelationEdge {
		for _, e := range msg.GetEdges() {
			if e.GetKind() == kind && e.GetFrom() == from && e.GetTo() == to {
				return e
			}
		}
		return nil
	}

	var msg *streamspb.GetStreamRelationsResponse
	require.Eventually(t, func() bool {
		msg = get()
		mirror := edge(msg, natstypes.StreamRelationKind_STREAM_RELATION_KIND_MIRROR, "ORDERS", "BACKUP")
		ghost := edge(msg, natstypes.StreamRelationKind_STREAM_RELATION_KIND_SOURCE, "GHOST", "AGG")
		return mirror != nil && mirror.GetState().GetActive().AsDuration() >= 0 &&
			ghost != nil && ghost.GetState().GetError() != ""
	}, 15*time.Second, 200*time.Millisecond, "mirror never became active or the missing source never reported an error")

	source := edge(msg, natstypes.StreamRelationKind_STREAM_RELATION_KIND_SOURCE, "ORDERS", "AGG")
	require.NotNil(t, source)
	assert.Equal(t, "orders.eu.>", source.GetSource().GetFilterSubject())
	assert.Equal(t, "ORDERS", source.GetState().GetName())

	republish := edge(msg, natstypes.StreamRelationKind_STREAM_RELATION_KIND_REPUBLISH, "EVENTS", "AUDIT")
	require.NotNil(t, republish)
	assert.Equal(t, "audit.events.>", republish.GetRepublish().GetDest())

	kinds := map[string]natstypes.StreamNodeKind{}
	for _, n := range msg.GetNodes() {
		kinds[n.GetId()] = n.GetKind()
		if n.GetKind() == natstypes.StreamNodeKind_STREAM_NODE_KIND_STREAM {
			assert.Empty(t, n.GetInfo().GetRaw(), n.GetId())
		}
	}
	assert.Equal(t, map[string]natstypes.StreamNodeKind{
		"AGG":    natstypes.StreamNodeKind_STREAM_NODE_KIND_STREAM,
		"AUDIT":  natstypes.StreamNodeKind_STREAM_NODE_KIND_STREAM,
		"BACKUP": natstypes.StreamNodeKind_STREAM_NODE_KIND_STREAM,
		"EVENTS": natstypes.StreamNodeKind_STREAM_NODE_KIND_STREAM,
		"GHOST":  natstypes.StreamNodeKind_STREAM_NODE_KIND_MISSING,
		"ORDERS": natstypes.StreamNodeKind_STREAM_NODE_KIND_STREAM,
	}, kinds)
}
