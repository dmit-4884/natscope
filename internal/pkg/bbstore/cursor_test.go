// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package bbstore

import (
	"encoding/base64"
	"errors"
	"testing"
)

func TestParseCursor(t *testing.T) {
	t.Run("empty is first page", func(t *testing.T) {
		c, err := parseCursor("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c.has != 0 {
			t.Errorf("empty cursor should be first page, got has=%d", c.has)
		}
	})

	t.Run("round-trips a valid cursor", func(t *testing.T) {
		c, err := parseCursor(encodeCursor(42, "id-7"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c.sort != 42 || c.id != "id-7" || c.has != 1 {
			t.Errorf("got %+v, want sort=42 id=id-7 has=1", c)
		}
	})

	// Every malformed form must surface ErrInvalidCursor so the transport maps
	// it to a clean invalid-argument status instead of leaking parse text.
	t.Run("invalid forms wrap ErrInvalidCursor", func(t *testing.T) {
		bad := []string{
			"!!!not-base64!!!",
			encodeNoSep(),
			encodeBadSort(),
		}
		for _, s := range bad {
			if _, err := parseCursor(s); !errors.Is(err, ErrInvalidCursor) {
				t.Errorf("parseCursor(%q) = %v, want ErrInvalidCursor", s, err)
			}
		}
	})
}

// encodeNoSep builds a base64 payload with no separator (single field).
func encodeNoSep() string {
	return base64.RawURLEncoding.EncodeToString([]byte("only-one-field"))
}

// encodeBadSort builds a well-formed two-field cursor whose sort key is not an int.
func encodeBadSort() string {
	return base64.RawURLEncoding.EncodeToString([]byte("not-a-number" + cursorSep + "id"))
}
