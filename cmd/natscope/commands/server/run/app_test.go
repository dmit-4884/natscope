// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package run

import (
	"errors"
	"fmt"
	"testing"
)

func TestParseMemoryLimit(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    int64
		wantErr bool
	}{
		{"plain bytes", "268435456", 268435456, false},
		{"MiB suffix", "256MiB", 256 * 1 << 20, false},
		{"GiB suffix", "1GiB", 1 << 30, false},
		{"KiB suffix", "512KiB", 512 * 1 << 10, false},
		{"B suffix", "100B", 100, false},
		{"whitespace", " 256MiB ", 256 * 1 << 20, false},
		{"garbage", "abc", 0, true},
		{"negative", "-5", 0, true},
		{"zero", "0", 0, true},
		{"empty unit value", "MiB", 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseMemoryLimit(tc.in)
			if (err != nil) != tc.wantErr {
				t.Fatalf("parseMemoryLimit(%q) error = %v, wantErr %v", tc.in, err, tc.wantErr)
			}
			if !tc.wantErr && got != tc.want {
				t.Errorf("parseMemoryLimit(%q) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}

func TestRootCause(t *testing.T) {
	leaf := errors.New("refusing to bind 0.0.0.0:4280 without authentication")
	wrapped := fmt.Errorf("could not build value group grpc.Handler: %w", leaf)
	wrapped = fmt.Errorf(`could not build arguments for function "reflect".makeFuncStub: %w`, wrapped)

	if got := rootCause(wrapped); !errors.Is(got, leaf) || got.Error() != leaf.Error() {
		t.Errorf("rootCause() = %v, want %v", got, leaf)
	}
	if got := rootCause(leaf); got != leaf {
		t.Errorf("rootCause() on an already-leaf error = %v, want it unchanged", got)
	}
}
