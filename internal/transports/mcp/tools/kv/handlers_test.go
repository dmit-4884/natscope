// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package kv

import (
	"encoding/base64"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/dmit-4884/natscope/internal/entities"

	mcptransport "github.com/dmit-4884/natscope/internal/transports/mcp"
)

func TestNewEntryView(t *testing.T) {
	t.Parallel()
	created := time.Unix(1790000000, 0).UTC()

	v := newEntryView(&entities.KVEntry{
		Key: "users.42", Value: base64.StdEncoding.EncodeToString([]byte(`{"name":"ann"}`)), Revision: 9, Created: created, Operation: "put",
	}, 1024, nil)
	assert.Equal(t, "users.42", v.Key)
	assert.Equal(t, uint64(9), v.Revision)
	assert.Equal(t, created, v.Created)
	assert.Equal(t, &mcptransport.Body{JSON: []byte(`{"name":"ann"}`)}, v.Value)
	assert.False(t, v.Truncated)

	v = newEntryView(&entities.KVEntry{Key: "gone", Operation: "delete"}, 1024, nil)
	assert.Nil(t, v.Value)
	assert.Equal(t, "delete", v.Operation)

	proto := &entities.KVEntry{Key: "users.7", Value: base64.StdEncoding.EncodeToString([]byte{0x0a, 0x03, 'a', 'n', 'n'})}
	decoded := &entities.DecodeResult{Success: true, Decoded: []byte(`{"name":"ann"}`), MessageType: "shop.User", Auto: true}
	v = newEntryView(proto, 1024, decoded)
	assert.Equal(t, "shop.User", v.DecodedType)
	assert.True(t, v.DecodedAuto)
	assert.JSONEq(t, `{"name":"ann"}`, string(v.Decoded))
	assert.Nil(t, v.Value)

	v = newEntryView(proto, 4, decoded)
	assert.Nil(t, v.Decoded, "a decoded form over the budget falls back to the raw value")
	assert.NotNil(t, v.Value)

	v = newEntryView(proto, 1024, &entities.DecodeResult{MessageType: "shop.User", Error: "bad wire"})
	assert.Equal(t, "bad wire", v.DecodeError)
	assert.NotNil(t, v.Value)

	v = newEntryView(&entities.KVEntry{Key: "s1", Created: created, TTL: 90 * time.Second}, 1024, nil)
	assert.Equal(t, "1m30s", v.TTL)
	assert.Equal(t, created, v.Created)

	v = newEntryView(&entities.KVEntry{Key: "s2"}, 1024, nil)
	assert.Empty(t, v.TTL)
}
