// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultListLimit(t *testing.T) {
	t.Parallel()
	assert.Equal(t, int64(50), DefaultListLimit)
}

func TestListBase_GetLimit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		limit *int64
		want  int64
	}{
		{name: "NilLimit_ReturnsDefault", limit: nil, want: DefaultListLimit},
		{name: "CustomLimit", limit: ptrInt64(25), want: 25},
		{name: "ZeroLimit_ReturnsDefault", limit: ptrInt64(0), want: DefaultListLimit},
		{name: "NegativeLimit_ReturnsDefault", limit: ptrInt64(-5), want: DefaultListLimit},
		{name: "AtMaxLimit", limit: ptrInt64(MaxListLimit), want: MaxListLimit},
		{name: "OverMaxLimit_Clamped", limit: ptrInt64(1000), want: MaxListLimit},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			l := &ListBase{Limit: tt.limit}
			assert.Equal(t, tt.want, l.GetLimit())
		})
	}
}

func TestSoftDelete(t *testing.T) {
	t.Parallel()

	sd := SoftDelete{
		Id:           "test-id",
		NewUpdatedAt: time.UnixMilli(123456).UTC(),
	}
	assert.Equal(t, "test-id", sd.Id)
	assert.Equal(t, time.UnixMilli(123456).UTC(), sd.NewUpdatedAt)
}
