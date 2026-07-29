// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"google.golang.org/protobuf/types/known/timestamppb"

	historypb "github.com/dmit-4884/natscope/proto/gen/services/grpc/history/v1/history"
	mappingspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/mappings/v1/mappings"
	connectionspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/connections"
	livepb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/live"
	managementpb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/management"
	messagespb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/messages"
	publishpb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/publish"
	statspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/stats"
	streamspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/streams"
	codecpb "github.com/dmit-4884/natscope/proto/gen/services/grpc/proto/v1/codec"
	registrypb "github.com/dmit-4884/natscope/proto/gen/services/grpc/proto/v1/registry"
	selectionspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/proto/v1/selections"
	sourcespb "github.com/dmit-4884/natscope/proto/gen/services/grpc/proto/v1/sources"
	settingspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/settings/v1/settings"
	templatespb "github.com/dmit-4884/natscope/proto/gen/services/grpc/templates/v1/templates"
	workspacepb "github.com/dmit-4884/natscope/proto/gen/services/grpc/workspace/v1/workspace"
	settingstypes "github.com/dmit-4884/natscope/proto/gen/types/settings"
)

// TestValidation drives all 16 Connect handlers with malformed requests and
// asserts protovalidate/domain validation rejects them with the expected connect.Code.
func TestValidation(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()

	cases := []struct {
		name     string
		call     func() error
		wantCode connect.Code
	}{
		{
			"connections.Create missing name",
			func() error {
				_, err := env.connections.CreateConnection(ctx, connect.NewRequest(&connectionspb.CreateConnectionRequest{
					Urls: []string{"nats://127.0.0.1:1"},
				}))
				return err
			},
			connect.CodeInvalidArgument,
		},
		{
			"connections.Create missing urls",
			func() error {
				_, err := env.connections.CreateConnection(ctx, connect.NewRequest(&connectionspb.CreateConnectionRequest{
					Name: "no-urls",
				}))
				return err
			},
			connect.CodeInvalidArgument,
		},
		{
			"connections.Update non-uuid id",
			func() error {
				_, err := env.connections.UpdateConnection(ctx, connect.NewRequest(&connectionspb.UpdateConnectionRequest{
					Id: "not-a-uuid",
				}))
				return err
			},
			connect.CodeInvalidArgument,
		},
		{
			"connections.Delete empty id",
			func() error {
				_, err := env.connections.DeleteConnection(ctx, connect.NewRequest(&connectionspb.DeleteConnectionRequest{}))
				return err
			},
			connect.CodeInvalidArgument,
		},
		{
			"connections.Test missing urls",
			func() error {
				_, err := env.connections.TestConnection(ctx, connect.NewRequest(&connectionspb.TestConnectionRequest{}))
				return err
			},
			connect.CodeInvalidArgument,
		},
		{
			"mappings.Create missing pattern/message_type/source_id",
			func() error {
				_, err := env.mappings.CreateMapping(ctx, connect.NewRequest(&mappingspb.CreateMappingRequest{}))
				return err
			},
			connect.CodeInvalidArgument,
		},
		{
			"mappings.Delete non-uuid id",
			func() error {
				_, err := env.mappings.DeleteMapping(ctx, connect.NewRequest(&mappingspb.DeleteMappingRequest{Id: "nope"}))
				return err
			},
			connect.CodeInvalidArgument,
		},
		{
			"mappings.BatchSave empty list",
			func() error {
				_, err := env.mappings.BatchSaveMappings(ctx, connect.NewRequest(&mappingspb.BatchSaveMappingsRequest{}))
				return err
			},
			connect.CodeInvalidArgument,
		},
		{
			"templates.Create missing name",
			func() error {
				_, err := env.templates.CreateTemplate(ctx, connect.NewRequest(&templatespb.CreateTemplateRequest{
					Subject: "x", Data: "{}",
				}))
				return err
			},
			connect.CodeInvalidArgument,
		},
		{
			"templates.Get non-uuid id",
			func() error {
				_, err := env.templates.GetTemplate(ctx, connect.NewRequest(&templatespb.GetTemplateRequest{Id: "nope"}))
				return err
			},
			connect.CodeInvalidArgument,
		},
		{
			"templates.BatchCreate empty list",
			func() error {
				_, err := env.templates.BatchCreateTemplates(ctx, connect.NewRequest(&templatespb.BatchCreateTemplatesRequest{}))
				return err
			},
			connect.CodeInvalidArgument,
		},
		{
			"management.CreateStream missing connection_id and name",
			func() error {
				_, err := env.management.CreateStream(ctx, connect.NewRequest(&managementpb.CreateStreamRequest{}))
				return err
			},
			connect.CodeInvalidArgument,
		},
		{
			"management.CreateConsumer missing name",
			func() error {
				_, err := env.management.CreateConsumer(ctx, connect.NewRequest(&managementpb.CreateConsumerRequest{
					ConnectionId: "x", StreamName: "x",
				}))
				return err
			},
			connect.CodeInvalidArgument,
		},
		{
			"management.DeleteMessage sequence zero",
			func() error {
				_, err := env.management.DeleteMessage(ctx, connect.NewRequest(&managementpb.DeleteMessageRequest{
					ConnectionId: "x", StreamName: "x", Sequence: 0,
				}))
				return err
			},
			connect.CodeInvalidArgument,
		},
		{
			"management.CreateKVBucket missing config",
			func() error {
				_, err := env.management.CreateKVBucket(ctx, connect.NewRequest(&managementpb.CreateKVBucketRequest{
					ConnectionId: "x",
				}))
				return err
			},
			connect.CodeInvalidArgument,
		},
		{
			"management.PutKVKey missing key",
			func() error {
				_, err := env.management.PutKVKey(ctx, connect.NewRequest(&managementpb.PutKVKeyRequest{
					ConnectionId: "x", Bucket: "b",
				}))
				return err
			},
			connect.CodeInvalidArgument,
		},
		{
			"management.CreateObjectBucket missing config",
			func() error {
				_, err := env.management.CreateObjectBucket(ctx, connect.NewRequest(&managementpb.CreateObjectBucketRequest{
					ConnectionId: "x",
				}))
				return err
			},
			connect.CodeInvalidArgument,
		},
		{
			"messages.ListMessages start_seq and start_time both set (CEL oneof)",
			func() error {
				_, err := env.messages.ListMessages(ctx, connect.NewRequest(&messagespb.ListMessagesRequest{
					ConnectionId: "x", StreamName: "x",
					StartSeq:  uint64Ptr(1),
					StartTime: timestamppb.New(time.Now()),
				}))
				return err
			},
			connect.CodeInvalidArgument,
		},
		{
			"publish.PublishMessage missing subject and data",
			func() error {
				_, err := env.publish.PublishMessage(ctx, connect.NewRequest(&publishpb.PublishMessageRequest{
					ConnectionId: "x",
				}))
				return err
			},
			connect.CodeInvalidArgument,
		},
		{
			"codec.DecodeMessage empty data",
			func() error {
				_, err := env.codec.DecodeMessage(ctx, connect.NewRequest(&codecpb.DecodeMessageRequest{
					MessageType: "a.B", SourceId: "s",
				}))
				return err
			},
			connect.CodeInvalidArgument,
		},
		{
			"codec.EncodeMessage missing data",
			func() error {
				_, err := env.codec.EncodeMessage(ctx, connect.NewRequest(&codecpb.EncodeMessageRequest{
					MessageType: "a.B", SourceId: "s",
				}))
				return err
			},
			connect.CodeInvalidArgument,
		},
		{
			"registry.GetProtoMessage missing full_name/source_id",
			func() error {
				_, err := env.registry.GetProtoMessage(ctx, connect.NewRequest(&registrypb.GetProtoMessageRequest{}))
				return err
			},
			connect.CodeInvalidArgument,
		},
		{
			"registry.GenerateExample missing full_name/source_id",
			func() error {
				_, err := env.registry.GenerateExample(ctx, connect.NewRequest(&registrypb.GenerateExampleRequest{}))
				return err
			},
			connect.CodeInvalidArgument,
		},
		{
			"sources.CreateSource missing name and source_type",
			func() error {
				_, err := env.sources.CreateSource(ctx, connect.NewRequest(&sourcespb.CreateSourceRequest{}))
				return err
			},
			connect.CodeInvalidArgument,
		},
		{
			"sources.GetSource non-uuid id",
			func() error {
				_, err := env.sources.GetSource(ctx, connect.NewRequest(&sourcespb.GetSourceRequest{Id: "nope"}))
				return err
			},
			connect.CodeInvalidArgument,
		},
		{
			"sources.ValidateRepository missing repository",
			func() error {
				_, err := env.sources.ValidateRepository(ctx, connect.NewRequest(&sourcespb.ValidateRepositoryRequest{}))
				return err
			},
			connect.CodeInvalidArgument,
		},
		{
			"selections.SelectVersion missing source_id/tag",
			func() error {
				_, err := env.selections.SelectVersion(ctx, connect.NewRequest(&selectionspb.SelectVersionRequest{}))
				return err
			},
			connect.CodeInvalidArgument,
		},
		{
			"stats.GetStreamStats missing connection_id",
			func() error {
				_, err := env.stats.GetStreamStats(ctx, connect.NewRequest(&statspb.GetStreamStatsRequest{
					StreamName: "x",
				}))
				return err
			},
			connect.CodeInvalidArgument,
		},
		{
			"stats.GetHealth missing connection_id",
			func() error {
				_, err := env.stats.GetHealth(ctx, connect.NewRequest(&statspb.GetHealthRequest{}))
				return err
			},
			connect.CodeInvalidArgument,
		},
		{
			"streams.GetStream missing stream_name",
			func() error {
				_, err := env.streams.GetStream(ctx, connect.NewRequest(&streamspb.GetStreamRequest{
					ConnectionId: "x",
				}))
				return err
			},
			connect.CodeInvalidArgument,
		},
		{
			"streams.ListStreams missing connection_id",
			func() error {
				_, err := env.streams.ListStreams(ctx, connect.NewRequest(&streamspb.ListStreamsRequest{}))
				return err
			},
			connect.CodeInvalidArgument,
		},
		{
			"history.ListPublishHistory negative page_size",
			func() error {
				_, err := env.history.ListPublishHistory(ctx, connect.NewRequest(&historypb.ListPublishHistoryRequest{
					PageSize: -1,
				}))
				return err
			},
			connect.CodeInvalidArgument,
		},
		{
			"settings.UpdateSettings out-of-range max_display_rate",
			func() error {
				rate := int32(999999)
				_, err := env.settings.UpdateSettings(ctx, connect.NewRequest(&settingspb.UpdateSettingsRequest{
					Live: &settingstypes.LiveSettings{MaxDisplayRate: &rate},
				}))
				return err
			},
			connect.CodeInvalidArgument,
		},
		{
			"workspace.ValidateWorkspace missing payload",
			func() error {
				_, err := env.workspace.ValidateWorkspace(ctx, connect.NewRequest(&workspacepb.ValidateWorkspaceRequest{}))
				return err
			},
			connect.CodeInvalidArgument,
		},
		{
			"workspace.ImportWorkspace missing payload",
			func() error {
				_, err := env.workspace.ImportWorkspace(ctx, connect.NewRequest(&workspacepb.ImportWorkspaceRequest{}))
				return err
			},
			connect.CodeInvalidArgument,
		},
		{
			"live.Subscribe missing subscriptions",
			func() error {
				stream, err := env.live.Subscribe(ctx, connect.NewRequest(&livepb.SubscribeRequest{
					ConnectionId: "x",
				}))
				require.NoError(t, err, "opening the stream itself should not fail")
				stream.Receive()
				return stream.Err()
			},
			connect.CodeInvalidArgument,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call()
			require.Error(t, err, "expected an error")
			assert.Equal(t, tc.wantCode, connect.CodeOf(err), "unexpected code for %q: %v", tc.name, err)
		})
	}
}

func uint64Ptr(v uint64) *uint64 { return &v }
