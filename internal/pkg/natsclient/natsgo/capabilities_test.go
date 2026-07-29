// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
)

// TestComputeCapabilities verifies API-level gating with semver fallback for
// servers that don't advertise a level (< 2.12 advertise 0).
func TestComputeCapabilities(t *testing.T) {
	tests := []struct {
		name      string
		version   string
		jsEnabled bool
		apiLevel  int
		want      entities.ServerCapabilities
	}{
		{
			name: "2.14 advertises level 4", version: "2.14.2", jsEnabled: true, apiLevel: 4,
			want: entities.ServerCapabilities{ApiLevel: 4, ConsumerPause: true, MessageTtl: true, AtomicPublish: true},
		},
		{
			name: "2.12 advertises level 2", version: "2.12.0", jsEnabled: true, apiLevel: 2,
			want: entities.ServerCapabilities{ApiLevel: 2, ConsumerPause: true, MessageTtl: true, AtomicPublish: true},
		},
		{
			name: "2.11 without advertised level falls back to semver", version: "2.11.3", jsEnabled: true, apiLevel: 0,
			want: entities.ServerCapabilities{ApiLevel: 1, ConsumerPause: true, MessageTtl: true, AtomicPublish: false},
		},
		{
			name: "v-prefixed version string", version: "v2.11.0", jsEnabled: true, apiLevel: 0,
			want: entities.ServerCapabilities{ApiLevel: 1, ConsumerPause: true, MessageTtl: true, AtomicPublish: false},
		},
		{
			name: "2.10 without advertised level has no gated features", version: "2.10.24", jsEnabled: true, apiLevel: 0,
			want: entities.ServerCapabilities{ApiLevel: 0},
		},
		{
			name: "jetstream disabled clears feature flags but keeps level", version: "2.14.2", jsEnabled: false, apiLevel: 4,
			want: entities.ServerCapabilities{ApiLevel: 4},
		},
		{
			name: "empty version is conservative", version: "", jsEnabled: true, apiLevel: 0,
			want: entities.ServerCapabilities{ApiLevel: 0},
		},
		{
			name: "garbage version is conservative", version: "not-a-version", jsEnabled: true, apiLevel: 0,
			want: entities.ServerCapabilities{ApiLevel: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeCapabilities(tt.version, tt.jsEnabled, tt.apiLevel)
			require.NotNil(t, got)
			assert.Equal(t, tt.want, *got)
		})
	}
}
