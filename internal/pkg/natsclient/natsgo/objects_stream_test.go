// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package natsgo

import (
	"bytes"
	"crypto/sha256"
	"io"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
)

func TestObjectStream_MovesAnObjectPastTheUnaryLimit(t *testing.T) {
	t.Parallel()
	_, url := jetStreamServer(t)
	nc, err := nats.Connect(url)
	require.NoError(t, err)
	t.Cleanup(nc.Close)
	js, err := jetstream.New(nc)
	require.NoError(t, err)
	_, err = js.CreateObjectStore(t.Context(), jetstream.ObjectStoreConfig{Bucket: "BIG"})
	require.NoError(t, err)
	c := dialClient(t, url)
	content := bytes.Repeat([]byte("natscope-"), (maxGetObjectBytes+1<<20)/9)

	info, err := c.PutObjectStream(t.Context(), "BIG", entities.ObjectMeta{Name: "dump.bin"}, bytes.NewReader(content), int64(len(content)))
	require.NoError(t, err)
	assert.Equal(t, uint64(len(content)), info.Size)

	r, got, err := c.OpenObject(t.Context(), "BIG", "dump.bin")
	require.NoError(t, err)
	t.Cleanup(func() { _ = r.Close() })
	assert.Equal(t, uint64(len(content)), got.Size)
	h := sha256.New()
	n, err := io.Copy(h, r)
	require.NoError(t, err)
	assert.Equal(t, int64(len(content)), n)
	assert.Equal(t, sha256.Sum256(content), [32]byte(h.Sum(nil)))
}
