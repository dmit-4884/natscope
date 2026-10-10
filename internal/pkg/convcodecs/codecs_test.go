// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package convcodecs

import (
	"math"
	"reflect"
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

func TestDurationString(t *testing.T) {
	type src struct {
		MaxAge  time.Duration
		AckWait time.Duration
	}
	type dst struct {
		MaxAge  string
		AckWait string
	}

	result := converter.Convert(&src{MaxAge: 90 * time.Minute}, &dst{}, converter.WithCodecs(DurationString))

	assert.Equal(t, "1h30m0s", result.MaxAge)
	assert.Empty(t, result.AckWait)
}

func TestEnumNames(t *testing.T) {
	type level int
	type src struct {
		Level   level
		Unknown level
		Count   int
	}
	type dst struct {
		Level   string
		Unknown string
		Count   int
	}

	codec := EnumNames(map[reflect.Type][]string{reflect.TypeFor[level](): {"low", "high"}})
	result := converter.Convert(&src{Level: 1, Unknown: 7, Count: 3}, &dst{}, converter.WithCodecs(codec))

	assert.Equal(t, "high", result.Level)
	assert.Equal(t, "7", result.Unknown)
	assert.Equal(t, 3, result.Count)
}
