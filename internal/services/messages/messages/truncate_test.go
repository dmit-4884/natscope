// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package messages

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/dmit-4884/natscope/internal/entities"
)

// TestTruncateMessages_HappyPath verifies DataBase64 truncates on a clean
// boundary and Decoded is dropped (not half-cut JSON) once over cap.
func TestTruncateMessages_HappyPath(t *testing.T) {
	t.Parallel()

	const rawSize = 8 * 1024 // 8 KB raw → 10 KB+ base64
	raw := make([]byte, rawSize)
	for i := range raw {
		raw[i] = byte('A' + (i % 26))
	}
	encoded := base64.StdEncoding.EncodeToString(raw)

	const cap = 1024 // 1 KB
	originalDecoded := strings.Repeat("x", 4096)
	msg := &entities.Message{
		Sequence:    42,
		DataBase64:  encoded,
		DataSize:    rawSize,
		Decoded:     json.RawMessage(originalDecoded),
		DecodedType: "blitz.bus.EmailMessage",
	}
	truncateMessages([]*entities.Message{msg}, cap)

	if !msg.Truncated {
		t.Fatalf("expected Truncated=true after cap=%d on payload of %d", cap, rawSize)
	}
	if msg.DataSize != rawSize {
		t.Fatalf("DataSize must remain original; got %d want %d", msg.DataSize, rawSize)
	}
	// Truncated base64 must still decode (no half-quartet at the tail).
	decoded, err := base64.StdEncoding.DecodeString(msg.DataBase64)
	if err != nil {
		t.Fatalf("truncated DataBase64 does not decode: %v (len=%d)", err, len(msg.DataBase64))
	}
	if len(decoded) > cap {
		t.Fatalf("decoded prefix too long: %d > cap %d", len(decoded), cap)
	}
	// Decoded becomes a small JSON-string preview ending in "…" — must stay
	// valid JSON for the frontend's parse, and much smaller than the original.
	if msg.Decoded == nil {
		t.Fatal("Decoded must be a small preview after truncate, got nil")
	}
	if len(msg.Decoded) >= len(originalDecoded) {
		t.Fatalf("Decoded preview must be smaller than original; got %d vs %d", len(msg.Decoded), len(originalDecoded))
	}
	var previewStr string
	if err := json.Unmarshal(msg.Decoded, &previewStr); err != nil {
		t.Fatalf("Decoded preview must be a JSON string: %v (raw=%q)", err, string(msg.Decoded))
	}
	if !strings.HasSuffix(previewStr, "…") {
		t.Fatalf("Decoded preview must end with ellipsis marker, got %q", previewStr)
	}
	// DecodedType is preserved on the entity so the UI can still tell the
	// user "this would decode as blitz.bus.EmailMessage if loaded".
	if msg.DecodedType != "blitz.bus.EmailMessage" {
		t.Fatalf("DecodedType lost: got %q", msg.DecodedType)
	}
}

// TestTruncateMessages_NoOpUnderCap proves that messages already smaller
// than the cap are left untouched (no Truncated flag, no field rewrites).
func TestTruncateMessages_NoOpUnderCap(t *testing.T) {
	t.Parallel()

	raw := []byte("short payload")
	encoded := base64.StdEncoding.EncodeToString(raw)
	msg := &entities.Message{
		DataBase64: encoded,
		DataSize:   len(raw),
		Decoded:    json.RawMessage(`{"ok":true}`),
	}
	truncateMessages([]*entities.Message{msg}, 1024)

	if msg.Truncated {
		t.Fatal("Truncated must stay false when payload < cap")
	}
	if msg.DataBase64 != encoded {
		t.Fatal("DataBase64 must be unchanged when payload < cap")
	}
	if string(msg.Decoded) != `{"ok":true}` {
		t.Fatalf("Decoded must be unchanged when payload < cap; got %q", string(msg.Decoded))
	}
}

// TestTruncateMessages_DisabledByZeroCap covers the contract that cap=0
// means "unlimited" — the function must not modify any field.
func TestTruncateMessages_DisabledByZeroCap(t *testing.T) {
	t.Parallel()

	raw := make([]byte, 4096)
	encoded := base64.StdEncoding.EncodeToString(raw)
	msg := &entities.Message{
		DataBase64: encoded,
		DataSize:   len(raw),
	}
	truncateMessages([]*entities.Message{msg}, 0)
	if msg.Truncated {
		t.Fatal("cap=0 must not set Truncated")
	}
	if msg.DataBase64 != encoded {
		t.Fatal("cap=0 must not change DataBase64")
	}
}

// TestTruncateMessages_DataSizeBackfill covers DataSize unset on input,
// backfilled from base64 length so the UI badge keeps making sense.
func TestTruncateMessages_DataSizeBackfill(t *testing.T) {
	t.Parallel()

	raw := make([]byte, 2048)
	encoded := base64.StdEncoding.EncodeToString(raw)
	msg := &entities.Message{
		DataBase64: encoded, // DataSize intentionally 0
	}
	truncateMessages([]*entities.Message{msg}, 256)

	if msg.DataSize == 0 {
		t.Fatal("DataSize should be backfilled from DataBase64 when missing")
	}
	if !msg.Truncated {
		t.Fatal("expected Truncated=true")
	}
}

// TestBase64DecodedLen verifies the helper handles padded and unpadded
// strings the same way the standard decoder would.
func TestBase64DecodedLen(t *testing.T) {
	t.Parallel()

	cases := []struct {
		raw int
	}{
		{0}, {1}, {2}, {3}, {4}, {16}, {1023}, {1024}, {1025},
	}
	for _, c := range cases {
		raw := make([]byte, c.raw)
		got := base64DecodedLen(base64.StdEncoding.EncodeToString(raw))
		if got != c.raw {
			t.Errorf("base64DecodedLen for %d raw bytes = %d", c.raw, got)
		}
	}
}
