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

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
)

func rawJetStream(t *testing.T, url string) jetstream.JetStream {
	t.Helper()
	nc, err := nats.Connect(url)
	require.NoError(t, err)
	t.Cleanup(nc.Close)
	js, err := jetstream.New(nc)
	require.NoError(t, err)
	return js
}

func kvStreamConfig(t *testing.T, js jetstream.JetStream, bucket string) jetstream.StreamConfig {
	t.Helper()
	s, err := js.Stream(t.Context(), "KV_"+bucket)
	require.NoError(t, err)
	info, err := s.Info(t.Context())
	require.NoError(t, err)
	return info.Config
}

func TestCreateKVBucket_KeyTTLMarkersAndCompression(t *testing.T) {
	t.Parallel()
	_, url := jetStreamServer(t)
	c := dialClient(t, url)

	info, err := c.CreateKVBucket(t.Context(), entities.KVBucketConfig{
		Bucket: "cfg", LimitMarkerTTL: 5 * time.Second, Compression: true,
	})

	require.NoError(t, err)
	assert.Equal(t, 5*time.Second, info.LimitMarkerTTL)
	assert.True(t, info.IsCompressed)
	cfg := kvStreamConfig(t, rawJetStream(t, url), "cfg")
	assert.True(t, cfg.AllowMsgTTL, "per-key TTL needs AllowMsgTTL on the stream")
	assert.Equal(t, 5*time.Second, cfg.SubjectDeleteMarkerTTL)
	assert.Equal(t, jetstream.S2Compression, cfg.Compression)
}

func TestGetKVBucket_ReportsLimits(t *testing.T) {
	t.Parallel()
	_, url := jetStreamServer(t)
	c := dialClient(t, url)
	_, err := c.CreateKVBucket(t.Context(), entities.KVBucketConfig{Bucket: "cfg", MaxValueSize: 1024, MaxBytes: 1 << 20})
	require.NoError(t, err)

	info, err := c.GetKVBucket(t.Context(), "cfg")

	require.NoError(t, err)
	assert.Equal(t, int32(1024), info.MaxValueSize)
	assert.Equal(t, int64(1<<20), info.MaxBytes)
	assert.Zero(t, info.LimitMarkerTTL)
}

func TestUpdateKVBucket_AppliesSettingsAndKeepsTheRest(t *testing.T) {
	t.Parallel()
	_, url := jetStreamServer(t)
	c := dialClient(t, url)
	js := rawJetStream(t, url)
	kv, err := js.CreateKeyValue(t.Context(), jetstream.KeyValueConfig{
		Bucket:    "cfg",
		RePublish: &jetstream.RePublish{Source: ">", Destination: "repub.>"},
	})
	require.NoError(t, err)
	_, err = kv.PutString(t.Context(), "a", "1")
	require.NoError(t, err)

	info, err := c.UpdateKVBucket(t.Context(), "cfg", entities.KVBucketSettings{
		Description:    "edited",
		History:        5,
		TTL:            30 * time.Second,
		MaxValueSize:   2048,
		MaxBytes:       1 << 20,
		Replicas:       1,
		Compression:    true,
		LimitMarkerTTL: 2 * time.Second,
		Metadata:       map[string]string{"team": "core"},
	})

	require.NoError(t, err)
	assert.Equal(t, "edited", info.Description)
	assert.Equal(t, uint8(5), info.History)
	assert.Equal(t, 30*time.Second, info.TTL)
	assert.Equal(t, int32(2048), info.MaxValueSize)
	assert.Equal(t, int64(1<<20), info.MaxBytes)
	assert.True(t, info.IsCompressed)
	assert.Equal(t, 2*time.Second, info.LimitMarkerTTL)
	assert.Equal(t, "core", info.Metadata["team"])

	cfg := kvStreamConfig(t, js, "cfg")
	assert.Equal(t, 30*time.Second, cfg.Duplicates, "the duplicate window may not exceed a TTL under two minutes")
	require.NotNil(t, cfg.RePublish, "settings the edit does not cover stay")
	assert.Equal(t, "repub.>", cfg.RePublish.Destination)
	assert.True(t, cfg.DenyDelete)
	assert.True(t, cfg.AllowDirect)
	entry, err := kv.Get(t.Context(), "a")
	require.NoError(t, err)
	assert.Equal(t, "1", string(entry.Value()))
}

func TestUpdateKVBucket_ZeroLimitsMeanUnlimited(t *testing.T) {
	t.Parallel()
	_, url := jetStreamServer(t)
	c := dialClient(t, url)
	_, err := c.CreateKVBucket(t.Context(), entities.KVBucketConfig{Bucket: "cfg", MaxValueSize: 10, MaxBytes: 1 << 20})
	require.NoError(t, err)

	_, err = c.UpdateKVBucket(t.Context(), "cfg", entities.KVBucketSettings{History: 1})

	require.NoError(t, err)
	cfg := kvStreamConfig(t, rawJetStream(t, url), "cfg")
	assert.Equal(t, int32(-1), cfg.MaxMsgSize)
	assert.Equal(t, int64(-1), cfg.MaxBytes)
	assert.Equal(t, 2*time.Minute, cfg.Duplicates)
}

func TestUpdateKVBucket_KeyTTLStaysOnceAllowed(t *testing.T) {
	t.Parallel()
	_, url := jetStreamServer(t)
	c := dialClient(t, url)
	_, err := c.CreateKVBucket(t.Context(), entities.KVBucketConfig{Bucket: "cfg", LimitMarkerTTL: 5 * time.Second})
	require.NoError(t, err)

	_, err = c.UpdateKVBucket(t.Context(), "cfg", entities.KVBucketSettings{History: 1})

	var verr *errs.NATSValidationError
	require.ErrorAs(t, err, &verr)
	assert.Contains(t, verr.Description, "per-key TTL")

	info, err := c.UpdateKVBucket(t.Context(), "cfg", entities.KVBucketSettings{History: 1, LimitMarkerTTL: time.Minute})
	require.NoError(t, err)
	assert.Equal(t, time.Minute, info.LimitMarkerTTL)
}

func TestUpdateKVBucket_RefusesWhatIsNotABucket(t *testing.T) {
	t.Parallel()
	_, url := jetStreamServer(t)
	c := dialClient(t, url)
	_, err := rawJetStream(t, url).CreateStream(t.Context(), jetstream.StreamConfig{Name: "KV_plain", Subjects: []string{"plain.>"}})
	require.NoError(t, err)

	_, err = c.UpdateKVBucket(t.Context(), "plain", entities.KVBucketSettings{History: 1})
	require.ErrorIs(t, err, errs.ErrNotAKVOrObjectBucket)

	_, err = c.UpdateKVBucket(t.Context(), "missing", entities.KVBucketSettings{History: 1})
	require.ErrorIs(t, err, errs.ErrBucketNotFound)
}
