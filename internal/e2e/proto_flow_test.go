// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"google.golang.org/protobuf/encoding/protowire"

	mappingspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/mappings/v1/mappings"
	connectionspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/connections"
	managementpb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/management"
	messagespb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/messages"
	publishpb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/publish"
	codecpb "github.com/dmit-4884/natscope/proto/gen/services/grpc/proto/v1/codec"
	registrypb "github.com/dmit-4884/natscope/proto/gen/services/grpc/proto/v1/registry"
	sourcespb "github.com/dmit-4884/natscope/proto/gen/services/grpc/proto/v1/sources"
	settingspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/settings/v1/settings"
	natspb "github.com/dmit-4884/natscope/proto/gen/types/nats"
	protopb "github.com/dmit-4884/natscope/proto/gen/types/proto"
	settingstypes "github.com/dmit-4884/natscope/proto/gen/types/settings"
)

const testProtoContent = `syntax = "proto3";

package e2eflow;

// A message flowing through the e2e pipeline.
message FlowMessage {
  string name = 1; // Display name.
  int32 count = 2;
  repeated string tags = 3;
}
`

func createLocalSource(t *testing.T, env *e2eEnv, name string, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for p, content := range files {
		full := filepath.Join(dir, filepath.FromSlash(p))
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(content), 0o600))
	}
	createResp, err := env.sources.CreateSource(t.Context(), connect.NewRequest(&sourcespb.CreateSourceRequest{
		Name: name, SourceType: protopb.SourceType_SOURCE_TYPE_LOCAL, LocalPath: &dir, WatcherEnabled: new(false),
	}))
	require.NoError(t, err)
	sourceID := createResp.Msg.GetSource().GetId()
	refreshResp, err := env.sources.RefreshSource(t.Context(), connect.NewRequest(&sourcespb.RefreshSourceRequest{SourceId: sourceID}))
	require.NoError(t, err)
	outcome := refreshResp.Msg.GetOutcome()
	require.True(t, outcome.GetValid(), "compile diagnostics: %v", outcome.GetDiagnostics())
	require.NotEmpty(t, refreshResp.Msg.GetSource().GetActiveSchema().GetFingerprint())
	return sourceID
}

// TestProtoFlow drives the full proto pipeline end-to-end: create, compile,
// mapping, and server-side decode — the path TestE2E's registry_codec_sources
// subtest explicitly skips for lack of a real proto source.
func TestProtoFlow(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()

	sourceID := createLocalSource(t, env, "flow-source", map[string]string{"flow.proto": testProtoContent})

	const fullName = "e2eflow.FlowMessage"

	t.Run("registry sees the compiled source", func(t *testing.T) {
		statusResp, err := env.registry.GetProtoStatus(ctx, connect.NewRequest(&registrypb.GetProtoStatusRequest{}))
		require.NoError(t, err)
		assert.True(t, statusResp.Msg.GetLoaded())
		assert.Greater(t, statusResp.Msg.GetMessageCount(), int32(0))

		typesResp, err := env.registry.ListTypes(ctx, connect.NewRequest(&registrypb.ListTypesRequest{SourceId: &sourceID}))
		require.NoError(t, err)
		require.Len(t, typesResp.Msg.GetTypes(), 1)
		assert.Equal(t, fullName, typesResp.Msg.GetTypes()[0].GetFullName())
		assert.Equal(t, "A message flowing through the e2e pipeline.", typesResp.Msg.GetTypes()[0].GetComment())
		assert.False(t, typesResp.Msg.GetTypes()[0].GetDependency())

		descResp, err := env.registry.DescribeType(ctx, connect.NewRequest(&registrypb.DescribeTypeRequest{
			FullName: fullName, SourceId: sourceID,
		}))
		require.NoError(t, err)
		require.Len(t, descResp.Msg.GetMessages(), 1)
		fields := descResp.Msg.GetMessages()[0].GetFields()
		require.Len(t, fields, 3)
		assert.Equal(t, "Display name.", fields[0].GetComment())
		assert.True(t, fields[2].GetRepeated())

		exResp, err := env.registry.GenerateExample(ctx, connect.NewRequest(&registrypb.GenerateExampleRequest{
			FullName: fullName, SourceId: sourceID,
		}))
		require.NoError(t, err)
		assert.NotEmpty(t, exResp.Msg.GetJson())
	})

	var wireBytes []byte

	t.Run("codec encode/validate/decode round-trip", func(t *testing.T) {
		encResp, err := env.codec.EncodeMessage(ctx, connect.NewRequest(&codecpb.EncodeMessageRequest{
			MessageType: fullName, SourceId: sourceID,
			Data: `{"name":"hello","count":42,"tags":["a","b"]}`,
		}))
		require.NoError(t, err)
		wireBytes = encResp.Msg.GetResult().GetData()
		require.NotEmpty(t, wireBytes)

		valResp, err := env.codec.ValidateMessage(ctx, connect.NewRequest(&codecpb.ValidateMessageRequest{
			Data: wireBytes, MessageType: fullName, SourceId: sourceID,
		}))
		require.NoError(t, err)
		assert.True(t, valResp.Msg.GetResult().GetValid())

		decResp, err := env.codec.DecodeMessage(ctx, connect.NewRequest(&codecpb.DecodeMessageRequest{
			Data: wireBytes, MessageType: fullName, SourceId: sourceID,
		}))
		require.NoError(t, err)
		assert.Contains(t, decResp.Msg.GetResult().GetData(), "hello")
	})

	// Publish a raw (proto-encoded) message on a subject mapped to FlowMessage
	// and verify it comes back decoded through ListMessages/GetMessage.
	connResp, err := env.connections.CreateConnection(ctx, connect.NewRequest(&connectionspb.CreateConnectionRequest{
		Name: "proto-flow-conn", Urls: []string{env.natsURL},
	}))
	require.NoError(t, err)
	connID := connResp.Msg.GetConnection().GetId()

	const stream = "PROTO_FLOW"
	const subject = "flow.msg"
	_, err = env.management.CreateStream(ctx, connect.NewRequest(&managementpb.CreateStreamRequest{
		ConnectionId: connID, Name: stream, Subjects: []string{subject},
	}))
	require.NoError(t, err)

	mappingResp, err := env.mappings.CreateMapping(ctx, connect.NewRequest(&mappingspb.CreateMappingRequest{
		Pattern: subject, MessageType: fullName, SourceId: sourceID,
	}))
	require.NoError(t, err)

	t.Run("mapping health follows the pinned version", func(t *testing.T) {
		pinnedResp, err := env.mappings.CreateMapping(ctx, connect.NewRequest(&mappingspb.CreateMappingRequest{
			Pattern: "flow.pinned", MessageType: fullName, SourceId: sourceID, PinnedFingerprint: new("no-such-fingerprint"),
		}))
		require.NoError(t, err)

		healthResp, err := env.mappings.BatchCheckMappingHealth(ctx, connect.NewRequest(&mappingspb.BatchCheckMappingHealthRequest{
			Ids: []string{mappingResp.Msg.GetMapping().GetId(), pinnedResp.Msg.GetMapping().GetId(), unknownUUID},
		}))
		require.NoError(t, err)
		items := healthResp.Msg.GetItems()
		require.Len(t, items, 3)
		assert.Equal(t, "ok", items[0].GetHealth())
		assert.Equal(t, "descriptor_missing", items[1].GetHealth())
		assert.Equal(t, "mapping_missing", items[2].GetHealth())

		_, err = env.mappings.DeleteMapping(ctx, connect.NewRequest(&mappingspb.DeleteMappingRequest{Id: pinnedResp.Msg.GetMapping().GetId()}))
		require.NoError(t, err)
	})

	t.Run("publish + server-side proto decode", func(t *testing.T) {
		msgType := fullName
		pubResp, err := env.publish.PublishMessage(ctx, connect.NewRequest(&publishpb.PublishMessageRequest{
			ConnectionId: connID, Subject: subject, MessageType: &msgType, SourceId: &sourceID,
			Data: `{"name":"decoded","count":7,"tags":["x"]}`,
		}))
		require.NoError(t, err)
		require.Empty(t, pubResp.Msg.GetError())
		seq := pubResp.Msg.GetSequence()
		require.Greater(t, seq, uint64(0))

		var got *messagespb.GetMessageResponse
		require.Eventually(t, func() bool {
			resp, err := env.messages.GetMessage(ctx, connect.NewRequest(&messagespb.GetMessageRequest{
				ConnectionId: connID, StreamName: stream, Sequence: seq,
			}))
			if err != nil {
				return false
			}
			got = resp.Msg
			return got.GetMessage() != nil
		}, 5*time.Second, 100*time.Millisecond)

		require.NotNil(t, got)
		decoded := got.GetMessage().GetDecoded()
		require.NotEmpty(t, decoded, "message on a mapped subject must be server-side proto-decoded")
		assert.Contains(t, decoded, "decoded")
		assert.Equal(t, fullName, got.GetMessage().GetDecodedType())
	})

	t.Run("unknown fields, partial decode and the wire dump", func(t *testing.T) {
		var known []byte
		known = protowire.AppendTag(known, 1, protowire.BytesType)
		known = protowire.AppendString(known, "newer")
		withExtra := protowire.AppendTag(append([]byte{}, known...), 15, protowire.VarintType)
		withExtra = protowire.AppendVarint(withExtra, 3)
		broken := append(append([]byte{}, known...), 0x1a, 0x09, 'x')

		nc, err := nats.Connect(env.natsURL)
		require.NoError(t, err)
		defer nc.Close()
		require.NoError(t, nc.Publish(subject, withExtra))
		require.NoError(t, nc.Publish(subject, broken))
		require.NoError(t, nc.Flush())

		var msgs []*natspb.NatsMessage
		require.Eventually(t, func() bool {
			resp, err := env.messages.ListMessages(ctx, connect.NewRequest(&messagespb.ListMessagesRequest{
				ConnectionId: connID, StreamName: stream, Limit: new(int64(10)),
			}))
			if err != nil {
				return false
			}
			msgs = resp.Msg.GetMessages()
			return len(msgs) >= 3
		}, 5*time.Second, 100*time.Millisecond)

		bySize := map[int32]*natspb.NatsMessage{}
		for _, m := range msgs {
			bySize[m.GetDataSize()] = m
		}
		extra := bySize[int32(len(withExtra))]
		require.NotNil(t, extra)
		assert.Empty(t, extra.GetDecodeError())
		assert.Equal(t, int32(1), extra.GetDecodedUnknownFields())
		assert.Contains(t, extra.GetDecoded(), "newer")

		partial := bySize[int32(len(broken))]
		require.NotNil(t, partial)
		assert.NotEmpty(t, partial.GetDecodeError())
		assert.Equal(t, int32(len(known)), partial.GetDecodedValidBytes())
		assert.Contains(t, partial.GetDecoded(), "newer")

		wire, err := env.codec.DecodeWire(ctx, connect.NewRequest(&codecpb.DecodeWireRequest{Data: withExtra}))
		require.NoError(t, err)
		require.Len(t, wire.Msg.GetFields(), 2)
		assert.Equal(t, "newer", wire.Msg.GetFields()[0].GetText())
		assert.Equal(t, int32(15), wire.Msg.GetFields()[1].GetNumber())
		assert.Equal(t, protopb.WireType_WIRE_TYPE_VARINT, wire.Msg.GetFields()[1].GetWireType())

		decResp, err := env.codec.DecodeMessage(ctx, connect.NewRequest(&codecpb.DecodeMessageRequest{
			Data: withExtra, MessageType: fullName, SourceId: sourceID,
		}))
		require.NoError(t, err)
		require.Len(t, decResp.Msg.GetResult().GetUnknownFields(), 1)
		assert.Equal(t, int32(15), decResp.Msg.GetResult().GetUnknownFields()[0].GetNumber())
	})

	t.Run("framing on the mapping", func(t *testing.T) {
		grpcFraming := &protopb.Framing{Kind: protopb.FramingKind_FRAMING_KIND_GRPC}
		upd, err := env.mappings.UpdateMapping(ctx, connect.NewRequest(&mappingspb.UpdateMappingRequest{
			Id: mappingResp.Msg.GetMapping().GetId(), Framing: grpcFraming,
		}))
		require.NoError(t, err)
		assert.Equal(t, protopb.FramingKind_FRAMING_KIND_GRPC, upd.Msg.GetMapping().GetFraming().GetKind())
		assert.Equal(t, fullName, upd.Msg.GetMapping().GetMessageType(), "unset fields stay")

		msgType := fullName
		pubResp, err := env.publish.PublishMessage(ctx, connect.NewRequest(&publishpb.PublishMessageRequest{
			ConnectionId: connID, Subject: subject, MessageType: &msgType, SourceId: &sourceID,
			Data: `{"name":"framed"}`, Framing: grpcFraming,
		}))
		require.NoError(t, err)
		require.Empty(t, pubResp.Msg.GetError())

		var got *natspb.NatsMessage
		require.Eventually(t, func() bool {
			resp, err := env.messages.GetMessage(ctx, connect.NewRequest(&messagespb.GetMessageRequest{
				ConnectionId: connID, StreamName: stream, Sequence: pubResp.Msg.GetSequence(),
			}))
			if err != nil {
				return false
			}
			got = resp.Msg.GetMessage()
			return got != nil
		}, 5*time.Second, 100*time.Millisecond)
		assert.Empty(t, got.GetDecodeError())
		assert.Contains(t, got.GetDecoded(), "framed")
		raw, err := base64.StdEncoding.DecodeString(got.GetDataBase64())
		require.NoError(t, err)
		assert.Equal(t, []byte{0, 0, 0, 0, byte(len(raw) - 5)}, raw[:5], "published inside a gRPC frame")

		dec, err := env.codec.DecodeMessage(ctx, connect.NewRequest(&codecpb.DecodeMessageRequest{
			Data: raw, MessageType: fullName, SourceId: sourceID,
		}))
		require.NoError(t, err)
		assert.Contains(t, dec.Msg.GetResult().GetError(), "set the mapping's framing to gRPC")

		_, err = env.mappings.UpdateMapping(ctx, connect.NewRequest(&mappingspb.UpdateMappingRequest{
			Id: mappingResp.Msg.GetMapping().GetId(), Framing: &protopb.Framing{},
		}))
		require.NoError(t, err)
	})

	t.Run("type detection on an unmapped subject", func(t *testing.T) {
		const autoStream = "PROTO_AUTO"
		_, err := env.management.CreateStream(ctx, connect.NewRequest(&managementpb.CreateStreamRequest{
			ConnectionId: connID, Name: autoStream, Subjects: []string{"auto.>"},
		}))
		require.NoError(t, err)

		enc, err := env.codec.EncodeMessage(ctx, connect.NewRequest(&codecpb.EncodeMessageRequest{
			MessageType: fullName, SourceId: sourceID, Data: `{"name":"guess","count":3}`,
		}))
		require.NoError(t, err)
		payload := enc.Msg.GetResult().GetData()

		detect, err := env.codec.DetectMessageType(ctx, connect.NewRequest(&codecpb.DetectMessageTypeRequest{Data: payload}))
		require.NoError(t, err)
		require.NotEmpty(t, detect.Msg.GetCandidates())
		best := detect.Msg.GetCandidates()[0]
		assert.Equal(t, fullName, best.GetMessageType())
		assert.Equal(t, sourceID, best.GetSourceId())
		assert.GreaterOrEqual(t, best.GetScore(), int32(85))
		assert.JSONEq(t, `{"name":"guess","count":3,"tags":[]}`, best.GetDecoded())

		nc, err := nats.Connect(env.natsURL)
		require.NoError(t, err)
		defer nc.Close()
		js, err := nc.JetStream()
		require.NoError(t, err)
		ack, err := js.Publish("auto.1001.flow", payload)
		require.NoError(t, err)

		get := func() *natspb.NatsMessage {
			resp, err := env.messages.GetMessage(ctx, connect.NewRequest(&messagespb.GetMessageRequest{
				ConnectionId: connID, StreamName: autoStream, Sequence: ack.Sequence,
			}))
			require.NoError(t, err)
			return resp.Msg.GetMessage()
		}
		got := get()
		assert.True(t, got.GetDecodedAuto())
		assert.Equal(t, fullName, got.GetDecodedType())
		assert.Equal(t, sourceID, got.GetDecodedSourceId())
		assert.Contains(t, got.GetDecoded(), "guess")

		_, err = env.settings.UpdateSettings(ctx, connect.NewRequest(&settingspb.UpdateSettingsRequest{
			Messages: &settingstypes.MessageSettings{DetectTypes: new(false)},
		}))
		require.NoError(t, err)
		off := get()
		assert.False(t, off.GetDecodedAuto())
		assert.Empty(t, off.GetDecoded())

		_, err = env.settings.ResetSettings(ctx, connect.NewRequest(&settingspb.ResetSettingsRequest{}))
		require.NoError(t, err)
	})

	t.Run("KV values encode and decode through mappings", func(t *testing.T) {
		for _, bucket := range []string{"PROTO_CFG", "PROTO_RAW"} {
			_, err := env.management.CreateKVBucket(ctx, connect.NewRequest(&managementpb.CreateKVBucketRequest{
				ConnectionId: connID, Config: &natspb.KVBucketConfig{Bucket: bucket, History: 5},
			}))
			require.NoError(t, err)
		}
		_, err := env.mappings.CreateMapping(ctx, connect.NewRequest(&mappingspb.CreateMappingRequest{
			Pattern: "$KV.PROTO_CFG.>", MessageType: fullName, SourceId: sourceID,
		}))
		require.NoError(t, err)

		put := func(bucket, key, json string) error {
			_, err := env.management.PutKVKey(ctx, connect.NewRequest(&managementpb.PutKVKeyRequest{
				ConnectionId: connID, Bucket: bucket, Key: key,
				Payload: &managementpb.PutKVKeyRequest_Proto{Proto: &managementpb.KVProtoValue{
					MessageType: fullName, SourceId: sourceID, Json: json,
				}},
			}))
			return err
		}
		require.NoError(t, put("PROTO_CFG", "app.limits", `{"name":"kv","count":5}`))
		require.NoError(t, put("PROTO_CFG", "app.limits", `{"name":"kv","count":6}`))

		got, err := env.management.GetKVKey(ctx, connect.NewRequest(&managementpb.GetKVKeyRequest{
			ConnectionId: connID, Bucket: "PROTO_CFG", Key: "app.limits",
		}))
		require.NoError(t, err)
		decoded := got.Msg.GetEntry().GetDecoded()
		require.NotNil(t, decoded)
		assert.JSONEq(t, `{"name":"kv","count":6,"tags":[]}`, decoded.GetData())
		assert.Equal(t, fullName, decoded.GetMessageType())
		assert.Equal(t, sourceID, decoded.GetSourceId())
		assert.False(t, decoded.GetAuto())
		raw, err := base64.StdEncoding.DecodeString(got.Msg.GetEntry().GetValue())
		require.NoError(t, err)
		assert.Equal(t, byte(0x0a), raw[0], "stored as Protobuf, not JSON")

		history, err := env.management.GetKVKeyHistory(ctx, connect.NewRequest(&managementpb.GetKVKeyHistoryRequest{
			ConnectionId: connID, Bucket: "PROTO_CFG", Key: "app.limits",
		}))
		require.NoError(t, err)
		require.Len(t, history.Msg.GetEntries(), 2)
		assert.Contains(t, history.Msg.GetEntries()[0].GetDecoded().GetData(), `"count":5`)

		err = put("PROTO_CFG", "app.limits", `{"count":"many"}`)
		require.Error(t, err)
		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
		assert.Equal(t, "PROTO_ENCODE_FAILED", errorReason(t, err))

		require.NoError(t, put("PROTO_RAW", "guess", `{"name":"raw","count":2,"tags":["a"]}`))
		auto, err := env.management.GetKVKey(ctx, connect.NewRequest(&managementpb.GetKVKeyRequest{
			ConnectionId: connID, Bucket: "PROTO_RAW", Key: "guess",
		}))
		require.NoError(t, err)
		assert.True(t, auto.Msg.GetEntry().GetDecoded().GetAuto())
		assert.Equal(t, fullName, auto.Msg.GetEntry().GetDecoded().GetMessageType())

		_, err = env.management.PutKVKey(ctx, connect.NewRequest(&managementpb.PutKVKeyRequest{
			ConnectionId: connID, Bucket: "PROTO_RAW", Key: "text",
			Payload: &managementpb.PutKVKeyRequest_Value{Value: base64.StdEncoding.EncodeToString([]byte("plain"))},
		}))
		require.NoError(t, err)
		text, err := env.management.GetKVKey(ctx, connect.NewRequest(&managementpb.GetKVKeyRequest{
			ConnectionId: connID, Bucket: "PROTO_RAW", Key: "text",
		}))
		require.NoError(t, err)
		assert.Nil(t, text.Msg.GetEntry().GetDecoded())
	})
}
