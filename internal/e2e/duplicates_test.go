// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mappingspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/mappings/v1/mappings"
	connectionspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/connections"
	sourcespb "github.com/dmit-4884/natscope/proto/gen/services/grpc/proto/v1/sources"
	protopb "github.com/dmit-4884/natscope/proto/gen/types/proto"
)

// TestDuplicates asserts connect.CodeAlreadyExists on a second write for the
// three enforced uniqueness constraints: connection name, source name, mapping (pattern, source_id).
func TestDuplicates(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()

	t.Run("connection name", func(t *testing.T) {
		_, err := env.connections.CreateConnection(ctx, connect.NewRequest(&connectionspb.CreateConnectionRequest{
			Name: "dup-conn", Urls: []string{env.natsURL},
		}))
		require.NoError(t, err)

		_, err = env.connections.CreateConnection(ctx, connect.NewRequest(&connectionspb.CreateConnectionRequest{
			Name: "dup-conn", Urls: []string{env.natsURL},
		}))
		require.Error(t, err)
		assert.Equal(t, connect.CodeAlreadyExists, connect.CodeOf(err))
	})

	t.Run("proto source name", func(t *testing.T) {
		_, err := env.sources.CreateSource(ctx, connect.NewRequest(&sourcespb.CreateSourceRequest{
			Name: "dup-source", SourceType: protopb.SourceType_SOURCE_TYPE_FILES,
			Files: []string{"/nonexistent/does-not-matter-for-this-test.proto"},
		}))
		require.NoError(t, err)

		_, err = env.sources.CreateSource(ctx, connect.NewRequest(&sourcespb.CreateSourceRequest{
			Name: "dup-source", SourceType: protopb.SourceType_SOURCE_TYPE_FILES,
			Files: []string{"/nonexistent/does-not-matter-for-this-test.proto"},
		}))
		require.Error(t, err)
		assert.Equal(t, connect.CodeAlreadyExists, connect.CodeOf(err))
	})

	t.Run("mapping pattern+source_id", func(t *testing.T) {
		_, err := env.mappings.CreateMapping(ctx, connect.NewRequest(&mappingspb.CreateMappingRequest{
			Pattern: "dup.pattern.>", MessageType: "x.Y", SourceId: "same-source",
		}))
		require.NoError(t, err)

		_, err = env.mappings.CreateMapping(ctx, connect.NewRequest(&mappingspb.CreateMappingRequest{
			Pattern: "dup.pattern.>", MessageType: "x.Z", SourceId: "same-source",
		}))
		require.Error(t, err)
		assert.Equal(t, connect.CodeAlreadyExists, connect.CodeOf(err))

		// Same pattern, DIFFERENT source_id must be allowed — uniqueness is the
		// (pattern, source_id) pair, not the pattern alone.
		_, err = env.mappings.CreateMapping(ctx, connect.NewRequest(&mappingspb.CreateMappingRequest{
			Pattern: "dup.pattern.>", MessageType: "x.Y", SourceId: "different-source",
		}))
		require.NoError(t, err, "same pattern with a different source_id must be allowed")
	})
}
