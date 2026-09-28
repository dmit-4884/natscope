// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"testing"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestObjectStreamSubjectsValid covers QA-066: ObjectStore(), unlike
// KeyValue(), does no sanity check that a stream is actually shaped like an
// Object Store, so DeleteObjectBucket/SealObjectBucket relies on this check
// instead of trusting the stream name alone.
func TestObjectStreamSubjectsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		bucket   string
		subjects []string
		want     bool
	}{
		{
			name:     "real object store subjects",
			bucket:   "b",
			subjects: []string{"$O.b.C.>", "$O.b.M.>"},
			want:     true,
		},
		{
			name:     "order doesn't matter",
			bucket:   "b",
			subjects: []string{"$O.b.M.>", "$O.b.C.>"},
			want:     true,
		},
		{
			name:     "plain stream subject",
			bucket:   "plainobj",
			subjects: []string{"plainobj.>"},
			want:     false,
		},
		{
			name:     "subjects for a different bucket",
			bucket:   "b",
			subjects: []string{"$O.other.C.>", "$O.other.M.>"},
			want:     false,
		},
		{
			name:     "no subjects",
			bucket:   "b",
			subjects: nil,
			want:     false,
		},
		{
			name:     "extra subject",
			bucket:   "b",
			subjects: []string{"$O.b.C.>", "$O.b.M.>", "extra.>"},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, objectStreamSubjectsValid(tt.bucket, tt.subjects))
		})
	}
}

// TestToObjectInfo_MapsLink covers QA-072: link metadata
// (jetstream.ObjectInfo.Opts.Link) sits under a nested Opts pointer the
// generic converter can't reach by field name, so it needs its own mapping.
func TestToObjectInfo_MapsLink(t *testing.T) {
	t.Parallel()

	t.Run("no link", func(t *testing.T) {
		t.Parallel()
		info := &jetstream.ObjectInfo{ObjectMeta: jetstream.ObjectMeta{Name: "plain"}}
		got := toObjectInfo(info)
		assert.Equal(t, "plain", got.Name)
		assert.Nil(t, got.Link)
	})

	t.Run("object link", func(t *testing.T) {
		t.Parallel()
		info := &jetstream.ObjectInfo{
			ObjectMeta: jetstream.ObjectMeta{
				Name: "lnk",
				Opts: &jetstream.ObjectMetaOptions{Link: &jetstream.ObjectLink{Bucket: "target-bucket", Name: "target-name"}},
			},
		}
		got := toObjectInfo(info)
		require.NotNil(t, got.Link)
		assert.Equal(t, "target-bucket", got.Link.Bucket)
		assert.Equal(t, "target-name", got.Link.Name)
	})

	t.Run("bucket link has no target name", func(t *testing.T) {
		t.Parallel()
		info := &jetstream.ObjectInfo{
			ObjectMeta: jetstream.ObjectMeta{
				Name: "blnk",
				Opts: &jetstream.ObjectMetaOptions{Link: &jetstream.ObjectLink{Bucket: "target-bucket"}},
			},
		}
		got := toObjectInfo(info)
		require.NotNil(t, got.Link)
		assert.Equal(t, "target-bucket", got.Link.Bucket)
		assert.Empty(t, got.Link.Name)
	})
}
