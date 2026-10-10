// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package streams

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"

	mcptransport "github.com/dmit-4884/natscope/internal/transports/mcp"
)

func TestStreamViewRendersEnumsAndDurations(t *testing.T) {
	t.Parallel()
	info := &entities.StreamInfo{
		Config: entities.StreamConfig{
			Name: "ORDERS", Subjects: []string{"orders.>"}, Retention: entities.RetentionWorkQueue,
			Storage: entities.StorageMemory, Discard: entities.DiscardNew, MaxAge: 36 * time.Hour,
			Compression: entities.CompressionS2, Mirror: &entities.StreamSourceRef{Name: "UPSTREAM"},
		},
		State: &entities.StreamState{Msgs: 12, LastSeq: 40},
	}

	v := converter.Convert(info, &streamView{}, mcptransport.ViewCodecs)

	assert.Equal(t, "workqueue", v.Config.Retention)
	assert.Equal(t, "memory", v.Config.Storage)
	assert.Equal(t, "new", v.Config.Discard)
	assert.Equal(t, "s2", v.Config.Compression)
	assert.Equal(t, "36h0m0s", v.Config.MaxAge)
	require.NotNil(t, v.Config.Mirror)
	assert.Equal(t, "UPSTREAM", v.Config.Mirror.Name)
	require.NotNil(t, v.State)
	assert.Equal(t, uint64(12), v.State.Msgs)

	summary := converter.Convert(info, &streamSummary{}, mcptransport.ViewCodecs)
	assert.Equal(t, "workqueue", summary.Config.Retention)
	assert.Equal(t, []string{"orders.>"}, summary.Config.Subjects)
}

func TestStreamViewRendersFeatureFlags(t *testing.T) {
	t.Parallel()
	info := &entities.StreamInfo{Config: entities.StreamConfig{
		Name: "FLAGS", AllowMsgCounter: true, AllowMsgSchedules: true, AllowAtomicPublish: true,
		SubjectDeleteMarkerTTL: 90 * time.Second, PersistMode: entities.PersistAsync, AllowBatchPublish: true,
	}}

	v := converter.Convert(info, &streamView{}, mcptransport.ViewCodecs)

	assert.True(t, v.Config.AllowMsgCounter)
	assert.True(t, v.Config.AllowMsgSchedules)
	assert.True(t, v.Config.AllowAtomicPublish)
	assert.Equal(t, "1m30s", v.Config.SubjectDeleteMarkerTTL)
	assert.Equal(t, "async", v.Config.PersistMode)
	assert.True(t, v.Config.AllowBatchPublish)

	plain := converter.Convert(&entities.StreamInfo{Config: entities.StreamConfig{Name: "PLAIN"}}, &streamView{}, mcptransport.ViewCodecs)
	assert.Equal(t, "default", plain.Config.PersistMode)
	assert.Empty(t, plain.Config.SubjectDeleteMarkerTTL)
}

func TestConsumerViewMergesConfig(t *testing.T) {
	t.Parallel()
	info := entities.ConsumerInfo{
		Name: "worker", Stream: "ORDERS", NumPending: 5, NumAckPending: 2,
		Config: &entities.ConsumerConfig{Name: "ignored", Durable: "worker", AckPolicy: entities.AckAll,
			DeliverPolicy: entities.DeliverLastPerSubject, AckWait: 30 * time.Second, FilterSubject: "orders.eu.>"},
	}

	v := converter.Convert(&info, &consumerView{}, mcptransport.ViewCodecs)
	v = converter.Convert(info.Config, v, mcptransport.ViewCodecs, converter.WithIgnoreFields("Name"))

	assert.Equal(t, "worker", v.Name)
	assert.Equal(t, "ORDERS", v.Stream)
	assert.Equal(t, uint64(5), v.NumPending)
	assert.Equal(t, "all", v.AckPolicy)
	assert.Equal(t, "last_per_subject", v.DeliverPolicy)
	assert.Equal(t, "30s", v.AckWait)
	assert.Equal(t, "orders.eu.>", v.FilterSubject)

	bare := consumerViewOf(entities.ConsumerInfo{Name: "w2", Stream: "S", Config: &entities.ConsumerConfig{AckPolicy: entities.AckNone}})
	assert.Equal(t, "none", bare.AckPolicy)
	assert.Equal(t, "all", bare.DeliverPolicy)
}

func TestConsumerViewRendersPriorityGroups(t *testing.T) {
	t.Parallel()
	pinnedAt := time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC)
	info := entities.ConsumerInfo{
		Name: "worker", Stream: "ORDERS",
		Config: &entities.ConsumerConfig{
			PriorityPolicy: entities.PriorityPinnedClient, PriorityGroups: []string{"jobs", "bulk"}, PinnedTTL: 2 * time.Minute,
		},
		PriorityGroups: []entities.PriorityGroupState{
			{Group: "jobs", PinnedClientID: "pin-1", PinnedTS: pinnedAt},
			{Group: "bulk"},
		},
	}

	v := consumerViewOf(info)

	assert.Equal(t, "pinned_client", v.PriorityPolicy)
	assert.Equal(t, "2m0s", v.PinnedTTL)
	assert.Equal(t, []string{"jobs", "bulk"}, v.PriorityGroups, "every configured group, not only the ones with state")
	require.Len(t, v.Pinned, 1, "only groups with a pinned client")
	assert.Equal(t, "jobs", v.Pinned[0].Group)
	assert.Equal(t, "pin-1", v.Pinned[0].PinnedClientID)
	assert.Equal(t, pinnedAt, v.Pinned[0].PinnedTS)

	plain := consumerViewOf(entities.ConsumerInfo{Name: "w", Config: &entities.ConsumerConfig{}})
	assert.Equal(t, "none", plain.PriorityPolicy)
	assert.Empty(t, plain.PinnedTTL)
	assert.Empty(t, plain.PriorityGroups)
	assert.Empty(t, plain.Pinned)

	flow := consumerViewOf(entities.ConsumerInfo{Name: "s", Config: &entities.ConsumerConfig{AckPolicy: entities.AckFlowControl}})
	assert.Equal(t, "flow_control", flow.AckPolicy)
}

func TestRelationsView(t *testing.T) {
	t.Parallel()
	relations := &entities.StreamRelations{
		Nodes: []entities.StreamRelationNode{
			{ID: "AGG", Name: "AGG", Kind: entities.StreamNodeStream},
			{ID: "BACKUP", Name: "BACKUP", Kind: entities.StreamNodeStream},
			{ID: "OTHER", Name: "OTHER", Kind: entities.StreamNodeStream},
			{ID: "TARGET", Name: "TARGET", Kind: entities.StreamNodeStream},
			{ID: "external $JS.hub.API ORDERS", Name: "ORDERS", Kind: entities.StreamNodeExternal,
				External: &entities.ExternalStreamRef{ApiPrefix: "$JS.hub.API"}},
		},
		Edges: []entities.StreamRelationEdge{
			{
				Kind: entities.StreamRelationSource, From: "external $JS.hub.API ORDERS", To: "AGG",
				Source: &entities.StreamSourceRef{Name: "ORDERS", SubjectTransforms: []entities.SubjectTransformConfig{{Source: "orders.>"}}},
				State:  &entities.StreamSourceInfo{Name: "ORDERS", Lag: 4, Active: 1500 * time.Millisecond, Error: "stream not found"},
			},
			{Kind: entities.StreamRelationMirror, From: "AGG", To: "BACKUP"},
			{Kind: entities.StreamRelationRepublish, From: "OTHER", To: "TARGET", Republish: &entities.StreamRePublish{Src: ">", Dest: "t.>"}},
		},
	}

	v := relationsView(relations, "AGG")

	require.Len(t, v.Relations, 2)
	assert.Equal(t, "source", v.Relations[0].Kind)
	assert.Equal(t, "orders.>", v.Relations[0].Source.SubjectTransforms[0].Source)
	assert.Equal(t, uint64(4), v.Relations[0].State.Lag)
	assert.Equal(t, "1.5s", v.Relations[0].State.Active)
	assert.Equal(t, "stream not found", v.Relations[0].State.Error)
	assert.Equal(t, "mirror", v.Relations[1].Kind)
	require.Len(t, v.Nodes, 3)
	assert.Equal(t, "external", v.Nodes[2].Kind)
	assert.Equal(t, "$JS.hub.API", v.Nodes[2].External.APIPrefix)

	all := relationsView(relations, "")
	assert.Len(t, all.Relations, 3)
	assert.Len(t, all.Nodes, 5)
	assert.Equal(t, "republish", all.Relations[2].Kind)
	assert.Equal(t, "t.>", all.Relations[2].Republish.Dest)
}

func TestUnreadableStreamViewNamesTheReason(t *testing.T) {
	t.Parallel()
	denied := unreadableStreamViewOf(t.Context(), entities.UnreadableStream{
		Stream: "SECRET",
		Access: &entities.AccessCheck{Status: entities.AccessDenied, Operation: "publish", Subject: "$JS.API.CONSUMER.LIST.SECRET"},
	})
	assert.Equal(t, unreadableStreamView{Stream: "SECRET", Reason: "no permission to publish to $JS.API.CONSUMER.LIST.SECRET"}, denied)

	broken := unreadableStreamViewOf(t.Context(), entities.UnreadableStream{Stream: "BROKEN", Err: errs.ErrJetStreamNotEnabled})
	assert.Equal(t, unreadableStreamView{Stream: "BROKEN", Reason: "jetstream not enabled"}, broken)

	leaky := unreadableStreamViewOf(t.Context(), entities.UnreadableStream{Stream: "LEAKY", Err: errors.New("dial tcp 10.0.0.7:4222")})
	assert.Equal(t, unreadableStreamView{Stream: "LEAKY", Reason: "internal error"}, leaky)
}
