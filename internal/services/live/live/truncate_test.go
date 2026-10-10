// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

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

// TestTruncateLiveMessage_ReleasesFullPayload checks that the capped payload doesn't pin the original array.
func TestTruncateLiveMessage_ReleasesFullPayload(t *testing.T) {
	t.Parallel()

	lm := &entities.LiveMessage{NatsMessage: entities.NatsMessage{Subject: "s", Data: make([]byte, 1<<20)}}
	truncateLiveMessage(lm, 1024)

	if c := cap(lm.NatsMessage.Data); c > 1024 {
		t.Fatalf("truncated Data cap=%d, want <= 1024", c)
	}
}

// TestMessageHandler_BoundsBufferedBytes checks that the producer drops payloads past maxBufferedBytes.
func TestMessageHandler_BoundsBufferedBytes(t *testing.T) {
	t.Parallel()

	svc := &Service{}
	sess := newSessionState("conn")
	ch := make(chan *entities.NatsMessage, 10)
	handle := svc.buildMessageHandler(">", ch, sess)

	handle(&entities.NatsMessage{Subject: "big", Data: make([]byte, maxBufferedBytes+1)})
	if len(ch) != 0 || sess.messagesDropped.Load() != 1 {
		t.Fatalf("oversized payload: buffered=%d dropped=%d, want 0 and 1", len(ch), sess.messagesDropped.Load())
	}

	handle(&entities.NatsMessage{Subject: "small", Data: make([]byte, 100)})
	if len(ch) != 1 || sess.bufferedBytes.Load() != 100 {
		t.Fatalf("small payload: buffered=%d bytes=%d, want 1 and 100", len(ch), sess.bufferedBytes.Load())
	}
}
