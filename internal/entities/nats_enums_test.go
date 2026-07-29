// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRetentionPolicy_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		p    RetentionPolicy
		want bool
	}{
		{name: "Limits", p: RetentionLimits, want: true},
		{name: "Interest", p: RetentionInterest, want: true},
		{name: "WorkQueue", p: RetentionWorkQueue, want: true},
		{name: "Invalid_Negative", p: RetentionPolicy(-1), want: false},
		{name: "Invalid_OutOfRange", p: RetentionPolicy(99), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.p.IsValid())
		})
	}
}

func TestStorageType_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		st   StorageType
		want bool
	}{
		{name: "File", st: StorageFile, want: true},
		{name: "Memory", st: StorageMemory, want: true},
		{name: "Invalid", st: StorageType(99), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.st.IsValid())
		})
	}
}

func TestDiscardPolicy_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		p    DiscardPolicy
		want bool
	}{
		{name: "Old", p: DiscardOld, want: true},
		{name: "New", p: DiscardNew, want: true},
		{name: "Invalid", p: DiscardPolicy(99), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.p.IsValid())
		})
	}
}

func TestStoreCompression_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		c    StoreCompression
		want bool
	}{
		{name: "None", c: CompressionNone, want: true},
		{name: "S2", c: CompressionS2, want: true},
		{name: "Invalid", c: StoreCompression(99), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.c.IsValid())
		})
	}
}

func TestDeliverPolicy_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		p    DeliverPolicy
		want bool
	}{
		{name: "All", p: DeliverAll, want: true},
		{name: "Last", p: DeliverLast, want: true},
		{name: "New", p: DeliverNew, want: true},
		{name: "ByStartSequence", p: DeliverByStartSequence, want: true},
		{name: "ByStartTime", p: DeliverByStartTime, want: true},
		{name: "LastPerSubject", p: DeliverLastPerSubject, want: true},
		{name: "Invalid", p: DeliverPolicy(99), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.p.IsValid())
		})
	}
}

func TestAckPolicy_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		p    AckPolicy
		want bool
	}{
		{name: "Explicit", p: AckExplicit, want: true},
		{name: "All", p: AckAll, want: true},
		{name: "None", p: AckNone, want: true},
		{name: "Invalid", p: AckPolicy(99), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.p.IsValid())
		})
	}
}

func TestReplayPolicy_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		p    ReplayPolicy
		want bool
	}{
		{name: "Instant", p: ReplayInstant, want: true},
		{name: "Original", p: ReplayOriginal, want: true},
		{name: "Invalid", p: ReplayPolicy(99), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.p.IsValid())
		})
	}
}
