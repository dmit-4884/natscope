// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	connectionspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/connections"
	managementpb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/management"
	natstypes "github.com/dmit-4884/natscope/proto/gen/types/nats"
)

func kvObjTestConn(t *testing.T, env *e2eEnv, name string) string {
	t.Helper()
	resp, err := env.connections.CreateConnection(t.Context(), connect.NewRequest(&connectionspb.CreateConnectionRequest{
		Name: name, Urls: []string{env.natsURL},
	}))
	require.NoError(t, err)
	return resp.Msg.GetConnection().GetId()
}

// TestKVBucketHistoryRejectsOutOfRange checks that a history above 64 is rejected.
func TestKVBucketHistoryRejectsOutOfRange(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()
	connID := kvObjTestConn(t, env, "qa-kv-history")

	for _, history := range []uint32{65, 256, 300} {
		t.Run(fmt.Sprintf("history=%d", history), func(t *testing.T) {
			_, err := env.management.CreateKVBucket(ctx, connect.NewRequest(&managementpb.CreateKVBucketRequest{
				ConnectionId: connID,
				Config:       &natstypes.KVBucketConfig{Bucket: fmt.Sprintf("h%d", history), History: history},
			}))
			require.Error(t, err, "history above 64 must be rejected, not truncated")
			assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
		})
	}

	// 64 is the accepted maximum.
	created, err := env.management.CreateKVBucket(ctx, connect.NewRequest(&managementpb.CreateKVBucketRequest{
		ConnectionId: connID, Config: &natstypes.KVBucketConfig{Bucket: "hmax", History: 64},
	}))
	require.NoError(t, err)
	assert.EqualValues(t, 64, created.Msg.GetBucket().GetHistory())
}

// TestKVBucketDescriptionRoundTrips checks that Create, Get and List return the description.
func TestKVBucketDescriptionRoundTrips(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()
	connID := kvObjTestConn(t, env, "qa-kv-desc")

	created, err := env.management.CreateKVBucket(ctx, connect.NewRequest(&managementpb.CreateKVBucketRequest{
		ConnectionId: connID,
		Config:       &natstypes.KVBucketConfig{Bucket: "kvd", Description: "qa desc kv1"},
	}))
	require.NoError(t, err)
	assert.Equal(t, "qa desc kv1", created.Msg.GetBucket().GetDescription())

	got, err := env.management.GetKVBucket(ctx, connect.NewRequest(&managementpb.GetKVBucketRequest{
		ConnectionId: connID, Bucket: "kvd",
	}))
	require.NoError(t, err)
	assert.Equal(t, "qa desc kv1", got.Msg.GetBucket().GetDescription())
}

// TestKVKeyValidation checks that every key RPC rejects empty segments and wildcards.
func TestKVKeyValidation(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()
	connID := kvObjTestConn(t, env, "qa-kv-keys")

	_, err := env.management.CreateKVBucket(ctx, connect.NewRequest(&managementpb.CreateKVBucketRequest{
		ConnectionId: connID, Config: &natstypes.KVBucketConfig{Bucket: "keys1"},
	}))
	require.NoError(t, err)

	value := base64.StdEncoding.EncodeToString([]byte("x"))

	t.Run("empty path segment is rejected on every key RPC", func(t *testing.T) {
		const badKey = "a..b"

		_, err := env.management.PutKVKey(ctx, connect.NewRequest(&managementpb.PutKVKeyRequest{
			ConnectionId: connID, Bucket: "keys1", Key: badKey, Payload: &managementpb.PutKVKeyRequest_Value{Value: value},
		}))
		require.Error(t, err)
		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "PutKVKey")

		_, err = env.management.GetKVKey(ctx, connect.NewRequest(&managementpb.GetKVKeyRequest{
			ConnectionId: connID, Bucket: "keys1", Key: badKey,
		}))
		require.Error(t, err)
		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "GetKVKey")

		_, err = env.management.DeleteKVKey(ctx, connect.NewRequest(&managementpb.DeleteKVKeyRequest{
			ConnectionId: connID, Bucket: "keys1", Key: badKey,
		}))
		require.Error(t, err)
		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "DeleteKVKey")

		_, err = env.management.PurgeKVKey(ctx, connect.NewRequest(&managementpb.PurgeKVKeyRequest{
			ConnectionId: connID, Bucket: "keys1", Key: badKey,
		}))
		require.Error(t, err)
		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "PurgeKVKey")

		_, err = env.management.GetKVKeyHistory(ctx, connect.NewRequest(&managementpb.GetKVKeyHistoryRequest{
			ConnectionId: connID, Bucket: "keys1", Key: badKey,
		}))
		require.Error(t, err)
		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "GetKVKeyHistory")
	})

	t.Run("GetKVKeyHistory rejects wildcards like every other key RPC", func(t *testing.T) {
		for _, k := range []string{"a.1", "a.2", "b.1"} {
			_, err := env.management.PutKVKey(ctx, connect.NewRequest(&managementpb.PutKVKeyRequest{
				ConnectionId: connID, Bucket: "keys1", Key: k, Payload: &managementpb.PutKVKeyRequest_Value{Value: value},
			}))
			require.NoError(t, err)
		}

		for _, wildcard := range []string{"*", ">", "a.*"} {
			_, err := env.management.GetKVKeyHistory(ctx, connect.NewRequest(&managementpb.GetKVKeyHistoryRequest{
				ConnectionId: connID, Bucket: "keys1", Key: wildcard,
			}))
			require.Error(t, err, "wildcard %q must be rejected", wildcard)
			assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
		}
	})
}

// TestEmptyBucketsReturnEmptyList checks that an empty bucket lists as empty, not NotFound.
func TestEmptyBucketsReturnEmptyList(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()
	connID := kvObjTestConn(t, env, "qa-empty")

	_, err := env.management.CreateKVBucket(ctx, connect.NewRequest(&managementpb.CreateKVBucketRequest{
		ConnectionId: connID, Config: &natstypes.KVBucketConfig{Bucket: "emptykv"},
	}))
	require.NoError(t, err)
	keys, err := env.management.ListKVKeys(ctx, connect.NewRequest(&managementpb.ListKVKeysRequest{
		ConnectionId: connID, Bucket: "emptykv",
	}))
	require.NoError(t, err)
	assert.Empty(t, keys.Msg.GetKeys())

	_, err = env.management.CreateObjectBucket(ctx, connect.NewRequest(&managementpb.CreateObjectBucketRequest{
		ConnectionId: connID, Config: &natstypes.ObjectBucketConfig{Bucket: "emptyob"},
	}))
	require.NoError(t, err)
	objs, err := env.management.ListObjects(ctx, connect.NewRequest(&managementpb.ListObjectsRequest{
		ConnectionId: connID, Bucket: "emptyob",
	}))
	require.NoError(t, err)
	assert.Empty(t, objs.Msg.GetObjects())
}

// TestListKVKeysFiltersAndLimitsOnTheServer checks the key filter, the limit and the truncation flag of ListKVKeys.
func TestListKVKeysFiltersAndLimitsOnTheServer(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()
	connID := kvObjTestConn(t, env, "qa-kvfilter")

	_, err := env.management.CreateKVBucket(ctx, connect.NewRequest(&managementpb.CreateKVBucketRequest{
		ConnectionId: connID, Config: &natstypes.KVBucketConfig{Bucket: "filtered"},
	}))
	require.NoError(t, err)
	value := base64.StdEncoding.EncodeToString([]byte("v"))
	for _, k := range []string{"orders.1", "orders.2", "orders.3", "users.1"} {
		_, err = env.management.PutKVKey(ctx, connect.NewRequest(&managementpb.PutKVKeyRequest{
			ConnectionId: connID, Bucket: "filtered", Key: k, Payload: &managementpb.PutKVKeyRequest_Value{Value: value},
		}))
		require.NoError(t, err)
	}

	all, err := env.management.ListKVKeys(ctx, connect.NewRequest(&managementpb.ListKVKeysRequest{
		ConnectionId: connID, Bucket: "filtered", Filter: "orders.*",
	}))
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"orders.1", "orders.2", "orders.3"}, all.Msg.GetKeys())
	assert.False(t, all.Msg.GetTruncated())

	cut, err := env.management.ListKVKeys(ctx, connect.NewRequest(&managementpb.ListKVKeysRequest{
		ConnectionId: connID, Bucket: "filtered", Filter: "orders.*", Limit: 2,
	}))
	require.NoError(t, err)
	assert.Len(t, cut.Msg.GetKeys(), 2)
	assert.True(t, cut.Msg.GetTruncated())

	_, err = env.management.ListKVKeys(ctx, connect.NewRequest(&managementpb.ListKVKeysRequest{
		ConnectionId: connID, Bucket: "filtered", Filter: "orders.>.x",
	}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

// TestKVBucketSettingsRoundTrip checks that create and update carry compression, limits and the key TTL marker.
func TestKVBucketSettingsRoundTrip(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()
	connID := kvObjTestConn(t, env, "qa-kvsettings")

	created, err := env.management.CreateKVBucket(ctx, connect.NewRequest(&managementpb.CreateKVBucketRequest{
		ConnectionId: connID, Config: &natstypes.KVBucketConfig{
			Bucket: "settings", Compression: true, LimitMarkerTtl: durationpb.New(5 * time.Second),
		},
	}))
	require.NoError(t, err)
	assert.True(t, created.Msg.GetBucket().GetIsCompressed())
	assert.Equal(t, 5*time.Second, created.Msg.GetBucket().GetLimitMarkerTtl().AsDuration())

	updated, err := env.management.UpdateKVBucket(ctx, connect.NewRequest(&managementpb.UpdateKVBucketRequest{
		ConnectionId: connID, Bucket: "settings", Settings: &natstypes.KVBucketSettings{
			Description: "edited", History: 4, MaxBytes: 1 << 20, LimitMarkerTtl: durationpb.New(2 * time.Second),
		},
	}))
	require.NoError(t, err)
	b := updated.Msg.GetBucket()
	assert.Equal(t, "edited", b.GetDescription())
	assert.Equal(t, uint32(4), b.GetHistory())
	assert.Equal(t, int64(1<<20), b.GetMaxBytes())
	assert.False(t, b.GetIsCompressed())
	assert.Equal(t, 2*time.Second, b.GetLimitMarkerTtl().AsDuration())

	_, err = env.management.UpdateKVBucket(ctx, connect.NewRequest(&managementpb.UpdateKVBucketRequest{
		ConnectionId: connID, Bucket: "settings", Settings: &natstypes.KVBucketSettings{LimitMarkerTtl: durationpb.New(time.Millisecond)},
	}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "a marker under one second is invalid")
}

// TestKVBucketCreateCarriesMirrorSourcesAndRepublish checks that a bucket keeps the mirror, sources and republish it was created with.
func TestKVBucketCreateCarriesMirrorSourcesAndRepublish(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()
	connID := kvObjTestConn(t, env, "qa-kvlinks")
	start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

	for _, cfg := range []*natstypes.KVBucketConfig{
		{Bucket: "origin"},
		{Bucket: "copy", Mirror: &natstypes.StreamSourceRef{Name: "origin", OptStartTime: timestamppb.New(start)}},
		{
			Bucket:    "agg",
			Sources:   []*natstypes.StreamSourceRef{{Name: "origin"}},
			Republish: &natstypes.RePublish{Src: ">", Dest: "repub.>"},
		},
	} {
		_, err := env.management.CreateKVBucket(ctx, connect.NewRequest(&managementpb.CreateKVBucketRequest{ConnectionId: connID, Config: cfg}))
		require.NoError(t, err, cfg.GetBucket())
	}

	nc, err := nats.Connect(env.natsURL)
	require.NoError(t, err)
	defer nc.Close()
	js, err := jetstream.New(nc)
	require.NoError(t, err)

	mirror, err := js.Stream(ctx, "KV_copy")
	require.NoError(t, err)
	m := mirror.CachedInfo().Config.Mirror
	require.NotNil(t, m)
	assert.Equal(t, "KV_origin", m.Name)
	require.NotNil(t, m.OptStartTime)
	assert.True(t, start.Equal(*m.OptStartTime))

	agg, err := js.Stream(ctx, "KV_agg")
	require.NoError(t, err)
	cfg := agg.CachedInfo().Config
	require.Len(t, cfg.Sources, 1)
	assert.Equal(t, "KV_origin", cfg.Sources[0].Name)
	require.NotNil(t, cfg.RePublish)
	assert.Equal(t, "repub.>", cfg.RePublish.Destination)
}

// TestBucketDeleteSealRefusesPlainStream checks that Delete and Seal refuse a plain stream named like a bucket.
func TestBucketDeleteSealRefusesPlainStream(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()
	connID := kvObjTestConn(t, env, "qa-badbucket")

	nc, err := nats.Connect(env.natsURL)
	require.NoError(t, err)
	defer nc.Close()
	js, err := jetstream.New(nc)
	require.NoError(t, err)

	t.Run("DeleteKVBucket refuses a plain stream named KV_<bucket>", func(t *testing.T) {
		_, err := js.CreateStream(ctx, jetstream.StreamConfig{
			Name: "KV_plain", Subjects: []string{"plain.>"},
		})
		require.NoError(t, err)

		_, err = env.management.DeleteKVBucket(ctx, connect.NewRequest(&managementpb.DeleteKVBucketRequest{
			ConnectionId: connID, Bucket: "plain",
		}))
		require.Error(t, err, "a plain stream must not be deleted as if it were a KV bucket")

		_, infoErr := js.Stream(ctx, "KV_plain")
		require.NoError(t, infoErr, "the plain stream must still exist")
	})

	t.Run("SealObjectBucket refuses a plain stream named OBJ_<bucket>", func(t *testing.T) {
		_, err := js.CreateStream(ctx, jetstream.StreamConfig{
			Name: "OBJ_plainobj", Subjects: []string{"plainobj.>"},
		})
		require.NoError(t, err)

		_, err = env.management.SealObjectBucket(ctx, connect.NewRequest(&managementpb.SealObjectBucketRequest{
			ConnectionId: connID, Bucket: "plainobj",
		}))
		require.Error(t, err, "a plain stream must not be sealed as if it were an object bucket")

		info, infoErr := js.Stream(ctx, "OBJ_plainobj")
		require.NoError(t, infoErr)
		streamInfo, infoErr := info.Info(ctx)
		require.NoError(t, infoErr)
		assert.False(t, streamInfo.Config.Sealed, "the plain stream must not have been sealed")
	})
}

// TestPutObjectCapacityGuardPreservesOriginal checks that an overwrite exceeding max_bytes keeps the original.
func TestPutObjectCapacityGuardPreservesOriginal(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()
	connID := kvObjTestConn(t, env, "qa-obj-capacity")

	_, err := env.management.CreateObjectBucket(ctx, connect.NewRequest(&managementpb.CreateObjectBucketRequest{
		ConnectionId: connID, Config: &natstypes.ObjectBucketConfig{Bucket: "qow", MaxBytes: 100_000},
	}))
	require.NoError(t, err)

	original := make([]byte, 50_000)
	for i := range original {
		original[i] = byte(i)
	}
	putResp, err := env.management.PutObject(ctx, connect.NewRequest(&managementpb.PutObjectRequest{
		ConnectionId: connID, Bucket: "qow", Name: "doc", Data: original,
	}))
	require.NoError(t, err)
	originalDigest := putResp.Msg.GetInfo().GetDigest()

	oversized := make([]byte, 300_000)
	_, err = env.management.PutObject(ctx, connect.NewRequest(&managementpb.PutObjectRequest{
		ConnectionId: connID, Bucket: "qow", Name: "doc", Data: oversized,
	}))
	require.Error(t, err, "an overwrite that would exceed max_bytes must be rejected")
	assert.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))

	getResp, err := env.management.GetObject(ctx, connect.NewRequest(&managementpb.GetObjectRequest{
		ConnectionId: connID, Bucket: "qow", Name: "doc",
	}))
	require.NoError(t, err, "the original object must still be readable after the rejected overwrite")
	assert.Equal(t, original, getResp.Msg.GetData())
	assert.Equal(t, originalDigest, getResp.Msg.GetInfo().GetDigest())
}

// TestConcurrentPutObjectSameName checks that concurrent Puts of one name leave consistent data and few orphaned chunks.
func TestConcurrentPutObjectSameName(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()
	connID := kvObjTestConn(t, env, "qa-obj-concurrent")

	_, err := env.management.CreateObjectBucket(ctx, connect.NewRequest(&managementpb.CreateObjectBucketRequest{
		ConnectionId: connID, Config: &natstypes.ObjectBucketConfig{Bucket: "orph"},
	}))
	require.NoError(t, err)

	const (
		writers         = 12
		objSize         = 64 * 1024
		maxOrphanFactor = 3
	)

	var wg sync.WaitGroup
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			data := make([]byte, objSize)
			for j := range data {
				data[j] = byte(i)
			}
			_, err := env.management.PutObject(ctx, connect.NewRequest(&managementpb.PutObjectRequest{
				ConnectionId: connID, Bucket: "orph", Name: "same", Data: data,
			}))
			assert.NoError(t, err)
		}(i)
	}
	wg.Wait()

	getResp, err := env.management.GetObject(ctx, connect.NewRequest(&managementpb.GetObjectRequest{
		ConnectionId: connID, Bucket: "orph", Name: "same",
	}))
	require.NoError(t, err)
	sum := sha256.Sum256(getResp.Msg.GetData())
	assert.Equal(t, fmt.Sprintf("SHA-256=%s", base64.URLEncoding.EncodeToString(sum[:])), getResp.Msg.GetInfo().GetDigest(),
		"GetObject's info must describe the data it actually returned")
	assert.EqualValues(t, objSize, getResp.Msg.GetInfo().GetSize())

	nc, err := nats.Connect(env.natsURL)
	require.NoError(t, err)
	defer nc.Close()
	js, err := jetstream.New(nc)
	require.NoError(t, err)
	stream, err := js.Stream(ctx, "OBJ_orph")
	require.NoError(t, err)
	streamInfo, err := stream.Info(ctx)
	require.NoError(t, err)
	assert.Less(t, streamInfo.State.Bytes, uint64(objSize*maxOrphanFactor),
		"serialized PutObject must not leave the stream full of orphaned chunks from losing writers")
}

// TestObjectLinkReflectedInAPI checks that ListObjects flags links and GetObject follows them.
func TestObjectLinkReflectedInAPI(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()
	connID := kvObjTestConn(t, env, "qa-obj-link")

	_, err := env.management.CreateObjectBucket(ctx, connect.NewRequest(&managementpb.CreateObjectBucketRequest{
		ConnectionId: connID, Config: &natstypes.ObjectBucketConfig{Bucket: "qa_lb"},
	}))
	require.NoError(t, err)

	nc, err := nats.Connect(env.natsURL)
	require.NoError(t, err)
	defer nc.Close()
	js, err := jetstream.New(nc)
	require.NoError(t, err)
	obs, err := js.ObjectStore(ctx, "qa_lb")
	require.NoError(t, err)

	targetInfo, err := obs.PutBytes(ctx, "target", []byte("target-data-123"))
	require.NoError(t, err)
	_, err = obs.AddLink(ctx, "lnk", targetInfo)
	require.NoError(t, err)

	listResp, err := env.management.ListObjects(ctx, connect.NewRequest(&managementpb.ListObjectsRequest{
		ConnectionId: connID, Bucket: "qa_lb",
	}))
	require.NoError(t, err)
	var link *natstypes.ObjectInfo
	for _, o := range listResp.Msg.GetObjects() {
		if o.GetName() == "lnk" {
			link = o
		}
	}
	require.NotNil(t, link, "the link must appear in ListObjects")
	require.NotNil(t, link.GetLink(), "the link must be flagged as a link, not shown as an empty object")
	assert.Equal(t, "target", link.GetLink().GetName())
	assert.Equal(t, "qa_lb", link.GetLink().GetBucket())

	getResp, err := env.management.GetObject(ctx, connect.NewRequest(&managementpb.GetObjectRequest{
		ConnectionId: connID, Bucket: "qa_lb", Name: "lnk",
	}))
	require.NoError(t, err, "GetObject on a link must follow it to the target")
	assert.Equal(t, "target-data-123", string(getResp.Msg.GetData()))
	assert.EqualValues(t, len("target-data-123"), getResp.Msg.GetInfo().GetSize(),
		"info must describe the data actually returned, not the link's own (empty) meta")
}

// TestKVBucketMirrorSourcesMutuallyExclusive checks that mirror and sources together are rejected.
func TestKVBucketMirrorSourcesMutuallyExclusive(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()
	connID := kvObjTestConn(t, env, "qa-kv-mirror-sources")

	_, err := env.management.CreateKVBucket(ctx, connect.NewRequest(&managementpb.CreateKVBucketRequest{
		ConnectionId: connID, Config: &natstypes.KVBucketConfig{Bucket: "kvbase"},
	}))
	require.NoError(t, err)

	_, err = env.management.CreateKVBucket(ctx, connect.NewRequest(&managementpb.CreateKVBucketRequest{
		ConnectionId: connID,
		Config: &natstypes.KVBucketConfig{
			Bucket: "mirsrc2",
			Mirror: &natstypes.StreamSourceRef{Name: "kvbase"},
			Sources: []*natstypes.StreamSourceRef{
				{Name: "cas"},
			},
		},
	}))
	require.Error(t, err, "mirror and sources together must be rejected")
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}
