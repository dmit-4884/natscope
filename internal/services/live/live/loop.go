// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package live

import (
	"bytes"
	"context"
	"encoding/json"
	"time"

	"github.com/dmit-4884/natscope/internal/entities"
)

// truncateLiveMessage caps the raw byte payload to maxBytes and drops any
// decoded JSON when it would exceed the cap, setting Truncated for the UI.
func truncateLiveMessage(lm *entities.LiveMessage, maxBytes int) {
	if maxBytes <= 0 {
		return
	}
	if len(lm.NatsMessage.Data) > maxBytes {
		lm.NatsMessage.Data = lm.NatsMessage.Data[:maxBytes]
		lm.Truncated = true
	}
	if lm.Decoded != nil && len(*lm.Decoded) > maxBytes {
		// Pretty-prints and truncates to N lines / a byte cap; full payload
		// is still reachable via MessagesService.Get(sequence) after pausing.
		preview := buildLivePreviewText(json.RawMessage(*lm.Decoded))
		lm.Decoded = &preview
		lm.Truncated = true
	}
}

const (
	livePreviewLines = 20
	livePreviewBytes = 2 * 1024
)

// buildLivePreviewText returns a JSON-encoded text preview so the live
// entity's Decoded stays valid JSON; frontend JSON.parses it uniformly.
func buildLivePreviewText(raw json.RawMessage) string {
	var indented bytes.Buffer
	var text string
	if err := json.Indent(&indented, raw, "", "  "); err != nil {
		text = clipLiveText(string(raw)) + "\n…"
	} else {
		text = indented.String()
		if idx := nthLineIndex(text, livePreviewLines); idx > 0 {
			text = text[:idx]
		}
		text = clipLiveText(text) + "\n…"
	}
	encoded, err := json.Marshal(text)
	if err != nil {
		return ""
	}
	return string(encoded)
}

func clipLiveText(s string) string {
	if len(s) <= livePreviewBytes {
		return s
	}
	return s[:livePreviewBytes]
}

func nthLineIndex(s string, n int) int {
	for i, count := 0, 0; i < len(s); i++ {
		if s[i] == '\n' {
			count++
			if count == n {
				return i
			}
		}
	}
	return -1
}

// runLoop is the core event loop; it rate-limits, decodes, and batches
// messages. ctx cancellation triggers a final flush before returning.
func (s *Service) runLoop(
	ctx context.Context,
	sess *sessionState,
	msgChan <-chan *entities.NatsMessage,
	maxDisplayRate int32,
	maxPayloadBytes int32,
	emit func(*entities.LiveEvent) error,
) error {
	batchTicker := time.NewTicker(batchInterval)
	defer batchTicker.Stop()
	statsTicker := time.NewTicker(statsInterval)
	defer statsTicker.Stop()

	batch := make([]*entities.LiveMessage, 0, maxBatchSize)

	var totalMessages int64
	var lastMsgCount int64
	lastTime := time.Now()
	warmup := true // skip first stats tick to avoid burst spike

	// Token-bucket state. Refilled at maxDisplayRate tokens/sec, capped at
	// maxDisplayRate (1-second burst).
	var tokenBucket float64
	var lastRefill time.Time
	if maxDisplayRate > 0 {
		tokenBucket = float64(maxDisplayRate)
		lastRefill = time.Now()
	}

	decoder := s.protoService.NewLiveDecoder()
	subjectCounts := make(map[string]int64)

	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		ev := &entities.LiveEvent{Batch: &entities.LiveBatch{Messages: batch}}
		batch = make([]*entities.LiveMessage, 0, maxBatchSize)
		return emit(ev)
	}

	for {
		select {
		case <-ctx.Done():
			_ = flush() //nolint:errcheck // best-effort flush during shutdown; peer disconnect is expected
			return nil

		case <-statsTicker.C:
			now := time.Now()
			current := totalMessages

			if warmup {
				warmup = false
				lastMsgCount = current
				lastTime = now
				continue
			}

			elapsed := now.Sub(lastTime).Seconds()
			var msgPerSec int64
			if elapsed > 0 {
				msgPerSec = int64(float64(current-lastMsgCount) / elapsed)
			}
			lastMsgCount = current
			lastTime = now

			ev := &entities.LiveEvent{
				Stats: &entities.LiveStats{
					TotalMessages:     current,
					MessagesPerSecond: msgPerSec,
					SubjectCounts:     subjectCounts,
					MessagesDropped:   sess.messagesDropped.Load(),
				},
			}
			if err := emit(ev); err != nil {
				return err
			}

		case <-batchTicker.C:
			if err := flush(); err != nil {
				return err
			}

		case msg := <-msgChan:
			totalMessages++
			if _, tracked := subjectCounts[msg.Subject]; tracked || len(subjectCounts) < maxSubjectCardinality {
				subjectCounts[msg.Subject]++
			}

			if maxDisplayRate > 0 {
				now := time.Now()
				elapsed := now.Sub(lastRefill).Seconds()
				lastRefill = now
				tokenBucket += elapsed * float64(maxDisplayRate)
				maxTokens := float64(maxDisplayRate)
				if tokenBucket > maxTokens {
					tokenBucket = maxTokens
				}
				if tokenBucket < 1.0 {
					sess.messagesDropped.Add(1)
					continue
				}
				tokenBucket -= 1.0
			}

			// Capture the pre-truncate byte count; the converter reads OriginalSize
			// so wire DataSize reflects what was published, not the truncated slice.
			lm := &entities.LiveMessage{NatsMessage: *msg, OriginalSize: len(msg.Data)}

			if sess.decoderDirty.CompareAndSwap(1, 0) {
				decoder.Reset()
			}
			if !decoder.Ready() {
				decoder.Init(ctx)
			}
			// Decode runs on the full payload (protobuf can't be partial-decoded),
			// then both the byte view and decoded JSON are truncated together.
			decoded, decodedType, decodeErr := decoder.Decode(ctx, msg.Data, msg.Subject)
			if len(decoded) > 0 {
				ds := string(decoded)
				lm.Decoded = &ds
				lm.DecodedType = &decodedType
			} else if decodeErr != "" {
				lm.DecodeError = &decodeErr
			}
			truncateLiveMessage(lm, int(maxPayloadBytes))

			batch = append(batch, lm)
			if len(batch) >= maxBatchSize {
				if err := flush(); err != nil {
					return err
				}
			}
		}
	}
}
