// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"math"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"

	"github.com/altessa-s/go-atlas/core/types/ptr"
	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"
)

// mergeConsumerUpdate is a test-only helper that replicates the inline converter call.
func (c *Client) mergeConsumerUpdate(current jetstream.ConsumerConfig, update entities.ConsumerUpdateRequest) jetstream.ConsumerConfig {
	converter.Convert(update, &current, converter.WithIgnoreNilValues())
	return current
}

// Helpers

// fullStreamConfig returns a StreamConfig with every mutable field set to a
// non-zero value and immutable fields populated for isolation checks.
func fullStreamConfig() jetstream.StreamConfig {
	return jetstream.StreamConfig{
		// Immutable fields.
		Name:               "test-stream",
		Retention:          jetstream.InterestPolicy,
		Storage:            jetstream.MemoryStorage,
		Replicas:           3,
		NoAck:              true,
		DenyDelete:         true,
		DenyPurge:          true,
		AllowRollup:        true,
		Compression:        jetstream.S2Compression,
		FirstSeq:           100,
		Sealed:             false,
		AllowMsgTTL:        true,
		AllowAtomicPublish: true,
		// Mutable fields.
		Description:          "test description",
		Subjects:             []string{"foo.>", "bar.>"},
		MaxMsgs:              1000,
		MaxBytes:             1048576,
		MaxAge:               24 * time.Hour,
		MaxMsgsPerSubject:    100,
		MaxMsgSize:           4096,
		MaxConsumers:         10,
		Duplicates:           5 * time.Minute,
		Discard:              jetstream.DiscardNew,
		DiscardNewPerSubject: true,
		AllowDirect:          true,
		MirrorDirect:         true,
		Metadata:             map[string]string{"env": "test", "version": "1"},
		Sources:              []*jetstream.StreamSource{{Name: "src1"}, {Name: "src2"}},
	}
}

// assertStreamUnchangedExcept asserts that all fields of result match expected,
// except the named fields in skip.
func assertStreamUnchangedExcept(t *testing.T, expected, result jetstream.StreamConfig, skip map[string]bool) {
	t.Helper()
	// Immutable fields (always checked).
	assert.Equal(t, expected.Name, result.Name, "Name")
	assert.Equal(t, expected.Retention, result.Retention, "Retention")
	assert.Equal(t, expected.Storage, result.Storage, "Storage")
	assert.Equal(t, expected.Replicas, result.Replicas, "Replicas")
	assert.Equal(t, expected.NoAck, result.NoAck, "NoAck")
	assert.Equal(t, expected.DenyDelete, result.DenyDelete, "DenyDelete")
	assert.Equal(t, expected.DenyPurge, result.DenyPurge, "DenyPurge")
	assert.Equal(t, expected.AllowRollup, result.AllowRollup, "AllowRollup")
	assert.Equal(t, expected.Compression, result.Compression, "Compression")
	assert.Equal(t, expected.FirstSeq, result.FirstSeq, "FirstSeq")
	assert.Equal(t, expected.Sealed, result.Sealed, "Sealed")
	assert.Equal(t, expected.AllowMsgTTL, result.AllowMsgTTL, "AllowMsgTTL")
	assert.Equal(t, expected.AllowAtomicPublish, result.AllowAtomicPublish, "AllowAtomicPublish")
	// Mutable fields (skipped if in the set).
	if !skip["Description"] {
		assert.Equal(t, expected.Description, result.Description, "Description")
	}
	if !skip["Subjects"] {
		assert.Equal(t, expected.Subjects, result.Subjects, "Subjects")
	}
	if !skip["MaxMsgs"] {
		assert.Equal(t, expected.MaxMsgs, result.MaxMsgs, "MaxMsgs")
	}
	if !skip["MaxBytes"] {
		assert.Equal(t, expected.MaxBytes, result.MaxBytes, "MaxBytes")
	}
	if !skip["MaxAge"] {
		assert.Equal(t, expected.MaxAge, result.MaxAge, "MaxAge")
	}
	if !skip["MaxMsgsPerSubject"] {
		assert.Equal(t, expected.MaxMsgsPerSubject, result.MaxMsgsPerSubject, "MaxMsgsPerSubject")
	}
	if !skip["MaxMsgSize"] {
		assert.Equal(t, expected.MaxMsgSize, result.MaxMsgSize, "MaxMsgSize")
	}
	if !skip["MaxConsumers"] {
		assert.Equal(t, expected.MaxConsumers, result.MaxConsumers, "MaxConsumers")
	}
	if !skip["Duplicates"] {
		assert.Equal(t, expected.Duplicates, result.Duplicates, "Duplicates")
	}
	if !skip["Discard"] {
		assert.Equal(t, expected.Discard, result.Discard, "Discard")
	}
	if !skip["DiscardNewPerSubject"] {
		assert.Equal(t, expected.DiscardNewPerSubject, result.DiscardNewPerSubject, "DiscardNewPerSubject")
	}
	if !skip["AllowDirect"] {
		assert.Equal(t, expected.AllowDirect, result.AllowDirect, "AllowDirect")
	}
	if !skip["MirrorDirect"] {
		assert.Equal(t, expected.MirrorDirect, result.MirrorDirect, "MirrorDirect")
	}
	if !skip["Metadata"] {
		assert.Equal(t, expected.Metadata, result.Metadata, "Metadata")
	}
	if !skip["Sources"] {
		assert.Equal(t, expected.Sources, result.Sources, "Sources")
	}
}

// fullConsumerConfig returns a ConsumerConfig with every mutable field set to
// a non-zero value and immutable fields populated for isolation checks.
func fullConsumerConfig() jetstream.ConsumerConfig {
	return jetstream.ConsumerConfig{
		// Immutable fields.
		Name:           "test-consumer",
		Durable:        "test-consumer",
		DeliverPolicy:  jetstream.DeliverLastPolicy,
		AckPolicy:      jetstream.AckAllPolicy,
		ReplayPolicy:   jetstream.ReplayOriginalPolicy,
		FilterSubject:  "foo.>",
		HeadersOnly:    true,
		Replicas:       3,
		MemoryStorage:  true,
		DeliverSubject: "deliver.test",
		DeliverGroup:   "group1",
		FlowControl:    true,
		IdleHeartbeat:  30 * time.Second,
		OptStartSeq:    42,
		// Mutable fields.
		Description:        "test consumer",
		AckWait:            30 * time.Second,
		MaxDeliver:         5,
		MaxAckPending:      100,
		MaxWaiting:         50,
		RateLimit:          10000,
		SampleFrequency:    "50",
		InactiveThreshold:  time.Hour,
		BackOff:            []time.Duration{time.Second, 5 * time.Second, 30 * time.Second},
		MaxRequestBatch:    10,
		MaxRequestMaxBytes: 2048,
		MaxRequestExpires:  time.Minute,
		Metadata:           map[string]string{"env": "test"},
	}
}

// assertConsumerUnchangedExcept asserts that all fields of result match
// expected, except the named fields in skip.
func assertConsumerUnchangedExcept(t *testing.T, expected, result jetstream.ConsumerConfig, skip map[string]bool) {
	t.Helper()
	// Immutable fields (always checked).
	assert.Equal(t, expected.Name, result.Name, "Name")
	assert.Equal(t, expected.Durable, result.Durable, "Durable")
	assert.Equal(t, expected.DeliverPolicy, result.DeliverPolicy, "DeliverPolicy")
	assert.Equal(t, expected.AckPolicy, result.AckPolicy, "AckPolicy")
	assert.Equal(t, expected.ReplayPolicy, result.ReplayPolicy, "ReplayPolicy")
	assert.Equal(t, expected.FilterSubject, result.FilterSubject, "FilterSubject")
	assert.Equal(t, expected.FilterSubjects, result.FilterSubjects, "FilterSubjects")
	assert.Equal(t, expected.HeadersOnly, result.HeadersOnly, "HeadersOnly")
	assert.Equal(t, expected.Replicas, result.Replicas, "Replicas")
	assert.Equal(t, expected.MemoryStorage, result.MemoryStorage, "MemoryStorage")
	assert.Equal(t, expected.DeliverSubject, result.DeliverSubject, "DeliverSubject")
	assert.Equal(t, expected.DeliverGroup, result.DeliverGroup, "DeliverGroup")
	assert.Equal(t, expected.FlowControl, result.FlowControl, "FlowControl")
	assert.Equal(t, expected.IdleHeartbeat, result.IdleHeartbeat, "IdleHeartbeat")
	assert.Equal(t, expected.OptStartSeq, result.OptStartSeq, "OptStartSeq")
	assert.Equal(t, expected.OptStartTime, result.OptStartTime, "OptStartTime")
	// Mutable fields (skipped if in the set).
	if !skip["Description"] {
		assert.Equal(t, expected.Description, result.Description, "Description")
	}
	if !skip["AckWait"] {
		assert.Equal(t, expected.AckWait, result.AckWait, "AckWait")
	}
	if !skip["MaxDeliver"] {
		assert.Equal(t, expected.MaxDeliver, result.MaxDeliver, "MaxDeliver")
	}
	if !skip["MaxAckPending"] {
		assert.Equal(t, expected.MaxAckPending, result.MaxAckPending, "MaxAckPending")
	}
	if !skip["MaxWaiting"] {
		assert.Equal(t, expected.MaxWaiting, result.MaxWaiting, "MaxWaiting")
	}
	if !skip["RateLimit"] {
		assert.Equal(t, expected.RateLimit, result.RateLimit, "RateLimit")
	}
	if !skip["SampleFrequency"] {
		assert.Equal(t, expected.SampleFrequency, result.SampleFrequency, "SampleFrequency")
	}
	if !skip["InactiveThreshold"] {
		assert.Equal(t, expected.InactiveThreshold, result.InactiveThreshold, "InactiveThreshold")
	}
	if !skip["BackOff"] {
		assert.Equal(t, expected.BackOff, result.BackOff, "BackOff")
	}
	if !skip["MaxRequestBatch"] {
		assert.Equal(t, expected.MaxRequestBatch, result.MaxRequestBatch, "MaxRequestBatch")
	}
	if !skip["MaxRequestMaxBytes"] {
		assert.Equal(t, expected.MaxRequestMaxBytes, result.MaxRequestMaxBytes, "MaxRequestMaxBytes")
	}
	if !skip["MaxRequestExpires"] {
		assert.Equal(t, expected.MaxRequestExpires, result.MaxRequestExpires, "MaxRequestExpires")
	}
	if !skip["Metadata"] {
		assert.Equal(t, expected.Metadata, result.Metadata, "Metadata")
	}
}

// TestApplyStreamUpdate

func TestApplyStreamUpdate(t *testing.T) {
	svc := &Client{}

	t.Run("empty update preserves all fields", func(t *testing.T) {
		current := jetstream.StreamConfig{
			Subjects:             []string{"foo"},
			Description:          "test",
			MaxMsgs:              100,
			MaxBytes:             1000,
			MaxAge:               time.Hour,
			MaxMsgsPerSubject:    10,
			MaxMsgSize:           1024,
			MaxConsumers:         5,
			Duplicates:           2 * time.Minute,
			AllowDirect:          true,
			MirrorDirect:         true,
			DiscardNewPerSubject: true,
			Discard:              jetstream.DiscardNew,
			Metadata:             map[string]string{"key": "val"},
			Sources:              []*jetstream.StreamSource{{Name: "src1"}},
		}

		result := svc.mergeStreamUpdate(current, entities.StreamUpdateRequest{})

		assert.Equal(t, []string{"foo"}, result.Subjects)
		assert.Equal(t, "test", result.Description)
		assert.Equal(t, int64(100), result.MaxMsgs)
		assert.Equal(t, int64(1000), result.MaxBytes)
		assert.Equal(t, time.Hour, result.MaxAge)
		assert.Equal(t, int64(10), result.MaxMsgsPerSubject)
		assert.Equal(t, int32(1024), result.MaxMsgSize)
		assert.Equal(t, 5, result.MaxConsumers)
		assert.Equal(t, 2*time.Minute, result.Duplicates)
		assert.True(t, result.AllowDirect)
		assert.True(t, result.MirrorDirect)
		assert.True(t, result.DiscardNewPerSubject)
		assert.Equal(t, jetstream.DiscardNew, result.Discard)
		assert.Equal(t, map[string]string{"key": "val"}, result.Metadata)
		assert.Len(t, result.Sources, 1)
		assert.Equal(t, "src1", result.Sources[0].Name)
	})

	t.Run("update only specified fields", func(t *testing.T) {
		t.Run("update description only", func(t *testing.T) {
			current := jetstream.StreamConfig{
				Description: "old",
				MaxMsgs:     100,
				AllowDirect: true,
			}
			result := svc.mergeStreamUpdate(current, entities.StreamUpdateRequest{
				Description: ptr.Wrap("new"),
			})
			assert.Equal(t, "new", result.Description)
			assert.Equal(t, int64(100), result.MaxMsgs)
			assert.True(t, result.AllowDirect)
		})

		t.Run("update MaxMsgs only", func(t *testing.T) {
			current := jetstream.StreamConfig{
				MaxMsgs:     100,
				Description: "test",
				AllowDirect: true,
			}
			result := svc.mergeStreamUpdate(current, entities.StreamUpdateRequest{
				MaxMsgs: ptr.Wrap(int64(500)),
			})
			assert.Equal(t, int64(500), result.MaxMsgs)
			assert.Equal(t, "test", result.Description)
			assert.True(t, result.AllowDirect)
		})

		t.Run("update AllowDirect only", func(t *testing.T) {
			current := jetstream.StreamConfig{
				AllowDirect: true,
				Description: "test",
				MaxMsgs:     100,
			}
			result := svc.mergeStreamUpdate(current, entities.StreamUpdateRequest{
				AllowDirect: ptr.Wrap(false),
			})
			assert.False(t, result.AllowDirect)
			assert.Equal(t, "test", result.Description)
			assert.Equal(t, int64(100), result.MaxMsgs)
		})
	})

	t.Run("can set fields to zero value", func(t *testing.T) {
		t.Run("set Description to empty string", func(t *testing.T) {
			current := jetstream.StreamConfig{Description: "old"}
			result := svc.mergeStreamUpdate(current, entities.StreamUpdateRequest{
				Description: ptr.Wrap(""),
			})
			assert.Equal(t, "", result.Description)
		})

		t.Run("set MaxMsgs to zero", func(t *testing.T) {
			current := jetstream.StreamConfig{MaxMsgs: 100}
			result := svc.mergeStreamUpdate(current, entities.StreamUpdateRequest{
				MaxMsgs: ptr.Wrap(int64(0)),
			})
			assert.Equal(t, int64(0), result.MaxMsgs)
		})

		t.Run("set MaxBytes to zero", func(t *testing.T) {
			current := jetstream.StreamConfig{MaxBytes: 1000}
			result := svc.mergeStreamUpdate(current, entities.StreamUpdateRequest{
				MaxBytes: ptr.Wrap(int64(0)),
			})
			assert.Equal(t, int64(0), result.MaxBytes)
		})

		t.Run("set MaxAge to zero", func(t *testing.T) {
			current := jetstream.StreamConfig{MaxAge: time.Hour}
			result := svc.mergeStreamUpdate(current, entities.StreamUpdateRequest{
				MaxAge: ptr.Wrap(time.Duration(0)),
			})
			assert.Equal(t, time.Duration(0), result.MaxAge)
		})

		t.Run("set AllowDirect to false", func(t *testing.T) {
			current := jetstream.StreamConfig{AllowDirect: true}
			result := svc.mergeStreamUpdate(current, entities.StreamUpdateRequest{
				AllowDirect: ptr.Wrap(false),
			})
			assert.False(t, result.AllowDirect)
		})

		t.Run("set MirrorDirect to false", func(t *testing.T) {
			current := jetstream.StreamConfig{MirrorDirect: true}
			result := svc.mergeStreamUpdate(current, entities.StreamUpdateRequest{
				MirrorDirect: ptr.Wrap(false),
			})
			assert.False(t, result.MirrorDirect)
		})

		t.Run("set DiscardNewPerSubject to false", func(t *testing.T) {
			current := jetstream.StreamConfig{DiscardNewPerSubject: true}
			result := svc.mergeStreamUpdate(current, entities.StreamUpdateRequest{
				DiscardNewPerSubject: ptr.Wrap(false),
			})
			assert.False(t, result.DiscardNewPerSubject)
		})
	})

	t.Run("boolean fields not sent are preserved", func(t *testing.T) {
		t.Run("AllowDirect=true preserved when not sent", func(t *testing.T) {
			current := jetstream.StreamConfig{AllowDirect: true}
			result := svc.mergeStreamUpdate(current, entities.StreamUpdateRequest{
				Description: ptr.Wrap("changed"),
			})
			assert.True(t, result.AllowDirect)
		})

		t.Run("MirrorDirect=true preserved when not sent", func(t *testing.T) {
			current := jetstream.StreamConfig{MirrorDirect: true}
			result := svc.mergeStreamUpdate(current, entities.StreamUpdateRequest{
				MaxMsgs: ptr.Wrap(int64(500)),
			})
			assert.True(t, result.MirrorDirect)
		})

		t.Run("DiscardNewPerSubject=true preserved when not sent", func(t *testing.T) {
			current := jetstream.StreamConfig{DiscardNewPerSubject: true}
			result := svc.mergeStreamUpdate(current, entities.StreamUpdateRequest{
				MaxBytes: ptr.Wrap(int64(2000)),
			})
			assert.True(t, result.DiscardNewPerSubject)
		})
	})

	t.Run("subjects update", func(t *testing.T) {
		t.Run("update subjects", func(t *testing.T) {
			current := jetstream.StreamConfig{Subjects: []string{"foo.>"}}
			result := svc.mergeStreamUpdate(current, entities.StreamUpdateRequest{
				Subjects: []string{"bar.>", "baz.>"},
			})
			assert.Equal(t, []string{"bar.>", "baz.>"}, result.Subjects)
		})

		t.Run("empty subjects preserved", func(t *testing.T) {
			current := jetstream.StreamConfig{Subjects: []string{"foo.>"}}
			result := svc.mergeStreamUpdate(current, entities.StreamUpdateRequest{})
			assert.Equal(t, []string{"foo.>"}, result.Subjects)
		})
	})

	t.Run("discard policy update", func(t *testing.T) {
		t.Run("set discard to new", func(t *testing.T) {
			current := jetstream.StreamConfig{Discard: jetstream.DiscardOld}
			result := svc.mergeStreamUpdate(current, entities.StreamUpdateRequest{
				Discard: ptr.Wrap(entities.DiscardNew),
			})
			assert.Equal(t, jetstream.DiscardNew, result.Discard)
		})

		t.Run("set discard to old", func(t *testing.T) {
			current := jetstream.StreamConfig{Discard: jetstream.DiscardNew}
			result := svc.mergeStreamUpdate(current, entities.StreamUpdateRequest{
				Discard: ptr.Wrap(entities.DiscardOld),
			})
			assert.Equal(t, jetstream.DiscardOld, result.Discard)
		})

		t.Run("discard nil preserved", func(t *testing.T) {
			current := jetstream.StreamConfig{Discard: jetstream.DiscardNew}
			result := svc.mergeStreamUpdate(current, entities.StreamUpdateRequest{})
			assert.Equal(t, jetstream.DiscardNew, result.Discard)
		})
	})

	t.Run("metadata update", func(t *testing.T) {
		t.Run("update metadata", func(t *testing.T) {
			current := jetstream.StreamConfig{Metadata: map[string]string{"key": "old"}}
			result := svc.mergeStreamUpdate(current, entities.StreamUpdateRequest{
				Metadata: map[string]string{"key": "new", "key2": "val2"},
			})
			assert.Equal(t, map[string]string{"key": "new", "key2": "val2"}, result.Metadata)
		})

		t.Run("nil metadata preserved", func(t *testing.T) {
			current := jetstream.StreamConfig{Metadata: map[string]string{"key": "val"}}
			result := svc.mergeStreamUpdate(current, entities.StreamUpdateRequest{})
			assert.Equal(t, map[string]string{"key": "val"}, result.Metadata)
		})
	})

	t.Run("sources append", func(t *testing.T) {
		t.Run("add sources", func(t *testing.T) {
			current := jetstream.StreamConfig{
				Sources: []*jetstream.StreamSource{{Name: "src1"}},
			}
			result := svc.mergeStreamUpdate(current, entities.StreamUpdateRequest{
				Sources: []*entities.StreamSource{{Name: "src2"}},
			})
			assert.Len(t, result.Sources, 2)
			assert.Equal(t, "src1", result.Sources[0].Name)
			assert.Equal(t, "src2", result.Sources[1].Name)
		})

		t.Run("empty sources preserved", func(t *testing.T) {
			current := jetstream.StreamConfig{
				Sources: []*jetstream.StreamSource{{Name: "src1"}},
			}
			result := svc.mergeStreamUpdate(current, entities.StreamUpdateRequest{})
			assert.Len(t, result.Sources, 1)
			assert.Equal(t, "src1", result.Sources[0].Name)
		})
	})

	t.Run("all numeric fields", func(t *testing.T) {
		t.Run("MaxMsgsPerSubject", func(t *testing.T) {
			current := jetstream.StreamConfig{MaxMsgsPerSubject: 10}
			result := svc.mergeStreamUpdate(current, entities.StreamUpdateRequest{
				MaxMsgsPerSubject: ptr.Wrap(int64(20)),
			})
			assert.Equal(t, int64(20), result.MaxMsgsPerSubject)
		})

		t.Run("MaxMsgSize", func(t *testing.T) {
			current := jetstream.StreamConfig{MaxMsgSize: 1024}
			result := svc.mergeStreamUpdate(current, entities.StreamUpdateRequest{
				MaxMsgSize: ptr.Wrap(int32(2048)),
			})
			assert.Equal(t, int32(2048), result.MaxMsgSize)
		})

		t.Run("MaxConsumers", func(t *testing.T) {
			current := jetstream.StreamConfig{MaxConsumers: 5}
			result := svc.mergeStreamUpdate(current, entities.StreamUpdateRequest{
				MaxConsumers: ptr.Wrap(10),
			})
			assert.Equal(t, 10, result.MaxConsumers)
		})

		t.Run("Duplicates", func(t *testing.T) {
			current := jetstream.StreamConfig{Duplicates: 2 * time.Minute}
			result := svc.mergeStreamUpdate(current, entities.StreamUpdateRequest{
				Duplicates: ptr.Wrap(5 * time.Minute),
			})
			assert.Equal(t, 5*time.Minute, result.Duplicates)
		})
	})

	t.Run("full update all fields at once", func(t *testing.T) {
		current := jetstream.StreamConfig{
			Subjects:             []string{"old.>"},
			Description:          "old",
			MaxMsgs:              100,
			MaxBytes:             1000,
			MaxAge:               time.Hour,
			MaxMsgsPerSubject:    10,
			MaxMsgSize:           1024,
			MaxConsumers:         5,
			Duplicates:           2 * time.Minute,
			Discard:              jetstream.DiscardOld,
			DiscardNewPerSubject: false,
			AllowDirect:          false,
			MirrorDirect:         false,
			Metadata:             map[string]string{"old": "val"},
		}

		result := svc.mergeStreamUpdate(current, entities.StreamUpdateRequest{
			Subjects:             []string{"new.>"},
			Description:          ptr.Wrap("new"),
			MaxMsgs:              ptr.Wrap(int64(500)),
			MaxBytes:             ptr.Wrap(int64(5000)),
			MaxAge:               ptr.Wrap(2 * time.Hour),
			MaxMsgsPerSubject:    ptr.Wrap(int64(50)),
			MaxMsgSize:           ptr.Wrap(int32(2048)),
			MaxConsumers:         ptr.Wrap(10),
			Duplicates:           ptr.Wrap(5 * time.Minute),
			Discard:              ptr.Wrap(entities.DiscardNew),
			DiscardNewPerSubject: ptr.Wrap(true),
			AllowDirect:          ptr.Wrap(true),
			MirrorDirect:         ptr.Wrap(true),
			Metadata:             map[string]string{"new": "val"},
			Sources:              []*entities.StreamSource{{Name: "src1"}},
		})

		assert.Equal(t, []string{"new.>"}, result.Subjects)
		assert.Equal(t, "new", result.Description)
		assert.Equal(t, int64(500), result.MaxMsgs)
		assert.Equal(t, int64(5000), result.MaxBytes)
		assert.Equal(t, 2*time.Hour, result.MaxAge)
		assert.Equal(t, int64(50), result.MaxMsgsPerSubject)
		assert.Equal(t, int32(2048), result.MaxMsgSize)
		assert.Equal(t, 10, result.MaxConsumers)
		assert.Equal(t, 5*time.Minute, result.Duplicates)
		assert.Equal(t, jetstream.DiscardNew, result.Discard)
		assert.True(t, result.DiscardNewPerSubject)
		assert.True(t, result.AllowDirect)
		assert.True(t, result.MirrorDirect)
		assert.Equal(t, map[string]string{"new": "val"}, result.Metadata)
		assert.Len(t, result.Sources, 1)
		assert.Equal(t, "src1", result.Sources[0].Name)
	})

	// Stream field isolation tests

	t.Run("field isolation", func(t *testing.T) {
		t.Run("update Description only", func(t *testing.T) {
			base := fullStreamConfig()
			result := svc.mergeStreamUpdate(base, entities.StreamUpdateRequest{
				Description: ptr.Wrap("changed"),
			})
			assert.Equal(t, "changed", result.Description)
			assertStreamUnchangedExcept(t, base, result, map[string]bool{"Description": true})
		})

		t.Run("update MaxMsgs only", func(t *testing.T) {
			base := fullStreamConfig()
			result := svc.mergeStreamUpdate(base, entities.StreamUpdateRequest{
				MaxMsgs: ptr.Wrap(int64(9999)),
			})
			assert.Equal(t, int64(9999), result.MaxMsgs)
			assertStreamUnchangedExcept(t, base, result, map[string]bool{"MaxMsgs": true})
		})

		t.Run("update MaxBytes only", func(t *testing.T) {
			base := fullStreamConfig()
			result := svc.mergeStreamUpdate(base, entities.StreamUpdateRequest{
				MaxBytes: ptr.Wrap(int64(999999)),
			})
			assert.Equal(t, int64(999999), result.MaxBytes)
			assertStreamUnchangedExcept(t, base, result, map[string]bool{"MaxBytes": true})
		})

		t.Run("update MaxAge only", func(t *testing.T) {
			base := fullStreamConfig()
			result := svc.mergeStreamUpdate(base, entities.StreamUpdateRequest{
				MaxAge: ptr.Wrap(48 * time.Hour),
			})
			assert.Equal(t, 48*time.Hour, result.MaxAge)
			assertStreamUnchangedExcept(t, base, result, map[string]bool{"MaxAge": true})
		})

		t.Run("update MaxMsgsPerSubject only", func(t *testing.T) {
			base := fullStreamConfig()
			result := svc.mergeStreamUpdate(base, entities.StreamUpdateRequest{
				MaxMsgsPerSubject: ptr.Wrap(int64(500)),
			})
			assert.Equal(t, int64(500), result.MaxMsgsPerSubject)
			assertStreamUnchangedExcept(t, base, result, map[string]bool{"MaxMsgsPerSubject": true})
		})

		t.Run("update MaxMsgSize only", func(t *testing.T) {
			base := fullStreamConfig()
			result := svc.mergeStreamUpdate(base, entities.StreamUpdateRequest{
				MaxMsgSize: ptr.Wrap(int32(8192)),
			})
			assert.Equal(t, int32(8192), result.MaxMsgSize)
			assertStreamUnchangedExcept(t, base, result, map[string]bool{"MaxMsgSize": true})
		})

		t.Run("update MaxConsumers only", func(t *testing.T) {
			base := fullStreamConfig()
			result := svc.mergeStreamUpdate(base, entities.StreamUpdateRequest{
				MaxConsumers: ptr.Wrap(20),
			})
			assert.Equal(t, 20, result.MaxConsumers)
			assertStreamUnchangedExcept(t, base, result, map[string]bool{"MaxConsumers": true})
		})

		t.Run("update Duplicates only", func(t *testing.T) {
			base := fullStreamConfig()
			result := svc.mergeStreamUpdate(base, entities.StreamUpdateRequest{
				Duplicates: ptr.Wrap(10 * time.Minute),
			})
			assert.Equal(t, 10*time.Minute, result.Duplicates)
			assertStreamUnchangedExcept(t, base, result, map[string]bool{"Duplicates": true})
		})

		t.Run("update AllowDirect only", func(t *testing.T) {
			base := fullStreamConfig()
			result := svc.mergeStreamUpdate(base, entities.StreamUpdateRequest{
				AllowDirect: ptr.Wrap(false),
			})
			assert.False(t, result.AllowDirect)
			assertStreamUnchangedExcept(t, base, result, map[string]bool{"AllowDirect": true})
		})

		t.Run("update MirrorDirect only", func(t *testing.T) {
			base := fullStreamConfig()
			result := svc.mergeStreamUpdate(base, entities.StreamUpdateRequest{
				MirrorDirect: ptr.Wrap(false),
			})
			assert.False(t, result.MirrorDirect)
			assertStreamUnchangedExcept(t, base, result, map[string]bool{"MirrorDirect": true})
		})

		t.Run("update Discard only", func(t *testing.T) {
			base := fullStreamConfig()
			result := svc.mergeStreamUpdate(base, entities.StreamUpdateRequest{
				Discard: ptr.Wrap(entities.DiscardOld),
			})
			assert.Equal(t, jetstream.DiscardOld, result.Discard)
			assertStreamUnchangedExcept(t, base, result, map[string]bool{"Discard": true})
		})

		t.Run("update DiscardNewPerSubject only", func(t *testing.T) {
			base := fullStreamConfig()
			result := svc.mergeStreamUpdate(base, entities.StreamUpdateRequest{
				DiscardNewPerSubject: ptr.Wrap(false),
			})
			assert.False(t, result.DiscardNewPerSubject)
			assertStreamUnchangedExcept(t, base, result, map[string]bool{"DiscardNewPerSubject": true})
		})

		t.Run("update Subjects only", func(t *testing.T) {
			base := fullStreamConfig()
			result := svc.mergeStreamUpdate(base, entities.StreamUpdateRequest{
				Subjects: []string{"new.>"},
			})
			assert.Equal(t, []string{"new.>"}, result.Subjects)
			assertStreamUnchangedExcept(t, base, result, map[string]bool{"Subjects": true})
		})

		t.Run("update Metadata only", func(t *testing.T) {
			base := fullStreamConfig()
			result := svc.mergeStreamUpdate(base, entities.StreamUpdateRequest{
				Metadata: map[string]string{"new": "value"},
			})
			assert.Equal(t, map[string]string{"new": "value"}, result.Metadata)
			assertStreamUnchangedExcept(t, base, result, map[string]bool{"Metadata": true})
		})
	})

	// Immutable fields preservation

	t.Run("immutable fields preserved after full update", func(t *testing.T) {
		base := fullStreamConfig()
		result := svc.mergeStreamUpdate(base, entities.StreamUpdateRequest{
			Description:          ptr.Wrap("changed"),
			MaxMsgs:              ptr.Wrap(int64(1)),
			MaxBytes:             ptr.Wrap(int64(1)),
			MaxAge:               ptr.Wrap(time.Second),
			MaxMsgsPerSubject:    ptr.Wrap(int64(1)),
			MaxMsgSize:           ptr.Wrap(int32(1)),
			MaxConsumers:         ptr.Wrap(1),
			Duplicates:           ptr.Wrap(time.Second),
			AllowDirect:          ptr.Wrap(false),
			MirrorDirect:         ptr.Wrap(false),
			Discard:              ptr.Wrap(entities.DiscardOld),
			DiscardNewPerSubject: ptr.Wrap(false),
			Subjects:             []string{"x"},
			Metadata:             map[string]string{"x": "y"},
			Sources:              []*entities.StreamSource{{Name: "new-src"}},
		})

		assert.Equal(t, "test-stream", result.Name)
		assert.Equal(t, jetstream.InterestPolicy, result.Retention)
		assert.Equal(t, jetstream.MemoryStorage, result.Storage)
		assert.Equal(t, 3, result.Replicas)
		assert.True(t, result.NoAck)
		assert.True(t, result.DenyDelete)
		assert.True(t, result.DenyPurge)
		assert.True(t, result.AllowRollup)
		assert.Equal(t, jetstream.S2Compression, result.Compression)
		assert.Equal(t, uint64(100), result.FirstSeq)
		assert.False(t, result.Sealed)
		assert.True(t, result.AllowMsgTTL)
		assert.True(t, result.AllowAtomicPublish)
	})

	// Sequential updates

	t.Run("sequential updates accumulate", func(t *testing.T) {
		base := fullStreamConfig()

		// First: update Description.
		r1 := svc.mergeStreamUpdate(base, entities.StreamUpdateRequest{
			Description: ptr.Wrap("step1"),
		})
		// Second: update MaxMsgs.
		r2 := svc.mergeStreamUpdate(r1, entities.StreamUpdateRequest{
			MaxMsgs: ptr.Wrap(int64(42)),
		})
		// Third: update AllowDirect.
		r3 := svc.mergeStreamUpdate(r2, entities.StreamUpdateRequest{
			AllowDirect: ptr.Wrap(false),
		})

		assert.Equal(t, "step1", r3.Description)
		assert.Equal(t, int64(42), r3.MaxMsgs)
		assert.False(t, r3.AllowDirect)
		// All other mutable fields unchanged from base.
		assertStreamUnchangedExcept(t, base, r3, map[string]bool{
			"Description": true, "MaxMsgs": true, "AllowDirect": true,
		})
	})

	t.Run("sequential update same field twice", func(t *testing.T) {
		base := fullStreamConfig()

		r1 := svc.mergeStreamUpdate(base, entities.StreamUpdateRequest{
			Description: ptr.Wrap("first"),
		})
		r2 := svc.mergeStreamUpdate(r1, entities.StreamUpdateRequest{
			Description: ptr.Wrap("second"),
		})

		assert.Equal(t, "second", r2.Description)
		assertStreamUnchangedExcept(t, base, r2, map[string]bool{"Description": true})
	})

	// Sources append-only semantics

	t.Run("sources append-only semantics", func(t *testing.T) {
		t.Run("appends to existing sources", func(t *testing.T) {
			current := jetstream.StreamConfig{
				Sources: []*jetstream.StreamSource{{Name: "existing1"}, {Name: "existing2"}},
			}
			result := svc.mergeStreamUpdate(current, entities.StreamUpdateRequest{
				Sources: []*entities.StreamSource{{Name: "new1"}, {Name: "new2"}},
			})
			assert.Len(t, result.Sources, 4)
			assert.Equal(t, "existing1", result.Sources[0].Name)
			assert.Equal(t, "existing2", result.Sources[1].Name)
			assert.Equal(t, "new1", result.Sources[2].Name)
			assert.Equal(t, "new2", result.Sources[3].Name)
		})

		t.Run("duplicate source name still appends", func(t *testing.T) {
			current := jetstream.StreamConfig{
				Sources: []*jetstream.StreamSource{{Name: "src1"}},
			}
			result := svc.mergeStreamUpdate(current, entities.StreamUpdateRequest{
				Sources: []*entities.StreamSource{{Name: "src1"}},
			})
			assert.Len(t, result.Sources, 2)
			assert.Equal(t, "src1", result.Sources[0].Name)
			assert.Equal(t, "src1", result.Sources[1].Name)
		})

		t.Run("sources with FilterSubject and SubjectTransforms preserved", func(t *testing.T) {
			current := jetstream.StreamConfig{
				Sources: []*jetstream.StreamSource{{Name: "src1"}},
			}
			result := svc.mergeStreamUpdate(current, entities.StreamUpdateRequest{
				Sources: []*entities.StreamSource{
					{
						Name:          "src2",
						FilterSubject: "orders.>",
						SubjectTransforms: []*entities.SubjectTransformConfig{
							{Source: "orders.>", Destination: "archive.orders.>"},
						},
					},
				},
			})
			assert.Len(t, result.Sources, 2)
			assert.Equal(t, "src2", result.Sources[1].Name)
			assert.Equal(t, "orders.>", result.Sources[1].FilterSubject)
			assert.Len(t, result.Sources[1].SubjectTransforms, 1)
			assert.Equal(t, "orders.>", result.Sources[1].SubjectTransforms[0].Source)
			assert.Equal(t, "archive.orders.>", result.Sources[1].SubjectTransforms[0].Destination)
		})
	})

	// Metadata replace semantics

	t.Run("metadata replace semantics", func(t *testing.T) {
		t.Run("replaces entire map", func(t *testing.T) {
			current := jetstream.StreamConfig{
				Metadata: map[string]string{"a": "1", "b": "2", "c": "3"},
			}
			result := svc.mergeStreamUpdate(current, entities.StreamUpdateRequest{
				Metadata: map[string]string{"x": "10"},
			})
			assert.Equal(t, map[string]string{"x": "10"}, result.Metadata)
		})

		t.Run("empty map replaces and clears", func(t *testing.T) {
			current := jetstream.StreamConfig{
				Metadata: map[string]string{"a": "1", "b": "2"},
			}
			result := svc.mergeStreamUpdate(current, entities.StreamUpdateRequest{
				Metadata: map[string]string{},
			})
			assert.Equal(t, map[string]string{}, result.Metadata)
		})
	})

	// Edge cases

	t.Run("edge cases", func(t *testing.T) {
		t.Run("all nil pointers and empty slices change nothing", func(t *testing.T) {
			base := fullStreamConfig()
			result := svc.mergeStreamUpdate(base, entities.StreamUpdateRequest{
				// All pointer fields nil, all slices empty, map nil.
			})
			assertStreamUnchangedExcept(t, base, result, nil)
		})

		t.Run("subjects can be set to single item from multi", func(t *testing.T) {
			current := jetstream.StreamConfig{
				Subjects: []string{"foo.>", "bar.>", "baz.>"},
			}
			result := svc.mergeStreamUpdate(current, entities.StreamUpdateRequest{
				Subjects: []string{"only.>"},
			})
			assert.Equal(t, []string{"only.>"}, result.Subjects)
		})

		t.Run("MaxAge set to very large value", func(t *testing.T) {
			base := fullStreamConfig()
			largeAge := time.Duration(math.MaxInt64)
			result := svc.mergeStreamUpdate(base, entities.StreamUpdateRequest{
				MaxAge: &largeAge,
			})
			assert.Equal(t, time.Duration(math.MaxInt64), result.MaxAge)
			assertStreamUnchangedExcept(t, base, result, map[string]bool{"MaxAge": true})
		})
	})
}

// TestApplyConsumerUpdate

func TestApplyConsumerUpdate(t *testing.T) {
	svc := &Client{}

	t.Run("empty update preserves all fields", func(t *testing.T) {
		current := jetstream.ConsumerConfig{
			Description:        "test",
			AckWait:            30 * time.Second,
			MaxDeliver:         5,
			MaxAckPending:      100,
			MaxWaiting:         50,
			RateLimit:          1000,
			SampleFrequency:    "100",
			InactiveThreshold:  time.Hour,
			BackOff:            []time.Duration{time.Second, 5 * time.Second, 30 * time.Second},
			MaxRequestBatch:    10,
			MaxRequestMaxBytes: 1024,
			MaxRequestExpires:  time.Minute,
			Metadata:           map[string]string{"k": "v"},
		}

		result := svc.mergeConsumerUpdate(current, entities.ConsumerUpdateRequest{})

		assert.Equal(t, "test", result.Description)
		assert.Equal(t, 30*time.Second, result.AckWait)
		assert.Equal(t, 5, result.MaxDeliver)
		assert.Equal(t, 100, result.MaxAckPending)
		assert.Equal(t, 50, result.MaxWaiting)
		assert.Equal(t, uint64(1000), result.RateLimit)
		assert.Equal(t, "100", result.SampleFrequency)
		assert.Equal(t, time.Hour, result.InactiveThreshold)
		assert.Equal(t, []time.Duration{time.Second, 5 * time.Second, 30 * time.Second}, result.BackOff)
		assert.Equal(t, 10, result.MaxRequestBatch)
		assert.Equal(t, 1024, result.MaxRequestMaxBytes)
		assert.Equal(t, time.Minute, result.MaxRequestExpires)
		assert.Equal(t, map[string]string{"k": "v"}, result.Metadata)
	})

	t.Run("update only specified fields", func(t *testing.T) {
		t.Run("update description only", func(t *testing.T) {
			current := jetstream.ConsumerConfig{
				Description:   "old",
				MaxDeliver:    5,
				MaxAckPending: 100,
			}
			result := svc.mergeConsumerUpdate(current, entities.ConsumerUpdateRequest{
				Description: ptr.Wrap("new"),
			})
			assert.Equal(t, "new", result.Description)
			assert.Equal(t, 5, result.MaxDeliver)
			assert.Equal(t, 100, result.MaxAckPending)
		})

		t.Run("update AckWait only", func(t *testing.T) {
			current := jetstream.ConsumerConfig{
				AckWait:     30 * time.Second,
				Description: "test",
			}
			result := svc.mergeConsumerUpdate(current, entities.ConsumerUpdateRequest{
				AckWait: ptr.Wrap(60 * time.Second),
			})
			assert.Equal(t, 60*time.Second, result.AckWait)
			assert.Equal(t, "test", result.Description)
		})

		t.Run("update MaxDeliver only", func(t *testing.T) {
			current := jetstream.ConsumerConfig{
				MaxDeliver:    5,
				MaxAckPending: 100,
			}
			result := svc.mergeConsumerUpdate(current, entities.ConsumerUpdateRequest{
				MaxDeliver: ptr.Wrap(10),
			})
			assert.Equal(t, 10, result.MaxDeliver)
			assert.Equal(t, 100, result.MaxAckPending)
		})
	})

	t.Run("can set fields to zero value", func(t *testing.T) {
		t.Run("set Description to empty", func(t *testing.T) {
			current := jetstream.ConsumerConfig{Description: "old"}
			result := svc.mergeConsumerUpdate(current, entities.ConsumerUpdateRequest{
				Description: ptr.Wrap(""),
			})
			assert.Equal(t, "", result.Description)
		})

		t.Run("set AckWait to zero", func(t *testing.T) {
			current := jetstream.ConsumerConfig{AckWait: 30 * time.Second}
			result := svc.mergeConsumerUpdate(current, entities.ConsumerUpdateRequest{
				AckWait: ptr.Wrap(time.Duration(0)),
			})
			assert.Equal(t, time.Duration(0), result.AckWait)
		})

		t.Run("set MaxDeliver to zero", func(t *testing.T) {
			current := jetstream.ConsumerConfig{MaxDeliver: 5}
			result := svc.mergeConsumerUpdate(current, entities.ConsumerUpdateRequest{
				MaxDeliver: ptr.Wrap(0),
			})
			assert.Equal(t, 0, result.MaxDeliver)
		})

		t.Run("set MaxAckPending to zero", func(t *testing.T) {
			current := jetstream.ConsumerConfig{MaxAckPending: 100}
			result := svc.mergeConsumerUpdate(current, entities.ConsumerUpdateRequest{
				MaxAckPending: ptr.Wrap(0),
			})
			assert.Equal(t, 0, result.MaxAckPending)
		})

		t.Run("set MaxWaiting to zero", func(t *testing.T) {
			current := jetstream.ConsumerConfig{MaxWaiting: 50}
			result := svc.mergeConsumerUpdate(current, entities.ConsumerUpdateRequest{
				MaxWaiting: ptr.Wrap(0),
			})
			assert.Equal(t, 0, result.MaxWaiting)
		})

		t.Run("set RateLimit to zero", func(t *testing.T) {
			current := jetstream.ConsumerConfig{RateLimit: 1000}
			result := svc.mergeConsumerUpdate(current, entities.ConsumerUpdateRequest{
				RateLimit: ptr.Wrap(uint64(0)),
			})
			assert.Equal(t, uint64(0), result.RateLimit)
		})

		t.Run("set SampleFrequency to empty", func(t *testing.T) {
			current := jetstream.ConsumerConfig{SampleFrequency: "100"}
			result := svc.mergeConsumerUpdate(current, entities.ConsumerUpdateRequest{
				SampleFrequency: ptr.Wrap(""),
			})
			assert.Equal(t, "", result.SampleFrequency)
		})

		t.Run("set InactiveThreshold to zero", func(t *testing.T) {
			current := jetstream.ConsumerConfig{InactiveThreshold: time.Hour}
			result := svc.mergeConsumerUpdate(current, entities.ConsumerUpdateRequest{
				InactiveThreshold: ptr.Wrap(time.Duration(0)),
			})
			assert.Equal(t, time.Duration(0), result.InactiveThreshold)
		})

		t.Run("set MaxRequestBatch to zero", func(t *testing.T) {
			current := jetstream.ConsumerConfig{MaxRequestBatch: 10}
			result := svc.mergeConsumerUpdate(current, entities.ConsumerUpdateRequest{
				MaxRequestBatch: ptr.Wrap(0),
			})
			assert.Equal(t, 0, result.MaxRequestBatch)
		})

		t.Run("set MaxRequestMaxBytes to zero", func(t *testing.T) {
			current := jetstream.ConsumerConfig{MaxRequestMaxBytes: 1024}
			result := svc.mergeConsumerUpdate(current, entities.ConsumerUpdateRequest{
				MaxRequestMaxBytes: ptr.Wrap(int64(0)),
			})
			assert.Equal(t, 0, result.MaxRequestMaxBytes)
		})

		t.Run("set MaxRequestExpires to zero", func(t *testing.T) {
			current := jetstream.ConsumerConfig{MaxRequestExpires: time.Minute}
			result := svc.mergeConsumerUpdate(current, entities.ConsumerUpdateRequest{
				MaxRequestExpires: ptr.Wrap(time.Duration(0)),
			})
			assert.Equal(t, time.Duration(0), result.MaxRequestExpires)
		})
	})

	t.Run("backoff update", func(t *testing.T) {
		t.Run("update backoff", func(t *testing.T) {
			current := jetstream.ConsumerConfig{
				BackOff: []time.Duration{time.Second, 5 * time.Second},
			}
			result := svc.mergeConsumerUpdate(current, entities.ConsumerUpdateRequest{
				BackOff: []time.Duration{2 * time.Second, 10 * time.Second, 30 * time.Second},
			})
			assert.Equal(t, []time.Duration{2 * time.Second, 10 * time.Second, 30 * time.Second}, result.BackOff)
		})

		t.Run("empty backoff preserved", func(t *testing.T) {
			current := jetstream.ConsumerConfig{
				BackOff: []time.Duration{time.Second, 5 * time.Second},
			}
			result := svc.mergeConsumerUpdate(current, entities.ConsumerUpdateRequest{})
			assert.Equal(t, []time.Duration{time.Second, 5 * time.Second}, result.BackOff)
		})
	})

	t.Run("metadata update", func(t *testing.T) {
		t.Run("update metadata", func(t *testing.T) {
			current := jetstream.ConsumerConfig{Metadata: map[string]string{"k": "old"}}
			result := svc.mergeConsumerUpdate(current, entities.ConsumerUpdateRequest{
				Metadata: map[string]string{"k": "new"},
			})
			assert.Equal(t, map[string]string{"k": "new"}, result.Metadata)
		})

		t.Run("nil metadata preserved", func(t *testing.T) {
			current := jetstream.ConsumerConfig{Metadata: map[string]string{"k": "v"}}
			result := svc.mergeConsumerUpdate(current, entities.ConsumerUpdateRequest{})
			assert.Equal(t, map[string]string{"k": "v"}, result.Metadata)
		})
	})

	t.Run("full update all fields", func(t *testing.T) {
		current := jetstream.ConsumerConfig{
			Description:        "old",
			AckWait:            30 * time.Second,
			MaxDeliver:         5,
			MaxAckPending:      100,
			MaxWaiting:         50,
			RateLimit:          1000,
			SampleFrequency:    "50",
			InactiveThreshold:  time.Hour,
			BackOff:            []time.Duration{time.Second},
			MaxRequestBatch:    10,
			MaxRequestMaxBytes: 1024,
			MaxRequestExpires:  time.Minute,
			Metadata:           map[string]string{"old": "val"},
		}

		result := svc.mergeConsumerUpdate(current, entities.ConsumerUpdateRequest{
			Description:        ptr.Wrap("new"),
			AckWait:            ptr.Wrap(60 * time.Second),
			MaxDeliver:         ptr.Wrap(10),
			MaxAckPending:      ptr.Wrap(200),
			MaxWaiting:         ptr.Wrap(100),
			RateLimit:          ptr.Wrap(uint64(2000)),
			SampleFrequency:    ptr.Wrap("100"),
			InactiveThreshold:  ptr.Wrap(2 * time.Hour),
			BackOff:            []time.Duration{2 * time.Second, 10 * time.Second},
			MaxRequestBatch:    ptr.Wrap(20),
			MaxRequestMaxBytes: ptr.Wrap(int64(2048)),
			MaxRequestExpires:  ptr.Wrap(2 * time.Minute),
			Metadata:           map[string]string{"new": "val"},
		})

		assert.Equal(t, "new", result.Description)
		assert.Equal(t, 60*time.Second, result.AckWait)
		assert.Equal(t, 10, result.MaxDeliver)
		assert.Equal(t, 200, result.MaxAckPending)
		assert.Equal(t, 100, result.MaxWaiting)
		assert.Equal(t, uint64(2000), result.RateLimit)
		assert.Equal(t, "100", result.SampleFrequency)
		assert.Equal(t, 2*time.Hour, result.InactiveThreshold)
		assert.Equal(t, []time.Duration{2 * time.Second, 10 * time.Second}, result.BackOff)
		assert.Equal(t, 20, result.MaxRequestBatch)
		assert.Equal(t, 2048, result.MaxRequestMaxBytes)
		assert.Equal(t, 2*time.Minute, result.MaxRequestExpires)
		assert.Equal(t, map[string]string{"new": "val"}, result.Metadata)
	})

	// Consumer field isolation tests

	t.Run("field isolation", func(t *testing.T) {
		t.Run("update Description only", func(t *testing.T) {
			base := fullConsumerConfig()
			result := svc.mergeConsumerUpdate(base, entities.ConsumerUpdateRequest{
				Description: ptr.Wrap("changed"),
			})
			assert.Equal(t, "changed", result.Description)
			assertConsumerUnchangedExcept(t, base, result, map[string]bool{"Description": true})
		})

		t.Run("update AckWait only", func(t *testing.T) {
			base := fullConsumerConfig()
			result := svc.mergeConsumerUpdate(base, entities.ConsumerUpdateRequest{
				AckWait: ptr.Wrap(90 * time.Second),
			})
			assert.Equal(t, 90*time.Second, result.AckWait)
			assertConsumerUnchangedExcept(t, base, result, map[string]bool{"AckWait": true})
		})

		t.Run("update MaxDeliver only", func(t *testing.T) {
			base := fullConsumerConfig()
			result := svc.mergeConsumerUpdate(base, entities.ConsumerUpdateRequest{
				MaxDeliver: ptr.Wrap(20),
			})
			assert.Equal(t, 20, result.MaxDeliver)
			assertConsumerUnchangedExcept(t, base, result, map[string]bool{"MaxDeliver": true})
		})

		t.Run("update MaxAckPending only", func(t *testing.T) {
			base := fullConsumerConfig()
			result := svc.mergeConsumerUpdate(base, entities.ConsumerUpdateRequest{
				MaxAckPending: ptr.Wrap(999),
			})
			assert.Equal(t, 999, result.MaxAckPending)
			assertConsumerUnchangedExcept(t, base, result, map[string]bool{"MaxAckPending": true})
		})

		t.Run("update MaxWaiting only", func(t *testing.T) {
			base := fullConsumerConfig()
			result := svc.mergeConsumerUpdate(base, entities.ConsumerUpdateRequest{
				MaxWaiting: ptr.Wrap(200),
			})
			assert.Equal(t, 200, result.MaxWaiting)
			assertConsumerUnchangedExcept(t, base, result, map[string]bool{"MaxWaiting": true})
		})

		t.Run("update RateLimit only", func(t *testing.T) {
			base := fullConsumerConfig()
			result := svc.mergeConsumerUpdate(base, entities.ConsumerUpdateRequest{
				RateLimit: ptr.Wrap(uint64(50000)),
			})
			assert.Equal(t, uint64(50000), result.RateLimit)
			assertConsumerUnchangedExcept(t, base, result, map[string]bool{"RateLimit": true})
		})

		t.Run("update SampleFrequency only", func(t *testing.T) {
			base := fullConsumerConfig()
			result := svc.mergeConsumerUpdate(base, entities.ConsumerUpdateRequest{
				SampleFrequency: ptr.Wrap("100"),
			})
			assert.Equal(t, "100", result.SampleFrequency)
			assertConsumerUnchangedExcept(t, base, result, map[string]bool{"SampleFrequency": true})
		})

		t.Run("update InactiveThreshold only", func(t *testing.T) {
			base := fullConsumerConfig()
			result := svc.mergeConsumerUpdate(base, entities.ConsumerUpdateRequest{
				InactiveThreshold: ptr.Wrap(2 * time.Hour),
			})
			assert.Equal(t, 2*time.Hour, result.InactiveThreshold)
			assertConsumerUnchangedExcept(t, base, result, map[string]bool{"InactiveThreshold": true})
		})

		t.Run("update BackOff only", func(t *testing.T) {
			base := fullConsumerConfig()
			result := svc.mergeConsumerUpdate(base, entities.ConsumerUpdateRequest{
				BackOff: []time.Duration{10 * time.Second, 30 * time.Second},
			})
			assert.Equal(t, []time.Duration{10 * time.Second, 30 * time.Second}, result.BackOff)
			assertConsumerUnchangedExcept(t, base, result, map[string]bool{"BackOff": true})
		})

		t.Run("update MaxRequestBatch only", func(t *testing.T) {
			base := fullConsumerConfig()
			result := svc.mergeConsumerUpdate(base, entities.ConsumerUpdateRequest{
				MaxRequestBatch: ptr.Wrap(50),
			})
			assert.Equal(t, 50, result.MaxRequestBatch)
			assertConsumerUnchangedExcept(t, base, result, map[string]bool{"MaxRequestBatch": true})
		})

		t.Run("update MaxRequestMaxBytes only", func(t *testing.T) {
			base := fullConsumerConfig()
			result := svc.mergeConsumerUpdate(base, entities.ConsumerUpdateRequest{
				MaxRequestMaxBytes: ptr.Wrap(int64(8192)),
			})
			assert.Equal(t, 8192, result.MaxRequestMaxBytes)
			assertConsumerUnchangedExcept(t, base, result, map[string]bool{"MaxRequestMaxBytes": true})
		})

		t.Run("update MaxRequestExpires only", func(t *testing.T) {
			base := fullConsumerConfig()
			result := svc.mergeConsumerUpdate(base, entities.ConsumerUpdateRequest{
				MaxRequestExpires: ptr.Wrap(5 * time.Minute),
			})
			assert.Equal(t, 5*time.Minute, result.MaxRequestExpires)
			assertConsumerUnchangedExcept(t, base, result, map[string]bool{"MaxRequestExpires": true})
		})
	})

	// Immutable fields preservation

	t.Run("immutable fields preserved after full update", func(t *testing.T) {
		base := fullConsumerConfig()
		result := svc.mergeConsumerUpdate(base, entities.ConsumerUpdateRequest{
			Description:        ptr.Wrap("changed"),
			AckWait:            ptr.Wrap(time.Second),
			MaxDeliver:         ptr.Wrap(1),
			MaxAckPending:      ptr.Wrap(1),
			MaxWaiting:         ptr.Wrap(1),
			RateLimit:          ptr.Wrap(uint64(1)),
			SampleFrequency:    ptr.Wrap("1"),
			InactiveThreshold:  ptr.Wrap(time.Second),
			BackOff:            []time.Duration{time.Second},
			MaxRequestBatch:    ptr.Wrap(1),
			MaxRequestMaxBytes: ptr.Wrap(int64(1)),
			MaxRequestExpires:  ptr.Wrap(time.Second),
			Metadata:           map[string]string{"x": "y"},
		})

		assert.Equal(t, "test-consumer", result.Name)
		assert.Equal(t, "test-consumer", result.Durable)
		assert.Equal(t, jetstream.DeliverLastPolicy, result.DeliverPolicy)
		assert.Equal(t, jetstream.AckAllPolicy, result.AckPolicy)
		assert.Equal(t, jetstream.ReplayOriginalPolicy, result.ReplayPolicy)
		assert.Equal(t, "foo.>", result.FilterSubject)
		assert.True(t, result.HeadersOnly)
		assert.Equal(t, 3, result.Replicas)
		assert.True(t, result.MemoryStorage)
		assert.Equal(t, "deliver.test", result.DeliverSubject)
		assert.Equal(t, "group1", result.DeliverGroup)
		assert.True(t, result.FlowControl)
		assert.Equal(t, 30*time.Second, result.IdleHeartbeat)
		assert.Equal(t, uint64(42), result.OptStartSeq)
		assert.Nil(t, result.FilterSubjects)
		assert.Nil(t, result.OptStartTime)
	})

	// Sequential updates

	t.Run("sequential updates accumulate", func(t *testing.T) {
		base := fullConsumerConfig()

		// First: update Description.
		r1 := svc.mergeConsumerUpdate(base, entities.ConsumerUpdateRequest{
			Description: ptr.Wrap("step1"),
		})
		// Second: update AckWait.
		r2 := svc.mergeConsumerUpdate(r1, entities.ConsumerUpdateRequest{
			AckWait: ptr.Wrap(90 * time.Second),
		})
		// Third: update MaxDeliver.
		r3 := svc.mergeConsumerUpdate(r2, entities.ConsumerUpdateRequest{
			MaxDeliver: ptr.Wrap(99),
		})

		assert.Equal(t, "step1", r3.Description)
		assert.Equal(t, 90*time.Second, r3.AckWait)
		assert.Equal(t, 99, r3.MaxDeliver)
		assertConsumerUnchangedExcept(t, base, r3, map[string]bool{
			"Description": true, "AckWait": true, "MaxDeliver": true,
		})
	})

	t.Run("sequential update same field twice", func(t *testing.T) {
		base := fullConsumerConfig()

		r1 := svc.mergeConsumerUpdate(base, entities.ConsumerUpdateRequest{
			Description: ptr.Wrap("first"),
		})
		r2 := svc.mergeConsumerUpdate(r1, entities.ConsumerUpdateRequest{
			Description: ptr.Wrap("second"),
		})

		assert.Equal(t, "second", r2.Description)
		assertConsumerUnchangedExcept(t, base, r2, map[string]bool{"Description": true})
	})

	// Metadata replace semantics

	t.Run("metadata replace semantics", func(t *testing.T) {
		t.Run("replaces entire map", func(t *testing.T) {
			current := jetstream.ConsumerConfig{
				Metadata: map[string]string{"a": "1", "b": "2", "c": "3"},
			}
			result := svc.mergeConsumerUpdate(current, entities.ConsumerUpdateRequest{
				Metadata: map[string]string{"x": "10"},
			})
			assert.Equal(t, map[string]string{"x": "10"}, result.Metadata)
		})

		t.Run("empty map replaces and clears", func(t *testing.T) {
			current := jetstream.ConsumerConfig{
				Metadata: map[string]string{"a": "1", "b": "2"},
			}
			result := svc.mergeConsumerUpdate(current, entities.ConsumerUpdateRequest{
				Metadata: map[string]string{},
			})
			assert.Equal(t, map[string]string{}, result.Metadata)
		})
	})

	// BackOff replace semantics

	t.Run("backoff replace semantics", func(t *testing.T) {
		t.Run("replaces entire slice", func(t *testing.T) {
			current := jetstream.ConsumerConfig{
				BackOff: []time.Duration{time.Second, 5 * time.Second, 30 * time.Second},
			}
			result := svc.mergeConsumerUpdate(current, entities.ConsumerUpdateRequest{
				BackOff: []time.Duration{10 * time.Second},
			})
			assert.Equal(t, []time.Duration{10 * time.Second}, result.BackOff)
		})

		t.Run("reducing backoff length works", func(t *testing.T) {
			current := jetstream.ConsumerConfig{
				BackOff: []time.Duration{time.Second, 5 * time.Second, 30 * time.Second, time.Minute},
			}
			result := svc.mergeConsumerUpdate(current, entities.ConsumerUpdateRequest{
				BackOff: []time.Duration{2 * time.Second, 10 * time.Second},
			})
			assert.Equal(t, []time.Duration{2 * time.Second, 10 * time.Second}, result.BackOff)
			assert.Len(t, result.BackOff, 2)
		})
	})

	// Edge cases

	t.Run("edge cases", func(t *testing.T) {
		t.Run("all nil pointers and empty slices change nothing", func(t *testing.T) {
			base := fullConsumerConfig()
			result := svc.mergeConsumerUpdate(base, entities.ConsumerUpdateRequest{})
			assertConsumerUnchangedExcept(t, base, result, nil)
		})

		t.Run("MaxRequestMaxBytes int64 to int conversion", func(t *testing.T) {
			base := fullConsumerConfig()
			result := svc.mergeConsumerUpdate(base, entities.ConsumerUpdateRequest{
				MaxRequestMaxBytes: ptr.Wrap(int64(math.MaxInt32)),
			})
			assert.Equal(t, math.MaxInt32, result.MaxRequestMaxBytes)
			assertConsumerUnchangedExcept(t, base, result, map[string]bool{"MaxRequestMaxBytes": true})
		})

		t.Run("multiple duration fields updated simultaneously", func(t *testing.T) {
			base := fullConsumerConfig()
			result := svc.mergeConsumerUpdate(base, entities.ConsumerUpdateRequest{
				AckWait:           ptr.Wrap(45 * time.Second),
				InactiveThreshold: ptr.Wrap(30 * time.Minute),
				MaxRequestExpires: ptr.Wrap(3 * time.Minute),
			})
			assert.Equal(t, 45*time.Second, result.AckWait)
			assert.Equal(t, 30*time.Minute, result.InactiveThreshold)
			assert.Equal(t, 3*time.Minute, result.MaxRequestExpires)
			assertConsumerUnchangedExcept(t, base, result, map[string]bool{
				"AckWait": true, "InactiveThreshold": true, "MaxRequestExpires": true,
			})
		})
	})
}
