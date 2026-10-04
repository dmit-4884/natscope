// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package live

import (
	"encoding/base64"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"

	natspb "github.com/dmit-4884/natscope/proto/gen/types/nats"
)

func TestToProtoLiveMessage_AllFields(t *testing.T) {
	t.Parallel()

	seq := uint64(42)
	ts := time.UnixMilli(1700000000000).UTC()
	msg := &entities.NatsMessage{
		Subject:   "orders.created",
		Data:      []byte(`{"id":1}`),
		Header:    map[string][]string{"X-Trace": {"abc-123"}, "X-Source": {"api", "secondary"}},
		Sequence:  &seq,
		Stream:    "ORDERS",
		Timestamp: ts,
	}

	pb := toProtoLiveMessage(msg)

	assert.Equal(t, "orders.created", pb.Subject)
	require.NotNil(t, pb.Timestamp)
	assert.Equal(t, ts.UnixMilli(), pb.Timestamp.AsTime().UnixMilli(), "Timestamp passes through unchanged when non-zero")
	assert.Equal(t, base64.StdEncoding.EncodeToString(msg.Data), pb.DataBase64, "BytesBase64 codec")
	assert.Equal(t, int32(len(msg.Data)), pb.DataSize)
	assert.Equal(t, string(entities.DetectContentType(msg.Data)), pb.ContentType)
	assert.Equal(t, uint64(42), pb.Sequence, "primitive deref *uint64 → uint64")
	require.NotNil(t, pb.Stream)
	assert.Equal(t, "ORDERS", *pb.Stream, "primitive ptr-init string → *string")

	require.Len(t, pb.Headers, 2)
	assert.Equal(t, "abc-123", pb.Headers["X-Trace"])
	assert.Equal(t, "api, secondary", pb.Headers["X-Source"], "every value of a repeated header is kept")
}

func TestToProtoLiveMessage_TimestampFallback(t *testing.T) {
	t.Parallel()

	before := time.Now().UnixMilli()
	msg := &entities.NatsMessage{
		Subject:   "x",
		Data:      []byte("y"),
		Timestamp: time.Time{}, // unset → fallback to wall clock
	}

	pb := toProtoLiveMessage(msg)
	after := time.Now().UnixMilli()

	require.NotNil(t, pb.Timestamp)
	assert.GreaterOrEqual(t, pb.Timestamp.AsTime().UnixMilli(), before)
	assert.LessOrEqual(t, pb.Timestamp.AsTime().UnixMilli(), after)
}

func TestToProtoLiveMessage_NoSequenceNoStreamNoHeaders(t *testing.T) {
	t.Parallel()

	msg := &entities.NatsMessage{
		Subject:   "x",
		Data:      []byte("data"),
		Timestamp: time.UnixMilli(1234).UTC(),
	}

	pb := toProtoLiveMessage(msg)

	assert.Equal(t, "x", pb.Subject)
	assert.Equal(t, uint64(0), pb.Sequence, "nil *uint64 src → zero uint64 dst")
	assert.Nil(t, pb.Stream, "empty string src → nil *string dst")
	assert.Nil(t, pb.Reply, "no reply subject → unset")
	assert.Empty(t, pb.Headers, "nil header map → empty/nil destination")
}

func TestToProtoLiveMessage_ReplySubject(t *testing.T) {
	t.Parallel()

	pb := toProtoLiveMessage(&entities.NatsMessage{Subject: "svc.echo", Data: []byte("hi"), Reply: "_INBOX.abc"})
	require.NotNil(t, pb.Reply)
	assert.Equal(t, "_INBOX.abc", *pb.Reply)
}

func TestToProtoLiveEvent_DeniedSubscription(t *testing.T) {
	t.Parallel()

	pb := toProtoLiveEvent(&entities.LiveEvent{Error: &entities.LiveError{
		Code:    "SUBSCRIBE_PERMISSION_DENIED",
		Message: `no permission to subscribe to "secret.>"`,
		Access:  &entities.AccessCheck{Status: entities.AccessDenied, Operation: "subscribe", Subject: "secret.>"},
	}})

	liveErr := pb.GetError()
	require.NotNil(t, liveErr)
	assert.Equal(t, "SUBSCRIBE_PERMISSION_DENIED", liveErr.GetCode())
	assert.Equal(t, natspb.AccessStatus_ACCESS_STATUS_DENIED, liveErr.GetAccess().GetStatus())
	assert.Equal(t, "subscribe", liveErr.GetAccess().GetOperation())
	assert.Equal(t, "secret.>", liveErr.GetAccess().GetSubject())
	assert.Nil(t, toProtoLiveEvent(&entities.LiveEvent{Error: &entities.LiveError{Code: "X"}}).GetError().Access)
}

func TestToProtoLiveBatchMessage_AutoDetected(t *testing.T) {
	t.Parallel()
	decoded, messageType, sourceID := `{"id":"1"}`, "shop.Order", "src-1"
	pb := toProtoLiveBatchMessage(&entities.LiveMessage{
		NatsMessage: entities.NatsMessage{Subject: "orders.1", Data: []byte{0x0a, 0x01, '1'}},
		Decoded:     &decoded, DecodedType: &messageType, DecodedAuto: true, DecodedSourceID: &sourceID,
	})

	assert.True(t, pb.GetDecodedAuto())
	assert.Equal(t, "src-1", pb.GetDecodedSourceId())
	assert.Equal(t, "shop.Order", pb.GetDecodedType())
	assert.JSONEq(t, decoded, pb.GetDecoded())
}
