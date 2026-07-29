// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"context"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

// stubDescriptors is an in-memory DescriptorsLookup that counts lookups per key
// so tests can prove cache hits vs rebuilds.
type stubDescriptors struct {
	byTag    map[string]*entities.ProtoDescriptor
	byFP     map[string]*entities.ProtoDescriptor
	tagCalls map[string]int
	fpCalls  int
}

func newStub() *stubDescriptors {
	return &stubDescriptors{
		byTag:    map[string]*entities.ProtoDescriptor{},
		byFP:     map[string]*entities.ProtoDescriptor{},
		tagCalls: map[string]int{},
	}
}

func (s *stubDescriptors) GetBySourceTag(_ context.Context, sourceID, tag string) (*entities.ProtoDescriptor, error) {
	key := sourceID + "\x00" + tag
	s.tagCalls[key]++
	d, ok := s.byTag[key]
	if !ok {
		return nil, errs.ErrProtoDescriptorNotFound
	}
	return d, nil
}

func (s *stubDescriptors) FindByFingerprint(_ context.Context, fp string) (*entities.ProtoDescriptor, error) {
	s.fpCalls++
	d, ok := s.byFP[fp]
	if !ok {
		return nil, errs.ErrProtoDescriptorNotFound
	}
	return d, nil
}

// descriptorSetBytes builds a valid serialized FileDescriptorSet with one
// message pkg.msg so ParseDescriptorSet yields a real descriptor.
func descriptorSetBytes(t *testing.T, pkg, msg string) []byte {
	t.Helper()
	file := &descriptorpb.FileDescriptorProto{
		Name:        proto.String(msg + ".proto"),
		Package:     proto.String(pkg),
		MessageType: []*descriptorpb.DescriptorProto{{Name: proto.String(msg)}},
		Syntax:      proto.String("proto3"),
	}
	set := &descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{file}}
	data, err := proto.Marshal(set)
	require.NoError(t, err)
	return data
}

func (s *stubDescriptors) put(sourceID, tag string, data []byte) *entities.ProtoDescriptor {
	d := &entities.ProtoDescriptor{SourceID: sourceID, Tag: tag, DescriptorSet: data}
	s.byTag[sourceID+"\x00"+tag] = d
	if len(data) > 0 {
		s.byFP[fingerprint(data)] = d
	}
	return d
}

func TestGetOrBuild_Validation(t *testing.T) {
	t.Parallel()
	c := NewCache(newStub())

	_, err := c.GetOrBuild(t.Context(), "", "v1")
	assert.ErrorIs(t, err, errs.ErrMappingSourceNotFound)

	_, err = c.GetOrBuild(t.Context(), "src", "")
	assert.ErrorIs(t, err, errs.ErrMappingDescriptorMissing)
}

func TestGetOrBuild_DescriptorNotFound(t *testing.T) {
	t.Parallel()
	c := NewCache(newStub())

	_, err := c.GetOrBuild(t.Context(), "src", "v1")
	assert.ErrorIs(t, err, errs.ErrMappingDescriptorMissing)
}

func TestGetOrBuild_EmptyDescriptorSet(t *testing.T) {
	t.Parallel()
	stub := newStub()
	stub.put("src", "v1", nil) // descriptor exists but carries no bytes
	c := NewCache(stub)

	_, err := c.GetOrBuild(t.Context(), "src", "v1")
	assert.ErrorIs(t, err, errs.ErrMappingDescriptorMissing)
}

func TestGetOrBuild_MissThenHit(t *testing.T) {
	t.Parallel()
	stub := newStub()
	data := descriptorSetBytes(t, "test.pkg", "Thing")
	stub.put("src", "v1", data)
	c := NewCache(stub)

	snap, err := c.GetOrBuild(t.Context(), "src", "v1")
	require.NoError(t, err)
	assert.Equal(t, "src", snap.SourceID)
	assert.Equal(t, "v1", snap.Tag)
	assert.Equal(t, fingerprint(data), snap.Fingerprint)
	assert.Contains(t, snap.Messages, "test.pkg.Thing")

	// Second call is served from cache: same pointer, no extra lookup.
	again, err := c.GetOrBuild(t.Context(), "src", "v1")
	require.NoError(t, err)
	assert.Same(t, snap, again)
	assert.Equal(t, 1, stub.tagCalls["src\x00v1"])
}

func TestInvalidate_ForcesRebuild(t *testing.T) {
	t.Parallel()
	stub := newStub()
	stub.put("src", "v1", descriptorSetBytes(t, "test.pkg", "Thing"))
	c := NewCache(stub)

	first, err := c.GetOrBuild(t.Context(), "src", "v1")
	require.NoError(t, err)

	c.Invalidate("src", "v1")

	second, err := c.GetOrBuild(t.Context(), "src", "v1")
	require.NoError(t, err)
	assert.NotSame(t, first, second, "invalidated entry must be rebuilt")
	assert.Equal(t, 2, stub.tagCalls["src\x00v1"])
}

func TestInvalidateSource_DropsOnlyThatSource(t *testing.T) {
	t.Parallel()
	stub := newStub()
	stub.put("src-A", "v1", descriptorSetBytes(t, "a.pkg", "A"))
	stub.put("src-B", "v1", descriptorSetBytes(t, "b.pkg", "B"))
	c := NewCache(stub)

	_, err := c.GetOrBuild(t.Context(), "src-A", "v1")
	require.NoError(t, err)
	_, err = c.GetOrBuild(t.Context(), "src-B", "v1")
	require.NoError(t, err)

	c.InvalidateSource("src-A")

	// src-A rebuilt, src-B still cached.
	_, err = c.GetOrBuild(t.Context(), "src-A", "v1")
	require.NoError(t, err)
	_, err = c.GetOrBuild(t.Context(), "src-B", "v1")
	require.NoError(t, err)
	assert.Equal(t, 2, stub.tagCalls["src-A\x00v1"])
	assert.Equal(t, 1, stub.tagCalls["src-B\x00v1"])
}

func TestGetByFingerprint_Validation(t *testing.T) {
	t.Parallel()
	c := NewCache(newStub())

	_, err := c.GetByFingerprint(t.Context(), "src", "")
	assert.ErrorIs(t, err, errs.ErrMappingDescriptorMissing)

	_, err = c.GetByFingerprint(t.Context(), "", "somehash")
	assert.ErrorIs(t, err, errs.ErrMappingSourceIDRequired)
}

func TestGetByFingerprint_CachedHitNoStorage(t *testing.T) {
	t.Parallel()
	stub := newStub()
	data := descriptorSetBytes(t, "test.pkg", "Thing")
	stub.put("src", "v1", data)
	c := NewCache(stub)

	built, err := c.GetOrBuild(t.Context(), "src", "v1")
	require.NoError(t, err)

	got, err := c.GetByFingerprint(t.Context(), "src", built.Fingerprint)
	require.NoError(t, err)
	assert.Same(t, built, got)
	assert.Equal(t, 0, stub.fpCalls, "cached fingerprint must not hit storage")
}

func TestGetByFingerprint_StorageLookup(t *testing.T) {
	t.Parallel()
	stub := newStub()
	data := descriptorSetBytes(t, "test.pkg", "Thing")
	stub.put("src", "v1", data)
	c := NewCache(stub)

	// Cold cache: resolves via FindByFingerprint then builds.
	got, err := c.GetByFingerprint(t.Context(), "src", fingerprint(data))
	require.NoError(t, err)
	assert.Equal(t, "src", got.SourceID)
	assert.Equal(t, "v1", got.Tag)
	assert.Equal(t, 1, stub.fpCalls)
}

func TestGetByFingerprint_CrossSourceRefused(t *testing.T) {
	t.Parallel()
	stub := newStub()
	data := descriptorSetBytes(t, "test.pkg", "Thing")
	// Fingerprint resolves to src-A, but caller asks for src-B.
	stub.put("src-A", "v1", data)
	c := NewCache(stub)

	_, err := c.GetByFingerprint(t.Context(), "src-B", fingerprint(data))
	assert.ErrorIs(t, err, errs.ErrMappingDescriptorMissing)
}

func TestGetByFingerprint_NotFound(t *testing.T) {
	t.Parallel()
	c := NewCache(newStub())

	_, err := c.GetByFingerprint(t.Context(), "src", "unknownhash")
	assert.ErrorIs(t, err, errs.ErrMappingDescriptorMissing)
}

func TestFingerprint(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "", fingerprint(nil))
	assert.Equal(t, "", fingerprint([]byte{}))

	a := fingerprint([]byte("hello"))
	assert.NotEmpty(t, a)
	assert.Equal(t, a, fingerprint([]byte("hello")), "identical input is stable")
	assert.NotEqual(t, a, fingerprint([]byte("world")), "different input differs")
}

func TestSnapshotKey(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "src\x00v1", snapshotKey("src", "v1"))
	// NUL separator keeps a colon-bearing tag unambiguous.
	assert.NotEqual(t, snapshotKey("src", "a:b"), snapshotKey("src:a", "b"))
}

func TestCache_EvictsOldestBeyondLimit(t *testing.T) {
	t.Parallel()
	stub := newStub()
	c := NewCache(stub)
	ctx := t.Context()

	total := maxCachedSnapshots + 10
	for i := range total {
		tag := "v" + strconv.Itoa(i)
		stub.put("src", tag, descriptorSetBytes(t, "pkg", "M"+strconv.Itoa(i)))
		_, err := c.GetOrBuild(ctx, "src", tag)
		require.NoError(t, err)
	}

	c.mu.RLock()
	size := len(c.entries)
	_, oldestStillCached := c.entries[snapshotKey("src", "v0")]
	_, newestCached := c.entries[snapshotKey("src", "v"+strconv.Itoa(total-1))]
	c.mu.RUnlock()

	assert.LessOrEqual(t, size, maxCachedSnapshots, "cache must stay bounded")
	assert.False(t, oldestStillCached, "the oldest snapshot must be evicted first")
	assert.True(t, newestCached, "the most recent snapshot must survive")
}

func TestCache_EvictedEntryIsRebuiltNotLost(t *testing.T) {
	t.Parallel()
	stub := newStub()
	c := NewCache(stub)
	ctx := t.Context()

	for i := range maxCachedSnapshots + 5 {
		tag := "v" + strconv.Itoa(i)
		stub.put("src", tag, descriptorSetBytes(t, "pkg", "M"+strconv.Itoa(i)))
		_, err := c.GetOrBuild(ctx, "src", tag)
		require.NoError(t, err)
	}

	before := stub.tagCalls[snapshotKey("src", "v0")]
	snap, err := c.GetOrBuild(ctx, "src", "v0")

	require.NoError(t, err)
	require.NotNil(t, snap)
	assert.Greater(t, stub.tagCalls[snapshotKey("src", "v0")], before,
		"an evicted snapshot must be rebuilt from storage, not dropped")
}
