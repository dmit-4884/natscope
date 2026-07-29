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
	managementpb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/management"
	messagespb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/messages"
	streamspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/streams"
	selectionspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/proto/v1/selections"
	sourcespb "github.com/dmit-4884/natscope/proto/gen/services/grpc/proto/v1/sources"
	templatespb "github.com/dmit-4884/natscope/proto/gen/services/grpc/templates/v1/templates"
)

const unknownUUID = "00000000-0000-0000-0000-000000000000"

// TestNotFound drives every lookup-by-id RPC with a well-formed but
// nonexistent id and asserts connect.CodeNotFound, not Internal/FailedPrecondition.
func TestNotFound(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()

	// A real connection is needed so "unknown stream/consumer/etc on a real
	// connection" exercises the NATS-facing not-found paths too.
	createResp, err := env.connections.CreateConnection(ctx, connect.NewRequest(&connectionspb.CreateConnectionRequest{
		Name: "notfound-conn",
		Urls: []string{env.natsURL},
	}))
	require.NoError(t, err)
	connID := createResp.Msg.GetConnection().GetId()

	cases := []struct {
		name string
		call func() error
	}{
		{"connections.Update unknown id", func() error {
			_, err := env.connections.UpdateConnection(ctx, connect.NewRequest(&connectionspb.UpdateConnectionRequest{Id: unknownUUID}))
			return err
		}},
		{"connections.Delete unknown id", func() error {
			_, err := env.connections.DeleteConnection(ctx, connect.NewRequest(&connectionspb.DeleteConnectionRequest{Id: unknownUUID}))
			return err
		}},
		{"connections.Duplicate unknown id", func() error {
			_, err := env.connections.DuplicateConnection(ctx, connect.NewRequest(&connectionspb.DuplicateConnectionRequest{Id: unknownUUID, Name: "x"}))
			return err
		}},
		{"mappings.Delete unknown id", func() error {
			_, err := env.mappings.DeleteMapping(ctx, connect.NewRequest(&mappingspb.DeleteMappingRequest{Id: unknownUUID}))
			return err
		}},
		{"templates.Get unknown id", func() error {
			_, err := env.templates.GetTemplate(ctx, connect.NewRequest(&templatespb.GetTemplateRequest{Id: unknownUUID}))
			return err
		}},
		{"templates.Update unknown id", func() error {
			name := "x"
			_, err := env.templates.UpdateTemplate(ctx, connect.NewRequest(&templatespb.UpdateTemplateRequest{Id: unknownUUID, Name: &name}))
			return err
		}},
		{"templates.Delete unknown id", func() error {
			_, err := env.templates.DeleteTemplate(ctx, connect.NewRequest(&templatespb.DeleteTemplateRequest{Id: unknownUUID}))
			return err
		}},
		{"sources.GetSource unknown id", func() error {
			_, err := env.sources.GetSource(ctx, connect.NewRequest(&sourcespb.GetSourceRequest{Id: unknownUUID}))
			return err
		}},
		{"sources.DeleteSource unknown id", func() error {
			_, err := env.sources.DeleteSource(ctx, connect.NewRequest(&sourcespb.DeleteSourceRequest{Id: unknownUUID}))
			return err
		}},
		{"selections.DeleteSelection unknown id", func() error {
			_, err := env.selections.DeleteSelection(ctx, connect.NewRequest(&selectionspb.DeleteSelectionRequest{Id: unknownUUID}))
			return err
		}},
		{"streams.GetStream unknown stream on real connection", func() error {
			_, err := env.streams.GetStream(ctx, connect.NewRequest(&streamspb.GetStreamRequest{
				ConnectionId: connID, StreamName: "NOPE_DOES_NOT_EXIST",
			}))
			return err
		}},
		{"management.DeleteStream unknown stream", func() error {
			_, err := env.management.DeleteStream(ctx, connect.NewRequest(&managementpb.DeleteStreamRequest{
				ConnectionId: connID, StreamName: "NOPE_DOES_NOT_EXIST",
			}))
			return err
		}},
		{"management.CreateConsumer unknown stream", func() error {
			_, err := env.management.CreateConsumer(ctx, connect.NewRequest(&managementpb.CreateConsumerRequest{
				ConnectionId: connID, StreamName: "NOPE_DOES_NOT_EXIST", Name: "c1",
			}))
			return err
		}},
		{"management.DeleteConsumer unknown consumer", func() error {
			_, err := env.management.DeleteConsumer(ctx, connect.NewRequest(&managementpb.DeleteConsumerRequest{
				ConnectionId: connID, StreamName: "NOPE_DOES_NOT_EXIST", ConsumerName: "NOPE",
			}))
			return err
		}},
		{"management.GetKVBucket unknown bucket", func() error {
			_, err := env.management.GetKVBucket(ctx, connect.NewRequest(&managementpb.GetKVBucketRequest{
				ConnectionId: connID, Bucket: "NOPE_BUCKET",
			}))
			return err
		}},
		{"management.GetKVKey unknown key", func() error {
			_, err := env.management.GetKVKey(ctx, connect.NewRequest(&managementpb.GetKVKeyRequest{
				ConnectionId: connID, Bucket: "NOPE_BUCKET", Key: "NOPE_KEY",
			}))
			return err
		}},
		{"management.GetObjectBucket unknown bucket", func() error {
			_, err := env.management.GetObjectBucket(ctx, connect.NewRequest(&managementpb.GetObjectBucketRequest{
				ConnectionId: connID, Bucket: "NOPE_OBJ_BUCKET",
			}))
			return err
		}},
		{"management.GetObject unknown object", func() error {
			_, err := env.management.GetObject(ctx, connect.NewRequest(&managementpb.GetObjectRequest{
				ConnectionId: connID, Bucket: "NOPE_OBJ_BUCKET", Name: "nope.txt",
			}))
			return err
		}},
		{"messages.GetMessage unknown stream", func() error {
			_, err := env.messages.GetMessage(ctx, connect.NewRequest(&messagespb.GetMessageRequest{
				ConnectionId: connID, StreamName: "NOPE_DOES_NOT_EXIST", Sequence: 1,
			}))
			return err
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call()
			require.Error(t, err, "expected an error")
			assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err), "unexpected code for %q: %v", tc.name, err)
		})
	}
}
