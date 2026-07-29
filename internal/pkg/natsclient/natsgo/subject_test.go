// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContainsWildcard(t *testing.T) {
	t.Parallel()

	assert.True(t, containsWildcard("orders.*"))
	assert.True(t, containsWildcard("orders.>"))
	assert.True(t, containsWildcard("*"))
	assert.False(t, containsWildcard("orders.created"))
	assert.False(t, containsWildcard(""))
}
