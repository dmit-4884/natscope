// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/pkg/appconfig"

	"google.golang.org/protobuf/types/known/durationpb"

	mcpcmd "github.com/dmit-4884/natscope/cmd/natscope/commands/mcp"
	mcptransport "github.com/dmit-4884/natscope/internal/transports/mcp"
	mappingspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/mappings/v1/mappings"
	connectionspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/connections"
	managementpb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/management"
	publishpb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/publish"
	codecpb "github.com/dmit-4884/natscope/proto/gen/services/grpc/proto/v1/codec"
	natstypes "github.com/dmit-4884/natscope/proto/gen/types/nats"
)

func mcpSession(t *testing.T, env *e2eEnv) *mcp.ClientSession {
	t.Helper()
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "e2e", Version: "test"}, nil).
		Connect(t.Context(), &mcp.StreamableClientTransport{Endpoint: env.baseURL + "/mcp", DisableStandaloneSSE: true}, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

func callTool[T any](t *testing.T, cs *mcp.ClientSession, name string, args map[string]any) T {
	t.Helper()
	res, err := cs.CallTool(t.Context(), &mcp.CallToolParams{Name: name, Arguments: args})
	require.NoError(t, err)
	require.False(t, res.IsError, "%s failed: %s", name, toolText(res))
	raw, err := json.Marshal(res.StructuredContent)
	require.NoError(t, err)
	var out T
	require.NoError(t, json.Unmarshal(raw, &out), "%s output: %s", name, raw)
	return out
}

func callToolError(t *testing.T, cs *mcp.ClientSession, name string, args map[string]any) string {
	t.Helper()
	res, err := cs.CallTool(t.Context(), &mcp.CallToolParams{Name: name, Arguments: args})
	require.NoError(t, err)
	require.True(t, res.IsError, "%s must fail", name)
	return toolText(res)
}

func toolText(res *mcp.CallToolResult) string {
	for _, c := range res.Content {
		if text, ok := c.(*mcp.TextContent); ok {
			return text.Text
		}
	}
	return ""
}

func toolNames(t *testing.T, cs *mcp.ClientSession) []string {
	t.Helper()
	var names []string
	for tool, err := range cs.Tools(t.Context(), nil) {
		require.NoError(t, err)
		names = append(names, tool.Name)
	}
	sort.Strings(names)
	return names
}

type mcpBody struct {
	JSON   json.RawMessage `json:"json"`
	Text   string          `json:"text"`
	Base64 string          `json:"base64"`
}

type mcpMessage struct {
	Seq         uint64          `json:"seq"`
	Subject     string          `json:"subject"`
	DecodedType string          `json:"decodedType"`
	Decoded     json.RawMessage `json:"decoded"`
	Body        *mcpBody        `json:"body"`
	Truncated   bool            `json:"truncated"`
}

type mcpMessagesPage struct {
	Messages []mcpMessage `json:"messages"`
	HasMore  bool         `json:"hasMore"`
	NextSeq  uint64       `json:"nextSeq"`
}

var readOnlyTools = []string{
	"decode_payload", "describe_message_type", "detect_message_type", "find_messages", "get_kv_entry", "get_kv_history", "get_message",
	"get_schema_status", "get_server_info", "get_stream", "get_stream_relations", "list_connections", "list_consumers", "list_kv_buckets",
	"list_kv_keys", "list_mappings", "list_message_types", "list_streams", "resolve_subject", "tail_subject", "validate_payload",
}

func TestMCPReadOnly(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()
	cs := mcpSession(t, env)

	t.Run("read-only tool set", func(t *testing.T) {
		assert.Equal(t, readOnlyTools, toolNames(t, cs))
		assert.Contains(t, cs.InitializeResult().Instructions, "read-only")
		for tool, err := range cs.Tools(ctx, nil) {
			require.NoError(t, err)
			require.NotNil(t, tool.Annotations, tool.Name)
			assert.True(t, tool.Annotations.ReadOnlyHint, tool.Name)
		}
	})

	t.Run("no connection saved yet", func(t *testing.T) {
		assert.Contains(t, callToolError(t, cs, "list_streams", nil), "no saved connections")
	})

	connResp, err := env.connections.CreateConnection(ctx, connect.NewRequest(&connectionspb.CreateConnectionRequest{
		Name: "mcp-local", Urls: []string{env.natsURL},
	}))
	require.NoError(t, err)
	connID := connResp.Msg.GetConnection().GetId()

	const stream = "MCP_ORDERS"
	_, err = env.management.CreateStream(ctx, connect.NewRequest(&managementpb.CreateStreamRequest{
		ConnectionId: connID, Name: stream, Subjects: []string{"mcp.orders.>"},
	}))
	require.NoError(t, err)
	for _, p := range []struct{ subject, data string }{
		{"mcp.orders.eu", `{"n":1}`}, {"mcp.orders.us", `{"n":2}`}, {"mcp.orders.eu", "hello text"},
	} {
		pubResp, pubErr := env.publish.PublishMessage(ctx, connect.NewRequest(&publishpb.PublishMessageRequest{
			ConnectionId: connID, Subject: p.subject, Data: p.data,
		}))
		require.NoError(t, pubErr)
		require.Empty(t, pubResp.Msg.GetError())
	}

	t.Run("connections never leak credentials", func(t *testing.T) {
		out := callTool[struct {
			Connections []struct {
				ID   string   `json:"id"`
				Name string   `json:"name"`
				URLs []string `json:"urls"`
			} `json:"connections"`
		}](t, cs, "list_connections", nil)
		require.Len(t, out.Connections, 1)
		assert.Equal(t, connID, out.Connections[0].ID)
		assert.Equal(t, []string{env.natsURL}, out.Connections[0].URLs)
	})

	t.Run("server info", func(t *testing.T) {
		out := callTool[struct {
			Jetstream    bool   `json:"jetstream"`
			Version      string `json:"version"`
			Capabilities *struct {
				ApiLevel      int32 `json:"apiLevel"`
				ConsumerReset bool  `json:"consumerReset"`
				MsgCounters   bool  `json:"msgCounters"`
			} `json:"capabilities"`
		}](t, cs, "get_server_info", map[string]any{"connection": "MCP-LOCAL"})
		assert.True(t, out.Jetstream)
		assert.NotEmpty(t, out.Version)
		require.NotNil(t, out.Capabilities)
		assert.GreaterOrEqual(t, out.Capabilities.ApiLevel, int32(4))
		assert.True(t, out.Capabilities.ConsumerReset)
		assert.True(t, out.Capabilities.MsgCounters)
	})

	t.Run("streams", func(t *testing.T) {
		list := callTool[struct {
			Streams []struct {
				Config struct {
					Name      string `json:"name"`
					Retention string `json:"retention"`
					Storage   string `json:"storage"`
				} `json:"config"`
				State struct {
					Messages uint64 `json:"messages"`
					LastSeq  uint64 `json:"lastSeq"`
				} `json:"state"`
			} `json:"streams"`
		}](t, cs, "list_streams", nil)
		require.Len(t, list.Streams, 1)
		assert.Equal(t, stream, list.Streams[0].Config.Name)
		assert.Equal(t, "limits", list.Streams[0].Config.Retention)
		assert.Equal(t, "file", list.Streams[0].Config.Storage)
		assert.Equal(t, uint64(3), list.Streams[0].State.Messages)

		detail := callTool[struct {
			Config struct {
				Subjects []string `json:"subjects"`
				Discard  string   `json:"discard"`
			} `json:"config"`
		}](t, cs, "get_stream", map[string]any{"stream": stream})
		assert.Equal(t, []string{"mcp.orders.>"}, detail.Config.Subjects)
		assert.Equal(t, "old", detail.Config.Discard)

		assert.Contains(t, callToolError(t, cs, "get_stream", map[string]any{"stream": "NOPE"}), "[NATS_STREAM_NOT_FOUND]")
	})

	t.Run("find and get messages", func(t *testing.T) {
		page := callTool[mcpMessagesPage](t, cs, "find_messages", map[string]any{"stream": stream})
		require.Len(t, page.Messages, 3)
		assert.Equal(t, uint64(3), page.Messages[0].Seq, "newest first by default")
		assert.Equal(t, "hello text", page.Messages[0].Body.Text)
		assert.JSONEq(t, `{"n":1}`, string(page.Messages[2].Body.JSON))

		forward := callTool[mcpMessagesPage](t, cs, "find_messages", map[string]any{
			"stream": stream, "subject": "mcp.orders.eu", "startSeq": 1, "limit": 1,
		})
		require.Len(t, forward.Messages, 1)
		assert.Equal(t, uint64(1), forward.Messages[0].Seq)
		assert.True(t, forward.HasMore)

		filtered := callTool[mcpMessagesPage](t, cs, "find_messages", map[string]any{"stream": stream, "contains": "HELLO"})
		require.Len(t, filtered.Messages, 1)
		assert.Equal(t, uint64(3), filtered.Messages[0].Seq)

		clipped := callTool[mcpMessage](t, cs, "get_message", map[string]any{"stream": stream, "seq": 3, "maxPayloadBytes": 5})
		assert.Equal(t, "hello", clipped.Body.Text)
		assert.True(t, clipped.Truncated)

		assert.Contains(t, callToolError(t, cs, "find_messages", map[string]any{"stream": stream, "since": "soon"}), "since")
	})

	t.Run("consumers", func(t *testing.T) {
		_, err := env.management.CreateConsumer(ctx, connect.NewRequest(&managementpb.CreateConsumerRequest{
			ConnectionId: connID, StreamName: stream, Name: "mcp-worker", AckPolicy: 0, AckWait: durationpb.New(30 * time.Second),
		}))
		require.NoError(t, err)

		type consumers struct {
			Consumers []struct {
				Name       string `json:"name"`
				Stream     string `json:"stream"`
				AckPolicy  string `json:"ackPolicy"`
				NumPending uint64 `json:"numPending"`
			} `json:"consumers"`
		}
		for _, args := range []map[string]any{{"stream": stream}, nil} {
			out := callTool[consumers](t, cs, "list_consumers", args)
			require.Len(t, out.Consumers, 1)
			assert.Equal(t, "mcp-worker", out.Consumers[0].Name)
			assert.Equal(t, stream, out.Consumers[0].Stream)
			assert.Equal(t, "explicit", out.Consumers[0].AckPolicy)
			assert.Equal(t, uint64(3), out.Consumers[0].NumPending)
		}
	})

	t.Run("key value", func(t *testing.T) {
		const bucket = "mcp_kv"
		_, err := env.management.CreateKVBucket(ctx, connect.NewRequest(&managementpb.CreateKVBucketRequest{
			ConnectionId: connID, Config: &natstypes.KVBucketConfig{Bucket: bucket, History: 5},
		}))
		require.NoError(t, err)
		for _, kv := range []struct{ key, value string }{{"users.1.profile", `{"name":"ann"}`}, {"users.1.profile", "v2"}, {"flags.dark", "on"}} {
			_, err = env.management.PutKVKey(ctx, connect.NewRequest(&managementpb.PutKVKeyRequest{
				ConnectionId: connID, Bucket: bucket, Key: kv.key, Value: base64.StdEncoding.EncodeToString([]byte(kv.value)),
			}))
			require.NoError(t, err)
		}

		buckets := callTool[struct {
			Buckets []struct {
				Bucket  string `json:"bucket"`
				History int    `json:"history"`
			} `json:"buckets"`
		}](t, cs, "list_kv_buckets", nil)
		require.Len(t, buckets.Buckets, 1)
		assert.Equal(t, 5, buckets.Buckets[0].History)

		keys := callTool[struct {
			Keys  []string `json:"keys"`
			Total int      `json:"total"`
		}](t, cs, "list_kv_keys", map[string]any{"bucket": bucket, "pattern": "users.*.profile"})
		assert.Equal(t, []string{"users.1.profile"}, keys.Keys)

		type entry struct {
			Revision uint64   `json:"revision"`
			Value    *mcpBody `json:"value"`
		}
		current := callTool[entry](t, cs, "get_kv_entry", map[string]any{"bucket": bucket, "key": "users.1.profile"})
		assert.Equal(t, "v2", current.Value.Text)

		history := callTool[struct {
			Entries []entry `json:"entries"`
			Total   int     `json:"total"`
		}](t, cs, "get_kv_history", map[string]any{"bucket": bucket, "key": "users.1.profile"})
		require.Len(t, history.Entries, 2)
		assert.Equal(t, "v2", history.Entries[0].Value.Text, "newest first")
		assert.JSONEq(t, `{"name":"ann"}`, string(history.Entries[1].Value.JSON))
	})

	t.Run("tail sees messages published while it listens", func(t *testing.T) {
		done := make(chan struct{})
		go func() {
			ticker := time.NewTicker(100 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-done:
					return
				case <-ticker.C:
					_, _ = env.publish.PublishMessage(ctx, connect.NewRequest(&publishpb.PublishMessageRequest{
						ConnectionId: connID, Subject: "mcp.orders.live", Data: `{"live":true}`,
					}))
				}
			}
		}()
		out := callTool[struct {
			Messages []struct {
				Subject string   `json:"subject"`
				Body    *mcpBody `json:"body"`
			} `json:"messages"`
			StoppedBy string `json:"stoppedBy"`
		}](t, cs, "tail_subject", map[string]any{"subject": "mcp.orders.*", "seconds": 10, "maxMessages": 2})
		close(done)

		assert.Equal(t, "maxMessages", out.StoppedBy)
		require.Len(t, out.Messages, 2)
		assert.Equal(t, "mcp.orders.live", out.Messages[0].Subject)
		assert.JSONEq(t, `{"live":true}`, string(out.Messages[0].Body.JSON))
	})

	t.Run("writes are not exposed", func(t *testing.T) {
		res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "publish_message", Arguments: map[string]any{"subject": "mcp.orders.eu", "text": "x"}})
		if err == nil {
			assert.True(t, res.IsError)
		}
	})
}

func TestMCPProtoPublishAndDecode(t *testing.T) {
	env := setupE2EWith(t, func(cfg *appconfig.Config) { cfg.MCP = &appconfig.MCPConfig{AllowWrites: true} })
	ctx := t.Context()
	cs := mcpSession(t, env)

	sourceID := createLocalSource(t, env, "mcp-flow", map[string]string{"flow.proto": testProtoContent})

	connResp, err := env.connections.CreateConnection(ctx, connect.NewRequest(&connectionspb.CreateConnectionRequest{
		Name: "mcp-proto", Urls: []string{env.natsURL},
	}))
	require.NoError(t, err)
	connID := connResp.Msg.GetConnection().GetId()
	_, err = env.management.CreateStream(ctx, connect.NewRequest(&managementpb.CreateStreamRequest{
		ConnectionId: connID, Name: "MCP_FLOW", Subjects: []string{"mcpflow.>"},
	}))
	require.NoError(t, err)
	_, err = env.mappings.CreateMapping(ctx, connect.NewRequest(&mappingspb.CreateMappingRequest{
		Pattern: "mcpflow.*", MessageType: "e2eflow.FlowMessage", SourceId: sourceID,
	}))
	require.NoError(t, err)

	const fullName = "e2eflow.FlowMessage"

	t.Run("write tool is exposed", func(t *testing.T) {
		assert.Contains(t, toolNames(t, cs), "publish_message")
		assert.Contains(t, toolNames(t, cs), "request_message")
		assert.NotContains(t, cs.InitializeResult().Instructions, "read-only")
	})

	t.Run("schema tools", func(t *testing.T) {
		types := callTool[struct {
			Types []struct {
				FullName string `json:"fullName"`
				SourceID string `json:"sourceId"`
			} `json:"types"`
			Total int `json:"total"`
		}](t, cs, "list_message_types", map[string]any{"filter": "flowmessage"})
		require.Equal(t, 1, types.Total)
		assert.Equal(t, sourceID, types.Types[0].SourceID)

		described := callTool[struct {
			FullName string `json:"fullName"`
			Fields   []struct {
				Name string `json:"name"`
			} `json:"fields"`
			Example map[string]any `json:"example"`
		}](t, cs, "describe_message_type", map[string]any{"type": fullName})
		assert.Equal(t, fullName, described.FullName)
		assert.Len(t, described.Fields, 3)
		assert.NotEmpty(t, described.Example)

		assert.Contains(t, callToolError(t, cs, "describe_message_type", map[string]any{"type": "e2eflow.Flow"}), "similar: e2eflow.FlowMessage")

		type validation struct {
			Valid bool   `json:"valid"`
			Error string `json:"error"`
		}
		ok := callTool[validation](t, cs, "validate_payload", map[string]any{"type": fullName, "payload": map[string]any{"name": "x", "count": 1}})
		assert.True(t, ok.Valid)
		bad := callTool[validation](t, cs, "validate_payload", map[string]any{"type": fullName, "payload": map[string]any{"count": "many"}})
		assert.False(t, bad.Valid)
		assert.NotEmpty(t, bad.Error)

		resolved := callTool[struct {
			Mapped  bool   `json:"mapped"`
			Health  string `json:"health"`
			Mapping struct {
				MessageType string `json:"messageType"`
			} `json:"mapping"`
		}](t, cs, "resolve_subject", map[string]any{"subject": "mcpflow.created"})
		assert.True(t, resolved.Mapped)
		assert.Equal(t, fullName, resolved.Mapping.MessageType)
		assert.Equal(t, "ok", resolved.Health)

		unmapped := callTool[struct {
			Mapped bool `json:"mapped"`
		}](t, cs, "resolve_subject", map[string]any{"subject": "other.subject"})
		assert.False(t, unmapped.Mapped)

		status := callTool[struct {
			MessageTypes int `json:"messageTypes"`
		}](t, cs, "get_schema_status", nil)
		assert.Positive(t, status.MessageTypes)

		mappings := callTool[struct {
			Mappings []struct {
				Pattern string `json:"pattern"`
			} `json:"mappings"`
		}](t, cs, "list_mappings", nil)
		require.Len(t, mappings.Mappings, 1)
		assert.Equal(t, "mcpflow.*", mappings.Mappings[0].Pattern)
	})

	t.Run("publish through the mapping, read back decoded", func(t *testing.T) {
		published := callTool[struct {
			Stream      string `json:"stream"`
			Seq         uint64 `json:"seq"`
			Encoding    string `json:"encoding"`
			MessageType string `json:"messageType"`
		}](t, cs, "publish_message", map[string]any{
			"subject": "mcpflow.created", "json": map[string]any{"name": "hello", "count": 42, "tags": []string{"a"}},
		})
		assert.Equal(t, "MCP_FLOW", published.Stream)
		assert.Equal(t, uint64(1), published.Seq)
		assert.Equal(t, "protobuf", published.Encoding)
		assert.Equal(t, fullName, published.MessageType)

		raw := callTool[struct {
			Encoding string `json:"encoding"`
		}](t, cs, "publish_message", map[string]any{"subject": "mcpflow.note", "json": map[string]any{"note": 1}, "raw": true})
		assert.Equal(t, "raw", raw.Encoding)

		page := callTool[mcpMessagesPage](t, cs, "find_messages", map[string]any{"stream": "MCP_FLOW", "direction": "forward"})
		require.Len(t, page.Messages, 2)
		assert.Equal(t, fullName, page.Messages[0].DecodedType)
		assert.JSONEq(t, `{"name":"hello","count":42,"tags":["a"]}`, string(page.Messages[0].Decoded))
		assert.Nil(t, page.Messages[0].Body)

		bad := callToolError(t, cs, "publish_message", map[string]any{"subject": "mcpflow.created", "json": map[string]any{"count": "many"}})
		assert.NotEmpty(t, bad)
	})

	t.Run("request a service", func(t *testing.T) {
		nc, err := nats.Connect(env.natsURL)
		require.NoError(t, err)
		defer nc.Close()
		_, err = nc.Subscribe("mcpsvc.echo", func(m *nats.Msg) {
			_ = m.Respond(fmt.Appendf(nil, `{"echo":%q}`, m.Data))
		})
		require.NoError(t, err)
		require.NoError(t, nc.Flush())

		out := callTool[struct {
			Subject  string             `json:"subject"`
			Size     int                `json:"size"`
			Body     *mcptransport.Body `json:"body"`
			Encoding string             `json:"encoding"`
		}](t, cs, "request_message", map[string]any{"subject": "mcpsvc.echo", "text": "ping", "timeoutMs": 2000})
		assert.True(t, strings.HasPrefix(out.Subject, nats.InboxPrefix), out.Subject)
		assert.Equal(t, "raw", out.Encoding)
		require.NotNil(t, out.Body)
		assert.JSONEq(t, `{"echo":"ping"}`, string(out.Body.JSON))
		assert.Equal(t, len(`{"echo":"ping"}`), out.Size)

		assert.Contains(t, callToolError(t, cs, "request_message", map[string]any{"subject": "mcpsvc.nobody"}), "NATS_NO_RESPONDERS")
	})

	t.Run("decode payload by type and by subject", func(t *testing.T) {
		enc, err := env.codec.EncodeMessage(ctx, connect.NewRequest(&codecpb.EncodeMessageRequest{
			MessageType: fullName, SourceId: sourceID, Data: `{"name":"wire","count":7}`,
		}))
		require.NoError(t, err)
		wire := base64.StdEncoding.EncodeToString(enc.Msg.GetResult().GetData())

		type decoded struct {
			MessageType string          `json:"messageType"`
			Decoded     json.RawMessage `json:"decoded"`
		}
		for _, args := range []map[string]any{{"base64": wire, "type": fullName}, {"base64": wire, "subject": "mcpflow.any"}} {
			out := callTool[decoded](t, cs, "decode_payload", args)
			assert.Equal(t, fullName, out.MessageType)
			assert.JSONEq(t, `{"name":"wire","count":7,"tags":[]}`, string(out.Decoded))
		}
		dump := callTool[struct {
			Wire []struct {
				Number   int32  `json:"number"`
				WireType string `json:"wireType"`
				Text     string `json:"text"`
				Varint   uint64 `json:"varint"`
			} `json:"wire"`
		}](t, cs, "decode_payload", map[string]any{"base64": wire})
		require.Len(t, dump.Wire, 2)
		assert.Equal(t, "wire", dump.Wire[0].Text)
		assert.Equal(t, "bytes", dump.Wire[0].WireType)
		assert.Equal(t, uint64(7), dump.Wire[1].Varint)

		guess := callTool[struct {
			Candidates []struct {
				SourceID    string          `json:"sourceId"`
				MessageType string          `json:"messageType"`
				Score       int             `json:"score"`
				Decoded     json.RawMessage `json:"decoded"`
			} `json:"candidates"`
		}](t, cs, "detect_message_type", map[string]any{"base64": wire})
		require.Len(t, guess.Candidates, 1)
		assert.Equal(t, fullName, guess.Candidates[0].MessageType)
		assert.Equal(t, sourceID, guess.Candidates[0].SourceID)
		assert.GreaterOrEqual(t, guess.Candidates[0].Score, 85)
		assert.JSONEq(t, `{"name":"wire","count":7,"tags":[]}`, string(guess.Candidates[0].Decoded))
		assert.Contains(t, callToolError(t, cs, "detect_message_type", map[string]any{"base64": ""}), "empty payload")
	})
}

func TestMCPStdioBridge(t *testing.T) {
	env := setupE2E(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	downstream, client := mcp.NewInMemoryTransports()
	bridgeDone := make(chan error, 1)
	go func() { bridgeDone <- mcpcmd.Bridge(ctx, env.baseURL+"/mcp", http.DefaultClient, downstream) }()

	cs, err := mcp.NewClient(&mcp.Implementation{Name: "stdio-client", Version: "test"}, nil).Connect(ctx, client, nil)
	require.NoError(t, err)

	assert.Equal(t, readOnlyTools, toolNames(t, cs))
	assert.Contains(t, cs.InitializeResult().Instructions, "saved connections")
	assert.Contains(t, callToolError(t, cs, "list_streams", nil), "no saved connections")
	out := callTool[struct {
		Connections []any `json:"connections"`
	}](t, cs, "list_connections", nil)
	assert.Empty(t, out.Connections)

	require.NoError(t, cs.Close())
	select {
	case <-bridgeDone:
	case <-time.After(10 * time.Second):
		t.Fatal("bridge did not stop after the client disconnected")
	}
}

func TestMCPStdioBridgeUnreachable(t *testing.T) {
	downstream, _ := mcp.NewInMemoryTransports()
	err := mcpcmd.Bridge(t.Context(), "http://"+freeLocalAddr(t)+"/mcp", http.DefaultClient, downstream)
	require.ErrorContains(t, err, "natscope is not reachable")
}

func TestMCPDisabled(t *testing.T) {
	env := setupE2EWith(t, func(cfg *appconfig.Config) { cfg.MCP = &appconfig.MCPConfig{Enabled: new(false)} })
	resp, err := http.Post(env.baseURL+"/mcp", "application/json", nil) //nolint:noctx
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode, "a disabled /mcp falls through to the SPA")
}
