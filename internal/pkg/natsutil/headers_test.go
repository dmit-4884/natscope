// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateHeaderName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		key  string
		want bool
	}{
		{name: "Simple", key: "X-Ok", want: true},
		{name: "Digits", key: "X-Trace-1", want: true},
		{name: "Empty", key: "", want: false},
		{name: "Space", key: "a b", want: false},
		{name: "Colon", key: "a:b", want: false},
		{name: "Unicode", key: "Ключ", want: false},
		{name: "CRLF", key: "a\r\nb", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, ValidateHeaderName(tt.key))
		})
	}
}

func TestValidateHeaderNames(t *testing.T) {
	t.Parallel()

	require := assert.New(t)
	require.NoError(ValidateHeaderNames(map[string]string{"X-Ok": "ok"}))
	require.NoError(ValidateHeaderNames(nil))
	require.Error(ValidateHeaderNames(map[string]string{"a b": "space"}))
	require.Error(ValidateHeaderNames(map[string]string{"X-Ok": "ok", "": "empty"}))
}
