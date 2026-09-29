// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package streamgraph

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
)

func stream(name string, subjects ...string) entities.StreamInfo {
	return entities.StreamInfo{Config: entities.StreamConfig{Name: name, Subjects: subjects}, Raw: `{"config":{}}`}
}

type edgeKey struct {
	kind     entities.StreamRelationKind
	from, to string
}

func edgeKeys(g *entities.StreamRelations) []edgeKey {
	keys := make([]edgeKey, 0, len(g.Edges))
	for _, e := range g.Edges {
		keys = append(keys, edgeKey{kind: e.Kind, from: e.From, to: e.To})
	}
	return keys
}

func nodeKinds(g *entities.StreamRelations) map[string]entities.StreamNodeKind {
	kinds := map[string]entities.StreamNodeKind{}
	for _, n := range g.Nodes {
		kinds[n.ID] = n.Kind
	}
	return kinds
}

func TestBuild(t *testing.T) {
	t.Parallel()

	t.Run("StreamsWithoutRelationsAreLeftOut", func(t *testing.T) {
		t.Parallel()
		g := Build([]entities.StreamInfo{stream("A", "a.>"), stream("B", "b.>")})
		require.Empty(t, g.Nodes)
		require.Empty(t, g.Edges)
	})

	t.Run("SourcesAndMirrors", func(t *testing.T) {
		t.Parallel()
		agg := stream("AGG")
		agg.Config.Sources = []*entities.StreamSourceRef{{Name: "EU"}, {Name: "GONE"}, {Name: "US", External: &entities.ExternalStreamRef{ApiPrefix: "$JS.us.API"}}}
		backup := stream("KV_backup")
		backup.Config.Mirror = &entities.StreamSourceRef{Name: "AGG"}
		backup.Mirror = &entities.StreamSourceInfo{Name: "AGG", Lag: 3, Active: -1}

		g := Build([]entities.StreamInfo{backup, agg, stream("EU", "eu.>"), stream("LONELY", "x")})

		require.Equal(t, []edgeKey{
			{kind: entities.StreamRelationSource, from: "EU", to: "AGG"},
			{kind: entities.StreamRelationSource, from: "GONE", to: "AGG"},
			{kind: entities.StreamRelationSource, from: "external $JS.us.API US", to: "AGG"},
			{kind: entities.StreamRelationMirror, from: "AGG", to: "KV_backup"},
		}, edgeKeys(g))
		require.Equal(t, map[string]entities.StreamNodeKind{
			"AGG":                    entities.StreamNodeStream,
			"EU":                     entities.StreamNodeStream,
			"GONE":                   entities.StreamNodeMissing,
			"KV_backup":              entities.StreamNodeKV,
			"external $JS.us.API US": entities.StreamNodeExternal,
		}, nodeKinds(g))
		require.Equal(t, uint64(3), g.Edges[3].State.Lag)
		for _, n := range g.Nodes {
			if n.Info != nil {
				require.Empty(t, n.Info.Raw)
			}
		}
		require.Equal(t, `{"config":{}}`, agg.Raw)
	})

	t.Run("MatchesSourceStatesByFilter", func(t *testing.T) {
		t.Parallel()
		agg := stream("AGG")
		agg.Config.Sources = []*entities.StreamSourceRef{
			{Name: "ORDERS", FilterSubject: "orders.eu.>"},
			{Name: "ORDERS", SubjectTransforms: []entities.SubjectTransformConfig{{Source: "orders.us.>", Destination: "us.>"}}},
		}
		agg.Sources = []*entities.StreamSourceInfo{
			{Name: "ORDERS", Lag: 2, SubjectTransforms: []entities.SubjectTransformConfig{{Source: "orders.us.>", Destination: "us.>"}}},
			{Name: "ORDERS", Lag: 1, FilterSubject: "orders.eu.>", SubjectTransforms: []entities.SubjectTransformConfig{{Source: "orders.eu.>"}}},
		}

		g := Build([]entities.StreamInfo{agg, stream("ORDERS", "orders.>")})

		require.Len(t, g.Edges, 2)
		require.Equal(t, uint64(1), g.Edges[0].State.Lag)
		require.Equal(t, uint64(2), g.Edges[1].State.Lag)
	})

	t.Run("FallbackNeverTakesAnExactState", func(t *testing.T) {
		t.Parallel()
		agg := stream("AGG")
		agg.Config.Sources = []*entities.StreamSourceRef{
			{Name: "ORDERS", FilterSubject: "orders.eu.>"},
			{Name: "ORDERS", FilterSubject: "orders.us.>"},
		}
		agg.Sources = []*entities.StreamSourceInfo{
			{Name: "ORDERS", Lag: 2, FilterSubject: "orders.us.>"},
			{Name: "ORDERS", Lag: 1, FilterSubject: "orders.EU.>"},
		}

		g := Build([]entities.StreamInfo{agg, stream("ORDERS", "orders.>")})

		require.Len(t, g.Edges, 2)
		require.Equal(t, uint64(1), g.Edges[0].State.Lag)
		require.Equal(t, uint64(2), g.Edges[1].State.Lag)
	})

	t.Run("CrossAccountSourcesShareOnePlaceholder", func(t *testing.T) {
		t.Parallel()
		shop := &entities.ExternalStreamRef{ApiPrefix: "shop.API", DeliverPrefix: "analytics.deliver"}
		sales := stream("SALES")
		sales.Config.Sources = []*entities.StreamSourceRef{
			{Name: "ORDERS", FilterSubject: "*.orders.paid", External: shop},
			{Name: "PAYMENTS", FilterSubject: "payments.*.captured", External: shop},
			{Name: "PAYMENTS", FilterSubject: "payments.*.refunded", External: shop},
		}
		sales.Sources = []*entities.StreamSourceInfo{
			{Name: "PAYMENTS", Lag: 3, FilterSubject: "payments.*.refunded", External: shop},
			{Name: "ORDERS", Lag: 1, FilterSubject: "*.orders.paid", External: shop},
			{Name: "PAYMENTS", Lag: 2, FilterSubject: "payments.*.captured", External: shop},
		}
		local := stream("PAYMENTS", "payments.>")
		last24h := stream("SALES_LAST_24H")
		last24h.Config.Mirror = &entities.StreamSourceRef{Name: "SALES"}

		g := Build([]entities.StreamInfo{sales, local, last24h})

		require.Equal(t, []edgeKey{
			{kind: entities.StreamRelationSource, from: "external shop.API ORDERS", to: "SALES"},
			{kind: entities.StreamRelationSource, from: "external shop.API PAYMENTS", to: "SALES"},
			{kind: entities.StreamRelationSource, from: "external shop.API PAYMENTS", to: "SALES"},
			{kind: entities.StreamRelationMirror, from: "SALES", to: "SALES_LAST_24H"},
		}, edgeKeys(g))
		require.Equal(t, []uint64{1, 2, 3}, []uint64{g.Edges[0].State.Lag, g.Edges[1].State.Lag, g.Edges[2].State.Lag})
		require.Nil(t, g.Edges[3].State)
		require.Len(t, g.Nodes, 4)
		require.NotContains(t, nodeKinds(g), "PAYMENTS")
	})

	t.Run("RepublishIntoCapturingStreams", func(t *testing.T) {
		t.Parallel()
		orders := stream("ORDERS", "orders.>")
		orders.Config.Republish = &entities.StreamRePublish{Src: "orders.>", Dest: "audit.{{wildcard(1)}}"}
		events := stream("EVENTS", "events.>")
		events.Config.Republish = &entities.StreamRePublish{Dest: "notify.events"}

		g := Build([]entities.StreamInfo{orders, events, stream("AUDIT", "audit.*"), stream("ALL_AUDIT", "audit.>"), stream("OTHER", "other")})

		require.Equal(t, []edgeKey{
			{kind: entities.StreamRelationRepublish, from: "EVENTS", to: "subject notify.events"},
			{kind: entities.StreamRelationRepublish, from: "ORDERS", to: "ALL_AUDIT"},
			{kind: entities.StreamRelationRepublish, from: "ORDERS", to: "AUDIT"},
		}, edgeKeys(g))
		require.Equal(t, entities.StreamNodeSubject, nodeKinds(g)["subject notify.events"])
		require.Equal(t, "audit.{{wildcard(1)}}", g.Edges[1].Republish.Dest)
	})

	t.Run("ObjectStoreKind", func(t *testing.T) {
		t.Parallel()
		copyStore := stream("OBJ_copy")
		copyStore.Config.Mirror = &entities.StreamSourceRef{Name: "OBJ_files"}
		g := Build([]entities.StreamInfo{copyStore, stream("OBJ_files", "$O.files.>")})
		require.Equal(t, entities.StreamNodeObjectStore, nodeKinds(g)["OBJ_files"])
	})
}
