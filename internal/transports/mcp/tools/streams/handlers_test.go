// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package streams

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"

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

	stats := converter.Convert(&entities.ConsumerStats{Name: "w2", Stream: "S", AckPolicy: entities.AckNone}, &consumerView{}, mcptransport.ViewCodecs)
	assert.Equal(t, "none", stats.AckPolicy)
	assert.Equal(t, "all", stats.DeliverPolicy)
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
