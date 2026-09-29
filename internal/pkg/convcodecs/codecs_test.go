// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package convcodecs

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-atlas/domain/converter"
	"github.com/altessa-s/go-atlas/domain/converter/codec/durpb"

	"google.golang.org/protobuf/types/known/durationpb"
)

// TestDurationSaturating_Overflow checks that an out-of-range Duration saturates instead of wrapping.
func TestDurationSaturating_Overflow(t *testing.T) {
	type dst struct {
		MaxAge time.Duration
	}
	type src struct {
		MaxAge *durationpb.Duration
	}

	huge := &durationpb.Duration{Seconds: 18446744074}

	result := converter.Convert(&src{MaxAge: huge}, &dst{},
		converter.WithCodecs(DurationSaturating, durpb.New()),
	)

	assert.Equal(t, time.Duration(math.MaxInt64), result.MaxAge)
}

// TestDurationSaturating_PointerPreservesExplicitZero checks that a present zero Duration yields a non-nil pointer.
func TestDurationSaturating_PointerPreservesExplicitZero(t *testing.T) {
	type dst struct {
		MaxAge *time.Duration
	}
	type src struct {
		MaxAge *durationpb.Duration
	}

	result := converter.Convert(&src{MaxAge: &durationpb.Duration{}}, &dst{},
		converter.WithCodecs(DurationSaturating, durpb.New(durpb.WithIgnoreZero())),
	)

	require.NotNil(t, result.MaxAge)
	assert.Equal(t, time.Duration(0), *result.MaxAge)
}

func TestDurationSaturating_NormalValuePassesThrough(t *testing.T) {
	type dst struct {
		AckWait time.Duration
	}
	type src struct {
		AckWait *durationpb.Duration
	}

	result := converter.Convert(&src{AckWait: durationpb.New(30 * time.Second)}, &dst{},
		converter.WithCodecs(DurationSaturating, durpb.New()),
	)

	assert.Equal(t, 30*time.Second, result.AckWait)
}
