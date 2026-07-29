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

	// StringSliceFirst codec: takes first element of each header value list.
	require.Len(t, pb.Headers, 2)
	assert.Equal(t, "abc-123", pb.Headers["X-Trace"])
	assert.Equal(t, "api", pb.Headers["X-Source"])
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
	assert.Empty(t, pb.Headers, "nil header map → empty/nil destination")
}
