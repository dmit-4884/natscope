// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mappingspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/mappings/v1/mappings"
	connectionspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/connections"
	templatespb "github.com/dmit-4884/natscope/proto/gen/services/grpc/templates/v1/templates"
)

// TestPagination creates more items than one page and walks next_page_token
// to exhaustion, asserting no duplicates/gaps and total_size matches created count.
func TestPagination(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()

	const total = 25
	const pageSize = 10

	t.Run("mappings", func(t *testing.T) {
		for i := 0; i < total; i++ {
			_, err := env.mappings.CreateMapping(ctx, connect.NewRequest(&mappingspb.CreateMappingRequest{
				Pattern:     fmt.Sprintf("page.test.%03d", i),
				MessageType: "x.Y",
				SourceId:    "page-source",
			}))
			require.NoError(t, err)
		}

		seen := map[string]bool{}
		token := ""
		pages := 0
		var reportedTotal int64
		for {
			resp, err := env.mappings.ListMappings(ctx, connect.NewRequest(&mappingspb.ListMappingsRequest{
				PageSize: pageSize, PageToken: token, IncludeTotalCount: true,
			}))
			require.NoError(t, err)
			pages++
			require.LessOrEqual(t, pages, total, "must not loop forever")
			for _, m := range resp.Msg.GetMappings() {
				assert.False(t, seen[m.GetId()], "item %s seen twice across pages", m.GetId())
				seen[m.GetId()] = true
			}
			reportedTotal = resp.Msg.GetTotalSize()
			next := resp.Msg.GetNextPageToken()
			if next == "" {
				break
			}
			token = next
		}
		// setupE2E boots a fresh app+storage per test, so all `total` mappings
		// created here are the only ones — the count below is exact.
		assert.GreaterOrEqual(t, len(seen), total)
		assert.EqualValues(t, len(seen), reportedTotal, "total_size must match the number of distinct items paged through")
	})

	t.Run("templates", func(t *testing.T) {
		for i := 0; i < total; i++ {
			_, err := env.templates.CreateTemplate(ctx, connect.NewRequest(&templatespb.CreateTemplateRequest{
				Name: fmt.Sprintf("page-template-%03d", i), Subject: "x", Data: "{}",
			}))
			require.NoError(t, err)
		}

		seen := map[string]bool{}
		token := ""
		pages := 0
		var reportedTotal int64
		for {
			resp, err := env.templates.ListTemplates(ctx, connect.NewRequest(&templatespb.ListTemplatesRequest{
				PageSize: pageSize, PageToken: token, IncludeTotalCount: true,
			}))
			require.NoError(t, err)
			pages++
			require.LessOrEqual(t, pages, total, "must not loop forever")
			for _, tmpl := range resp.Msg.GetTemplates() {
				assert.False(t, seen[tmpl.GetId()], "item %s seen twice across pages", tmpl.GetId())
				seen[tmpl.GetId()] = true
			}
			reportedTotal = resp.Msg.GetTotalSize()
			next := resp.Msg.GetNextPageToken()
			if next == "" {
				break
			}
			token = next
		}
		assert.GreaterOrEqual(t, len(seen), total)
		assert.EqualValues(t, len(seen), reportedTotal)
	})

	t.Run("connections (no total_size field, just exhaustion)", func(t *testing.T) {
		for i := 0; i < total; i++ {
			_, err := env.connections.CreateConnection(ctx, connect.NewRequest(&connectionspb.CreateConnectionRequest{
				Name: fmt.Sprintf("page-conn-%03d", i), Urls: []string{"nats://127.0.0.1:1"},
			}))
			require.NoError(t, err)
		}

		seen := map[string]bool{}
		token := ""
		pages := 0
		for {
			resp, err := env.connections.ListConnections(ctx, connect.NewRequest(&connectionspb.ListConnectionsRequest{
				PageSize: pageSize, PageToken: token,
			}))
			require.NoError(t, err)
			pages++
			require.LessOrEqual(t, pages, total, "must not loop forever")
			for _, c := range resp.Msg.GetConnections() {
				assert.False(t, seen[c.GetId()], "item %s seen twice across pages", c.GetId())
				seen[c.GetId()] = true
			}
			next := resp.Msg.GetNextPageToken()
			if next == "" {
				break
			}
			token = next
		}
		assert.GreaterOrEqual(t, len(seen), total)
	})
}
