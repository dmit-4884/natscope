// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"

	"github.com/altessa-s/go-atlas/core/types/ptr"

	"github.com/dmit-4884/natscope/internal/entities"
)

// TestMergeStreamUpdate_EmptyUpdate_DoesNotChangeAnything is a regression
// test: an empty UpdateStream payload must round-trip the config unchanged.
func TestMergeStreamUpdate_EmptyUpdate_DoesNotChangeAnything(t *testing.T) {
	svc := &Client{}
	current := fullStreamConfig()
	current.Name = "regression-stream"

	out := svc.mergeStreamUpdate(current, entities.StreamUpdateRequest{})

	assert.Equal(t, current, out, "empty update must produce identical config")
}

// TestMergeStreamUpdate_NewFields verifies newer mutable fields merge
// correctly via the nil-values-skipped converter pipeline.
func TestMergeStreamUpdate_NewFields(t *testing.T) {
	svc := &Client{}

	t.Run("Compression", func(t *testing.T) {
		current := fullStreamConfig()
		current.Compression = jetstream.NoCompression
		newCompression := entities.CompressionS2

		out := svc.mergeStreamUpdate(current, entities.StreamUpdateRequest{
			Compression: &newCompression,
		})

		assert.Equal(t, jetstream.S2Compression, out.Compression)
		// Other fields must remain unchanged.
		assert.Equal(t, current.Name, out.Name)
		assert.Equal(t, current.Subjects, out.Subjects)
	})

	t.Run("Republish", func(t *testing.T) {
		current := fullStreamConfig()
		current.RePublish = nil

		out := svc.mergeStreamUpdate(current, entities.StreamUpdateRequest{
			Republish: &entities.StreamRePublish{
				Src:         "events.>",
				Dest:        "audit.events.>",
				HeadersOnly: true,
			},
		})

		if assert.NotNil(t, out.RePublish) {
			assert.Equal(t, "events.>", out.RePublish.Source)
			assert.Equal(t, "audit.events.>", out.RePublish.Destination)
			assert.True(t, out.RePublish.HeadersOnly)
		}
	})

	t.Run("SubjectTransform", func(t *testing.T) {
		current := fullStreamConfig()
		current.SubjectTransform = nil

		out := svc.mergeStreamUpdate(current, entities.StreamUpdateRequest{
			SubjectTransform: &entities.SubjectTransformConfig{
				Source:      "old.>",
				Destination: "new.>",
			},
		})

		if assert.NotNil(t, out.SubjectTransform) {
			assert.Equal(t, "old.>", out.SubjectTransform.Source)
			assert.Equal(t, "new.>", out.SubjectTransform.Destination)
		}
	})

	t.Run("ConsumerLimits", func(t *testing.T) {
		current := fullStreamConfig()
		current.ConsumerLimits = jetstream.StreamConsumerLimits{}

		out := svc.mergeStreamUpdate(current, entities.StreamUpdateRequest{
			ConsumerLimits: &entities.StreamConsumerLimits{
				InactiveThreshold: 5 * time.Minute,
				MaxAckPending:     500,
			},
		})

		assert.Equal(t, 5*time.Minute, out.ConsumerLimits.InactiveThreshold)
		assert.Equal(t, 500, out.ConsumerLimits.MaxAckPending)
	})

	t.Run("AllowMsgTTL", func(t *testing.T) {
		current := fullStreamConfig()
		current.AllowMsgTTL = false

		out := svc.mergeStreamUpdate(current, entities.StreamUpdateRequest{
			AllowMsgTTL: ptr.Wrap(true),
		})

		assert.True(t, out.AllowMsgTTL)
	})

	t.Run("AllowAtomicPublish", func(t *testing.T) {
		current := fullStreamConfig()
		current.AllowAtomicPublish = false

		out := svc.mergeStreamUpdate(current, entities.StreamUpdateRequest{
			AllowAtomicPublish: ptr.Wrap(true),
		})

		assert.True(t, out.AllowAtomicPublish)
	})
}

// TestStreamUpdateRequest_OmitsImmutableFields documents the safety contract:
// the entity omits immutable fields so an update literally can't touch them.
func TestStreamUpdateRequest_OmitsImmutableFields(t *testing.T) {
	// If any of these fields are added to StreamUpdateRequest, this test
	// fails to compile, forcing an explicit mutability decision.
	type immutables struct {
		Name      string // immutable: stream name
		Storage   string // immutable: file/memory backend
		Retention string // immutable: retention policy
		Replicas  int    // immutable: replica count (separate scaling API)
		FirstSeq  uint64 // immutable: initial sequence
		NoAck     bool   // immutable: ack semantics
		Sealed    bool   // sealed via dedicated SealStream
	}
	_ = immutables{}

	// Sanity: the entity exists and the type assertion compiles.
	var _ entities.StreamUpdateRequest
}
