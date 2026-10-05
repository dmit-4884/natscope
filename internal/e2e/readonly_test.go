// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/pkg/appconfig"

	connectionspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/connections"
	managementpb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/management"
	messagespb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/messages"
	publishpb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/publish"
	natstypes "github.com/dmit-4884/natscope/proto/gen/types/nats"
)

// TestReadOnlyConnection checks that a read-only connection serves reads and refuses every write, over the API and MCP.
func TestReadOnlyConnection(t *testing.T) {
	env := setupE2EWith(t, func(cfg *appconfig.Config) { cfg.MCP = &appconfig.MCPConfig{AllowWrites: true} })
	ctx := t.Context()

	nc, err := nats.Connect(env.natsURL)
	require.NoError(t, err)
	defer nc.Close()
	js, err := jetstream.New(nc)
	require.NoError(t, err)
	_, err = js.CreateStream(ctx, jetstream.StreamConfig{Name: "RO", Subjects: []string{"ro.>"}})
	require.NoError(t, err)
	_, err = js.Publish(ctx, "ro.a", []byte("kept"))
	require.NoError(t, err)
	_, err = js.CreateOrUpdateConsumer(ctx, "RO", jetstream.ConsumerConfig{Durable: "C1"})
	require.NoError(t, err)

	created, err := env.connections.CreateConnection(ctx, connect.NewRequest(&connectionspb.CreateConnectionRequest{
		Name:     "prod",
		Urls:     []string{env.natsURL},
		ReadOnly: true,
		Label:    &natstypes.ConnectionLabel{Text: " PROD ", Color: natstypes.LabelColor_LABEL_COLOR_RED},
	}))
	require.NoError(t, err)
	conn := created.Msg.GetConnection()
	connID := conn.GetId()
	assert.True(t, conn.GetReadOnly())
	assert.Equal(t, "PROD", conn.GetLabel().GetText())
	assert.Equal(t, natstypes.LabelColor_LABEL_COLOR_RED, conn.GetLabel().GetColor())

	refused := func(t *testing.T, err error) {
		t.Helper()
		require.Error(t, err)
		assert.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err), "%v", err)
		assert.Equal(t, "CONNECTION_READ_ONLY", errorReason(t, err))
	}

	t.Run("writes are refused", func(t *testing.T) {
		_, err := env.publish.PublishMessage(ctx, connect.NewRequest(&publishpb.PublishMessageRequest{
			ConnectionId: connID, Subject: "ro.b", Data: "x",
		}))
		refused(t, err)
		_, err = env.publish.RequestMessage(ctx, connect.NewRequest(&publishpb.RequestMessageRequest{
			ConnectionId: connID, Subject: "svc.echo", Data: "x",
		}))
		refused(t, err)
		_, err = env.management.CreateStream(ctx, connect.NewRequest(&managementpb.CreateStreamRequest{
			ConnectionId: connID, Name: "RO2", Subjects: []string{"ro2.>"},
		}))
		refused(t, err)
		_, err = env.management.DeleteConsumer(ctx, connect.NewRequest(&managementpb.DeleteConsumerRequest{
			ConnectionId: connID, StreamName: "RO", ConsumerName: "C1",
		}))
		refused(t, err)
		_, err = env.management.DeleteMessage(ctx, connect.NewRequest(&managementpb.DeleteMessageRequest{
			ConnectionId: connID, StreamName: "RO", Sequence: 1,
		}))
		refused(t, err)
		_, err = env.management.CreateKVBucket(ctx, connect.NewRequest(&managementpb.CreateKVBucketRequest{
			ConnectionId: connID, Config: &natstypes.KVBucketConfig{Bucket: "ro_kv"},
		}))
		refused(t, err)
	})

	t.Run("reads work", func(t *testing.T) {
		msg, err := env.messages.GetMessage(ctx, connect.NewRequest(&messagespb.GetMessageRequest{
			ConnectionId: connID, StreamName: "RO", Sequence: 1,
		}))
		require.NoError(t, err)
		assert.Equal(t, "ro.a", msg.Msg.GetMessage().GetSubject())
		consumers, err := env.management.ListConsumers(ctx, connect.NewRequest(&managementpb.ListConsumersRequest{
			ConnectionId: connID, StreamName: "RO",
		}))
		require.NoError(t, err)
		require.Len(t, consumers.Msg.GetConsumers(), 1)
	})

	t.Run("MCP sees the label and cannot publish", func(t *testing.T) {
		cs := mcpSession(t, env)
		list := callTool[struct {
			Connections []struct {
				ReadOnly bool `json:"readOnly"`
				Label    *struct {
					Text  string `json:"text"`
					Color string `json:"color"`
				} `json:"label"`
			} `json:"connections"`
		}](t, cs, "list_connections", map[string]any{})
		require.Len(t, list.Connections, 1)
		assert.True(t, list.Connections[0].ReadOnly)
		require.NotNil(t, list.Connections[0].Label)
		assert.Equal(t, "PROD", list.Connections[0].Label.Text)
		assert.Equal(t, "red", list.Connections[0].Label.Color)

		text := callToolError(t, cs, "publish_message", map[string]any{"subject": "ro.mcp", "text": "x"})
		assert.Contains(t, text, "read-only")
	})

	t.Run("switching read-only off allows writes and an empty label clears it", func(t *testing.T) {
		updated, err := env.connections.UpdateConnection(ctx, connect.NewRequest(&connectionspb.UpdateConnectionRequest{
			Id: connID, ReadOnly: new(false), Label: &natstypes.ConnectionLabel{},
		}))
		require.NoError(t, err)
		assert.False(t, updated.Msg.GetConnection().GetReadOnly())
		assert.Nil(t, updated.Msg.GetConnection().GetLabel())

		_, err = env.publish.PublishMessage(ctx, connect.NewRequest(&publishpb.PublishMessageRequest{
			ConnectionId: connID, Subject: "ro.b", Data: "x",
		}))
		require.NoError(t, err)
	})
}
