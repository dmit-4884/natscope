// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublishHistoryNew(t *testing.T) {
	t.Parallel()

	t.Run("WithoutInit", func(t *testing.T) {
		t.Parallel()
		h := PublishHistoryNew()
		require.NotNil(t, h)
		assert.NotEmpty(t, h.Id)
		assert.Positive(t, h.CreatedAt)
		assert.Nil(t, h.ConnectionID)
		assert.Empty(t, h.ConnectionURL)
		assert.Empty(t, h.Subject)
		assert.Empty(t, h.MessageType)
		assert.False(t, h.Success)
		assert.Nil(t, h.Sequence)
		assert.Nil(t, h.Error)
	})

	t.Run("WithInit", func(t *testing.T) {
		t.Parallel()
		seq := uint64(42)
		h := PublishHistoryNew(func(h *PublishHistory) {
			h.ConnectionURL = "nats://localhost:4222"
			h.Stream = "ORDERS"
			h.Subject = "orders.created"
			h.MessageType = "api.v1.Order"
			h.PayloadJSON = `{"id": 1}`
			h.PayloadSize = 8
			h.Sequence = &seq
			h.Success = true
		})
		assert.Equal(t, "ORDERS", h.Stream)
		assert.True(t, h.Success)
		require.NotNil(t, h.Sequence)
		assert.Equal(t, uint64(42), *h.Sequence)
	})

	t.Run("WithNilInit", func(t *testing.T) {
		t.Parallel()
		h := PublishHistoryNew(nil)
		require.NotNil(t, h)
	})
}
