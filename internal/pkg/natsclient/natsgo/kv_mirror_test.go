// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKVMirror_IsListed(t *testing.T) {
	t.Parallel()
	_, url := jetStreamServer(t)
	nc, err := nats.Connect(url)
	require.NoError(t, err)
	t.Cleanup(nc.Close)
	js, err := jetstream.New(nc)
	require.NoError(t, err)
	origin, err := js.CreateKeyValue(t.Context(), jetstream.KeyValueConfig{Bucket: "CONFIG"})
	require.NoError(t, err)
	for _, key := range []string{"a", "b"} {
		_, err = origin.PutString(t.Context(), key, "v")
		require.NoError(t, err)
	}
	_, err = js.CreateKeyValue(t.Context(), jetstream.KeyValueConfig{Bucket: "CONFIG_EU", Mirror: &jetstream.StreamSource{Name: "CONFIG"}})
	require.NoError(t, err)
	mirror, err := js.Stream(t.Context(), "KV_CONFIG_EU")
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		info, infoErr := mirror.Info(t.Context())
		return infoErr == nil && info.State.Msgs == 2
	}, 5*time.Second, 20*time.Millisecond)
	c := dialClient(t, url)

	buckets, err := c.ListKVBuckets(t.Context())
	require.NoError(t, err)
	names := make([]string, 0, len(buckets))
	for _, b := range buckets {
		names = append(names, b.Bucket)
	}
	assert.ElementsMatch(t, []string{"CONFIG", "CONFIG_EU"}, names)
	for _, b := range buckets {
		if b.Bucket == "CONFIG_EU" {
			assert.Equal(t, uint64(2), b.Values)
		}
	}
}
