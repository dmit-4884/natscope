// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package messages

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/dmit-4884/natscope/internal/entities"
)

// makeMessages builds n messages with the given payload size, pre-encoded to
// base64 (matching what toMessage produces in production).
func makeMessages(n, payloadSize int) []*entities.Message {
	msgs := make([]*entities.Message, n)
	raw := make([]byte, payloadSize)
	_, _ = rand.Read(raw)
	encoded := base64.StdEncoding.EncodeToString(raw)
	for i := range msgs {
		msgs[i] = &entities.Message{
			Sequence:    uint64(i + 1),
			Subject:     "bench.subject",
			DataBase64:  encoded,
			DataSize:    payloadSize,
			ContentType: entities.ContentTypeBinary,
		}
	}
	return msgs
}

// makeMixedMessages mirrors the BatchMixed shape used in nats bench.
func makeMixedMessages() []*entities.Message {
	sizes := []int{
		4 * 1024, 4 * 1024, 4 * 1024, 4 * 1024, 4 * 1024,
		100 * 1024, 100 * 1024, 100 * 1024,
		1600 * 1024, 1600 * 1024, 1600 * 1024, 1600 * 1024,
		5 * 1024 * 1024, 5 * 1024 * 1024,
	}
	msgs := make([]*entities.Message, len(sizes))
	for i, sz := range sizes {
		raw := make([]byte, sz)
		_, _ = rand.Read(raw)
		msgs[i] = &entities.Message{
			Sequence:    uint64(i + 1),
			Subject:     "bench.subject",
			DataBase64:  base64.StdEncoding.EncodeToString(raw),
			DataSize:    sz,
			ContentType: entities.ContentTypeBinary,
		}
	}
	return msgs
}

// BenchmarkTruncateMessages_23x1_6MB measures the truncate pass over 23
// messages of 1.6 MB; should be cheap (slice in place) but allocations tell.
func BenchmarkTruncateMessages_23x1_6MB(b *testing.B) {
	const maxBytes = 4096
	template := makeMessages(23, 1600*1024)

	b.ResetTimer()
	b.SetBytes(23 * 1600 * 1024)
	for b.Loop() {
		b.StopTimer()
		msgs := cloneMessages(template)
		b.StartTimer()
		truncateMessages(msgs, maxBytes)
	}
}

// BenchmarkTruncateMessages_Mixed exercises a mixed-size batch.
func BenchmarkTruncateMessages_Mixed(b *testing.B) {
	const maxBytes = 4096
	template := makeMixedMessages()

	b.ResetTimer()
	for b.Loop() {
		b.StopTimer()
		msgs := cloneMessages(template)
		b.StartTimer()
		truncateMessages(msgs, maxBytes)
	}
}

// BenchmarkDecodeableFilter_Mixed measures decodeableMessages — the cheap
// O(n) filter that decides which messages are worth sending to proto decode.
func BenchmarkDecodeableFilter_Mixed(b *testing.B) {
	const maxBytes int32 = 4096
	msgs := makeMixedMessages()

	b.ResetTimer()
	for b.Loop() {
		_ = decodeableMessages(msgs, maxBytes)
	}
}

// BenchmarkDecodedPreview_1MB measures the JSON preview-building cost (pretty
// print + line cut) hit once per truncated message with decoded JSON.
func BenchmarkDecodedPreview_1MB(b *testing.B) {
	// Build a 1 MB JSON document: array of objects.
	type item struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
		Body string `json:"body"`
	}
	const itemSize = 1024
	body := make([]byte, itemSize)
	for i := range body {
		body[i] = 'x'
	}
	items := make([]item, 1024)
	for i := range items {
		items[i] = item{ID: i, Name: "item", Body: string(body)}
	}
	raw, _ := json.Marshal(items)

	b.ResetTimer()
	b.SetBytes(int64(len(raw)))
	for b.Loop() {
		_ = decodedPreview(json.RawMessage(raw))
	}
}

// BenchmarkFilterByContent_Mixed measures filterByContent — substring search
// across decoded JSON + raw payload; worst case scans every message's body.
func BenchmarkFilterByContent_Mixed(b *testing.B) {
	msgs := makeMixedMessages()
	const needle = "nonexistent_needle_xyz"

	b.ResetTimer()
	for b.Loop() {
		_ = filterByContent(msgs, needle)
	}
}

func cloneMessages(src []*entities.Message) []*entities.Message {
	out := make([]*entities.Message, len(src))
	for i, m := range src {
		c := *m
		out[i] = &c
	}
	return out
}
