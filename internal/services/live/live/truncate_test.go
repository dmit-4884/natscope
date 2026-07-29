// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package live

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/dmit-4884/natscope/internal/entities"
)

// TestTruncateLiveMessage_HappyPath verifies Data is byte-capped and Decoded
// becomes a bounded JSON preview ending in "…", flipping Truncated for the UI.
func TestTruncateLiveMessage_HappyPath(t *testing.T) {
	t.Parallel()

	bigData := make([]byte, 16*1024)
	for i := range bigData {
		bigData[i] = byte('A' + (i % 26))
	}
	bigDecoded := strings.Repeat("x", 16*1024)
	dt := "blitz.bus.EmailMessage"

	lm := &entities.LiveMessage{
		NatsMessage: entities.NatsMessage{
			Subject: "bench.42",
			Data:    bigData,
		},
		Decoded:     &bigDecoded,
		DecodedType: &dt,
	}
	truncateLiveMessage(lm, 1024)

	if !lm.Truncated {
		t.Fatal("expected Truncated=true")
	}
	if len(lm.NatsMessage.Data) != 1024 {
		t.Fatalf("Data len=%d want 1024", len(lm.NatsMessage.Data))
	}
	if lm.NatsMessage.Subject != "bench.42" {
		t.Fatal("metadata should be untouched")
	}
	if lm.Decoded == nil {
		t.Fatal("Decoded must be a small preview after truncate, got nil")
	}
	if len(*lm.Decoded) >= len(bigDecoded) {
		t.Fatalf("Decoded preview must be smaller than original; got %d vs %d", len(*lm.Decoded), len(bigDecoded))
	}
	var previewStr string
	if err := json.Unmarshal([]byte(*lm.Decoded), &previewStr); err != nil {
		t.Fatalf("Decoded preview must be a JSON string: %v (raw=%q)", err, *lm.Decoded)
	}
	if !strings.HasSuffix(previewStr, "…") {
		t.Fatalf("Decoded preview must end with ellipsis marker, got %q", previewStr)
	}
	if lm.DecodedType == nil || *lm.DecodedType != "blitz.bus.EmailMessage" {
		t.Fatal("DecodedType must be preserved so the UI can still show the proto type")
	}
}

// TestTruncateLiveMessage_NoOpUnderCap ensures small messages are not
// flagged as truncated.
func TestTruncateLiveMessage_NoOpUnderCap(t *testing.T) {
	t.Parallel()

	dec := "{}"
	lm := &entities.LiveMessage{
		NatsMessage: entities.NatsMessage{Data: []byte("hi")},
		Decoded:     &dec,
	}
	truncateLiveMessage(lm, 1024)

	if lm.Truncated {
		t.Fatal("Truncated must stay false")
	}
	if string(lm.NatsMessage.Data) != "hi" {
		t.Fatal("Data must not be modified")
	}
	if *lm.Decoded != "{}" {
		t.Fatal("Decoded must not be modified")
	}
}

// TestTruncateLiveMessage_ZeroCapDisabled covers the "feature off" path —
// cap=0 means unlimited.
func TestTruncateLiveMessage_ZeroCapDisabled(t *testing.T) {
	t.Parallel()

	bigData := make([]byte, 8*1024)
	lm := &entities.LiveMessage{
		NatsMessage: entities.NatsMessage{Data: bigData},
	}
	truncateLiveMessage(lm, 0)

	if lm.Truncated {
		t.Fatal("cap=0 must not flip Truncated")
	}
	if len(lm.NatsMessage.Data) != len(bigData) {
		t.Fatal("cap=0 must not shrink Data")
	}
}

// TestTruncateLiveMessage_DecodedNilSafe ensures we don't panic when the
// decoder did not produce output.
func TestTruncateLiveMessage_DecodedNilSafe(t *testing.T) {
	t.Parallel()

	bigData := make([]byte, 8*1024)
	lm := &entities.LiveMessage{
		NatsMessage: entities.NatsMessage{Data: bigData},
	}
	truncateLiveMessage(lm, 1024)
	if !lm.Truncated {
		t.Fatal("expected Truncated=true on big Data")
	}
	if lm.Decoded != nil {
		t.Fatal("Decoded should remain nil")
	}
}
