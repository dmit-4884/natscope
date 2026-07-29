// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"crypto/rand"
	"encoding/base64"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// BenchmarkToMessage_1_6MB measures the per-message cost of converting a raw
// JetStream message to entities.Message — dominated by base64 encoding.
func BenchmarkToMessage_1_6MB(b *testing.B) {
	payload := make([]byte, 1600*1024)
	_, _ = rand.Read(payload)

	msg := &jetstream.RawStreamMsg{
		Subject:  "test.subject.bench",
		Sequence: 42,
		Time:     time.Now(),
		Data:     payload,
		Header:   nil,
	}

	b.ResetTimer()
	b.SetBytes(int64(len(payload)))
	for b.Loop() {
		_ = toMessage(msg)
	}
}

// BenchmarkToMessage_4KB measures the same path for tiny payloads — sanity
// baseline showing per-msg overhead independent of payload size.
func BenchmarkToMessage_4KB(b *testing.B) {
	payload := make([]byte, 4*1024)
	_, _ = rand.Read(payload)

	msg := &jetstream.RawStreamMsg{
		Subject:  "test.subject.bench",
		Sequence: 42,
		Time:     time.Now(),
		Data:     payload,
	}

	b.ResetTimer()
	b.SetBytes(int64(len(payload)))
	for b.Loop() {
		_ = toMessage(msg)
	}
}

// BenchmarkBase64StdLib_1_6MB isolates just the stdlib base64 encode step at
// 1.6 MB to separate codec/converter overhead from raw encoding cost.
func BenchmarkBase64StdLib_1_6MB(b *testing.B) {
	payload := make([]byte, 1600*1024)
	_, _ = rand.Read(payload)

	b.ResetTimer()
	b.SetBytes(int64(len(payload)))
	for b.Loop() {
		_ = base64.StdEncoding.EncodeToString(payload)
	}
}

// BenchmarkToMessage_5MB measures a single large message (top of user's range).
func BenchmarkToMessage_5MB(b *testing.B) {
	payload := make([]byte, 5*1024*1024)
	_, _ = rand.Read(payload)

	msg := &jetstream.RawStreamMsg{
		Subject:  "test.subject.bench",
		Sequence: 42,
		Time:     time.Now(),
		Data:     payload,
	}

	b.ResetTimer()
	b.SetBytes(int64(len(payload)))
	for b.Loop() {
		_ = toMessage(msg)
	}
}

// BenchmarkToMessage_Batch23x1_6MB simulates the full BIG_TEST-style fetch:
// 23 messages × 1.6 MB each, as in the production getMessagesParallel loop.
func BenchmarkToMessage_Batch23x1_6MB(b *testing.B) {
	const count = 23
	msgs := make([]*jetstream.RawStreamMsg, count)
	for i := range msgs {
		payload := make([]byte, 1600*1024)
		_, _ = rand.Read(payload)
		msgs[i] = &jetstream.RawStreamMsg{
			Subject:  "test.subject.bench",
			Sequence: uint64(i + 1),
			Time:     time.Now(),
			Data:     payload,
		}
	}

	b.ResetTimer()
	b.SetBytes(int64(count) * 1600 * 1024)
	for b.Loop() {
		for _, m := range msgs {
			_ = toMessage(m)
		}
	}
}

// BenchmarkToMessage_BatchMixed simulates a realistic stream: mix of small
// to huge messages, closer to production data than a uniform stream.
func BenchmarkToMessage_BatchMixed(b *testing.B) {
	sizes := []int{
		4 * 1024, 4 * 1024, 4 * 1024, 4 * 1024, 4 * 1024,
		100 * 1024, 100 * 1024, 100 * 1024,
		1600 * 1024, 1600 * 1024, 1600 * 1024, 1600 * 1024,
		5 * 1024 * 1024, 5 * 1024 * 1024,
	}
	msgs := make([]*jetstream.RawStreamMsg, len(sizes))
	var totalBytes int64
	for i, sz := range sizes {
		payload := make([]byte, sz)
		_, _ = rand.Read(payload)
		msgs[i] = &jetstream.RawStreamMsg{
			Subject:  "test.subject.bench",
			Sequence: uint64(i + 1),
			Time:     time.Now(),
			Data:     payload,
		}
		totalBytes += int64(sz)
	}

	b.ResetTimer()
	b.SetBytes(totalBytes)
	for b.Loop() {
		for _, m := range msgs {
			_ = toMessage(m)
		}
	}
}
