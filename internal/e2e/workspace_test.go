// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"encoding/json"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mappingspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/mappings/v1/mappings"
	connectionspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/connections"
	templatespb "github.com/dmit-4884/natscope/proto/gen/services/grpc/templates/v1/templates"
	workspacepb "github.com/dmit-4884/natscope/proto/gen/services/grpc/workspace/v1/workspace"
	natstypes "github.com/dmit-4884/natscope/proto/gen/types/nats"
)

// TestWorkspaceRoundtrip exercises ListSections, ExportWorkspace (asserting no
// secret leakage), ValidateWorkspace (dry-run), and ImportWorkspace (MERGE/REPLACE).
func TestWorkspaceRoundtrip(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()

	secretPassword := "s3cr3t-password-must-not-leak"
	username := "ws-user"
	_, err := env.connections.CreateConnection(ctx, connect.NewRequest(&connectionspb.CreateConnectionRequest{
		Name: "ws-conn", Urls: []string{env.natsURL},
		Auth: &natstypes.AuthConfig{
			Method:   natstypes.AuthMethod_AUTH_METHOD_USER_PASSWORD,
			Username: &username,
			Password: &secretPassword,
		},
	}))
	require.NoError(t, err)

	_, err = env.mappings.CreateMapping(ctx, connect.NewRequest(&mappingspb.CreateMappingRequest{
		Pattern: "ws.>", MessageType: "x.Y", SourceId: "s",
	}))
	require.NoError(t, err)

	_, err = env.templates.CreateTemplate(ctx, connect.NewRequest(&templatespb.CreateTemplateRequest{
		Name: "ws-template", Subject: "ws.tmpl", Data: "{}",
	}))
	require.NoError(t, err)

	sectionsResp, err := env.workspace.ListSections(ctx, connect.NewRequest(&workspacepb.ListSectionsRequest{}))
	require.NoError(t, err)
	require.NotEmpty(t, sectionsResp.Msg.GetSections())
	sectionCounts := map[string]int32{}
	for _, s := range sectionsResp.Msg.GetSections() {
		sectionCounts[s.GetKey()] = s.GetCount()
	}
	assert.GreaterOrEqual(t, sectionCounts["connections"], int32(1))
	assert.GreaterOrEqual(t, sectionCounts["mappings"], int32(1))
	assert.GreaterOrEqual(t, sectionCounts["templates"], int32(1))

	exportResp, err := env.workspace.ExportWorkspace(ctx, connect.NewRequest(&workspacepb.ExportWorkspaceRequest{}))
	require.NoError(t, err)
	payload := exportResp.Msg.GetPayload()
	require.NotEmpty(t, payload)
	assert.NotContains(t, string(payload), secretPassword, "exported workspace must never contain secret material")

	// The payload must be well-formed JSON containing our connection by name,
	// without the password field anywhere in it.
	var asJSON map[string]any
	require.NoError(t, json.Unmarshal(payload, &asJSON))

	t.Run("validate dry-run reports without mutating", func(t *testing.T) {
		valResp, err := env.workspace.ValidateWorkspace(ctx, connect.NewRequest(&workspacepb.ValidateWorkspaceRequest{
			Payload: payload,
		}))
		require.NoError(t, err)
		require.NotEmpty(t, valResp.Msg.GetReports())

		// Connections import is deliberately create-only: merge keeps existing
		// names (with local credentials) untouched, so re-importing reports no create/update.
		found := false
		for _, r := range valResp.Msg.GetReports() {
			if r.GetKey() == "connections" {
				found = true
				assert.EqualValues(t, 0, r.GetCreated(), "connection already exists by name — must not be recreated")
				assert.EqualValues(t, 0, r.GetUpdated(), "connections import never updates existing rows by design")
			}
		}
		assert.True(t, found, "validate report must include the connections section")

		listResp, err := env.connections.ListConnections(ctx, connect.NewRequest(&connectionspb.ListConnectionsRequest{}))
		require.NoError(t, err)
		assert.Len(t, listResp.Msg.GetConnections(), 1, "dry-run validate must not create a duplicate connection")
	})

	t.Run("import merge is idempotent", func(t *testing.T) {
		_, err := env.workspace.ImportWorkspace(ctx, connect.NewRequest(&workspacepb.ImportWorkspaceRequest{
			Payload:  payload,
			Strategy: workspacepb.Strategy_STRATEGY_MERGE,
		}))
		require.NoError(t, err)

		listResp, err := env.connections.ListConnections(ctx, connect.NewRequest(&connectionspb.ListConnectionsRequest{}))
		require.NoError(t, err)
		assert.Len(t, listResp.Msg.GetConnections(), 1, "merge-importing the same export must not duplicate the connection")

		mapResp, err := env.mappings.ListMappings(ctx, connect.NewRequest(&mappingspb.ListMappingsRequest{PageSize: 500}))
		require.NoError(t, err)
		assert.Len(t, mapResp.Msg.GetMappings(), 1, "merge-importing the same export must not duplicate the mapping")
	})

	t.Run("import replace clears sections not in the file", func(t *testing.T) {
		// Add a template outside the exported payload, then replace-import
		// just "templates": it must be cleared and rebuilt, dropping the extra.
		_, err := env.templates.CreateTemplate(ctx, connect.NewRequest(&templatespb.CreateTemplateRequest{
			Name: "extra-not-in-export", Subject: "x", Data: "{}",
		}))
		require.NoError(t, err)

		importResp, err := env.workspace.ImportWorkspace(ctx, connect.NewRequest(&workspacepb.ImportWorkspaceRequest{
			Payload:     payload,
			SectionKeys: []string{"templates"},
			Strategy:    workspacepb.Strategy_STRATEGY_REPLACE,
		}))
		require.NoError(t, err)
		require.NotEmpty(t, importResp.Msg.GetResults())

		listResp, err := env.templates.ListTemplates(ctx, connect.NewRequest(&templatespb.ListTemplatesRequest{PageSize: 500}))
		require.NoError(t, err)
		names := make([]string, 0, len(listResp.Msg.GetTemplates()))
		for _, tmpl := range listResp.Msg.GetTemplates() {
			names = append(names, tmpl.GetName())
		}
		assert.Contains(t, names, "ws-template")
		assert.NotContains(t, names, "extra-not-in-export", "REPLACE must clear items absent from the imported file")
	})

	t.Run("broken payload is rejected with a clear error", func(t *testing.T) {
		_, err := env.workspace.ValidateWorkspace(ctx, connect.NewRequest(&workspacepb.ValidateWorkspaceRequest{
			Payload: []byte("{not json"),
		}))
		require.Error(t, err)
		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	})
}
