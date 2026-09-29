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
	}, 1024)
	assert.Equal(t, "users.42", v.Key)
	assert.Equal(t, uint64(9), v.Revision)
	assert.Equal(t, created, v.Created)
	assert.Equal(t, &mcptransport.Body{JSON: []byte(`{"name":"ann"}`)}, v.Value)
	assert.False(t, v.Truncated)

	v = newEntryView(&entities.KVEntry{Key: "gone", Operation: "delete"}, 1024)
	assert.Nil(t, v.Value)
	assert.Equal(t, "delete", v.Operation)
}
