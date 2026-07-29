// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package history

import (
	"strings"
	"testing"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/pkg/bbstore/bbstoretest"

	historyBbolt "github.com/dmit-4884/natscope/internal/storages/history/bbolt"
)

// TestRecord_PayloadTruncated: over-cap payloads store truncated with the flag
// set; PayloadSize keeps the original byte count.
func TestRecord_PayloadTruncated(t *testing.T) {
	t.Parallel()

	storage, err := historyBbolt.New(t.Context(), bbstoretest.NewMemoryDB(t))
	if err != nil {
		t.Fatalf("new history storage: %v", err)
	}
	svc := New(storage)

	huge := strings.Repeat("x", entities.HistoryPayloadPreviewBytes*4)
	got, err := svc.Record(t.Context(), &entities.PublishHistoryCreate{
		ConnectionURL: "nats://localhost:4222",
		Stream:        "BENCH",
		Subject:       "bench.1",
		EncodingType:  entities.EncodingTypeJSON,
		PayloadJSON:   huge,
		Success:       true,
	})
	if err != nil {
		t.Fatalf("record: %v", err)
	}
	if !got.PayloadTruncated {
		t.Fatal("expected PayloadTruncated=true for huge payload")
	}
	if got.PayloadSize != len(huge) {
		t.Fatalf("PayloadSize=%d want %d (original size must be preserved)", got.PayloadSize, len(huge))
	}
	if len(got.PayloadJSON) != entities.HistoryPayloadPreviewBytes {
		t.Fatalf("PayloadJSON len=%d want %d (preview cap)", len(got.PayloadJSON), entities.HistoryPayloadPreviewBytes)
	}
}

// TestRecord_PayloadSmallNoTruncation: small payloads store verbatim, flag false.
func TestRecord_PayloadSmallNoTruncation(t *testing.T) {
	t.Parallel()

	storage, err := historyBbolt.New(t.Context(), bbstoretest.NewMemoryDB(t))
	if err != nil {
		t.Fatalf("new history storage: %v", err)
	}
	svc := New(storage)

	got, err := svc.Record(t.Context(), &entities.PublishHistoryCreate{
		ConnectionURL: "nats://localhost:4222",
		Stream:        "ORDERS",
		Subject:       "orders.created",
		EncodingType:  entities.EncodingTypeJSON,
		PayloadJSON:   `{"id":"1"}`,
		Success:       true,
	})
	if err != nil {
		t.Fatalf("record: %v", err)
	}
	if got.PayloadTruncated {
		t.Fatal("PayloadTruncated must be false for small payload")
	}
	if got.PayloadSize != len(`{"id":"1"}`) {
		t.Fatalf("PayloadSize=%d want %d", got.PayloadSize, len(`{"id":"1"}`))
	}
	if got.PayloadJSON != `{"id":"1"}` {
		t.Fatalf("PayloadJSON=%q must equal input", got.PayloadJSON)
	}
}
