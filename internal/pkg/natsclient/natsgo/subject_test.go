// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package natsgo

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dmit-4884/natscope/internal/errs"
)

func TestContainsWildcard(t *testing.T) {
	t.Parallel()

	assert.True(t, containsWildcard("orders.*"))
	assert.True(t, containsWildcard("orders.>"))
	assert.True(t, containsWildcard("*"))
	assert.False(t, containsWildcard("orders.created"))
	assert.False(t, containsWildcard(""))
}

func TestValidateLengths(t *testing.T) {
	t.Parallel()

	assert.NoError(t, validateNATSNameLength("stream name", strings.Repeat("s", MaxNATSNameLength)))
	assert.ErrorIs(t, validateNATSNameLength("stream name", strings.Repeat("s", MaxNATSNameLength+1)), errs.ErrNATSInvalidArgument)
	assert.NoError(t, validateNATSSubjectLength("subject", strings.Repeat("a.", 300)))
	assert.ErrorIs(t, validateNATSSubjectLength("subject", strings.Repeat("a", MaxNATSSubjectLength+1)), errs.ErrNATSInvalidArgument)
}
