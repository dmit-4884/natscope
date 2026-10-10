// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package natsgo

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
)

func kvClient(t *testing.T, config entities.KVBucketConfig) *Client {
	t.Helper()
	_, url := jetStreamServer(t)
	c := dialClient(t, url)
	_, err := c.CreateKVBucket(t.Context(), config)
	require.NoError(t, err)
	return c
}

func TestCreateKVKey_KeyCarriesItsTTLAndExpires(t *testing.T) {
	t.Parallel()
	c := kvClient(t, entities.KVBucketConfig{Bucket: "sessions", LimitMarkerTTL: time.Second})

	rev, err := c.CreateKVKey(t.Context(), "sessions", "s1", []byte("v"), time.Second)
	require.NoError(t, err)
	assert.NotZero(t, rev)

	entry, err := c.GetKVKey(t.Context(), "sessions", "s1")
	require.NoError(t, err)
	assert.Equal(t, time.Second, entry.TTL)

	require.Eventually(t, func() bool {
		_, err := c.GetKVKey(t.Context(), "sessions", "s1")
		return err != nil
	}, 10*time.Second, 200*time.Millisecond, "the key should expire")
}

func TestCreateKVKey_WithoutTTLIsAPlainCreate(t *testing.T) {
	t.Parallel()
	c := kvClient(t, entities.KVBucketConfig{Bucket: "cfg"})

	_, err := c.CreateKVKey(t.Context(), "cfg", "a", []byte("v"), 0)
	require.NoError(t, err)

	entry, err := c.GetKVKey(t.Context(), "cfg", "a")
	require.NoError(t, err)
	assert.Zero(t, entry.TTL)
}

func TestCreateKVKey_RefusesAnExistingKey(t *testing.T) {
	t.Parallel()
	c := kvClient(t, entities.KVBucketConfig{Bucket: "sessions", LimitMarkerTTL: time.Second})
	_, err := c.PutKVKey(t.Context(), "sessions", "s1", []byte("v"), 0)
	require.NoError(t, err)

	_, err = c.CreateKVKey(t.Context(), "sessions", "s1", []byte("v2"), time.Minute)

	var verr *errs.NATSValidationError
	require.ErrorAs(t, err, &verr)
	assert.Contains(t, verr.Description, "already exists")
}

func TestCreateKVKey_RefusesATTLTheBucketDoesNotAllow(t *testing.T) {
	t.Parallel()
	c := kvClient(t, entities.KVBucketConfig{Bucket: "cfg"})

	_, err := c.CreateKVKey(t.Context(), "cfg", "a", []byte("v"), time.Minute)

	var verr *errs.NATSValidationError
	require.ErrorAs(t, err, &verr)
	assert.Contains(t, verr.Description, "does not allow a TTL per key")
}

func TestCreateKVKey_RefusesATTLUnderOneSecond(t *testing.T) {
	t.Parallel()
	c := kvClient(t, entities.KVBucketConfig{Bucket: "sessions", LimitMarkerTTL: time.Second})

	_, err := c.CreateKVKey(t.Context(), "sessions", "a", []byte("v"), 500*time.Millisecond)

	var verr *errs.NATSValidationError
	require.ErrorAs(t, err, &verr)
}

func TestParseMsgTTL(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		in   string
		want time.Duration
	}{
		{"", 0},
		{"never", 0},
		{"30s", 30 * time.Second},
		{"1h0m0s", time.Hour},
		{"90", 90 * time.Second},
		{"junk", 0},
	} {
		assert.Equal(t, tc.want, parseMsgTTL(tc.in), tc.in)
	}
}
