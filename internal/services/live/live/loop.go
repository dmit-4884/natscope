// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package live

import (
	"bytes"
	"context"
	"encoding/json"
	"slices"
	"time"

	"github.com/altessa-s/go-atlas/core/runtime/panics"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/natsutil"
)

// truncateLiveMessage caps the raw byte payload to maxBytes and drops any
// decoded JSON when it would exceed the cap, setting Truncated for the UI.
func truncateLiveMessage(lm *entities.LiveMessage, maxBytes int) {
	if maxBytes <= 0 {
		return
	}
	if len(lm.NatsMessage.Data) > maxBytes {
		lm.NatsMessage.Data = bytes.Clone(lm.NatsMessage.Data[:maxBytes])
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

	// emitStallTimeout bounds a single emit (stream.Send) to a client that stopped reading.
	emitStallTimeout = 10 * time.Second
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

// emitWithDeadline runs emit on a goroutine and gives up after emitStallTimeout.
func emitWithDeadline(ctx context.Context, emit func(*entities.LiveEvent) error, ev *entities.LiveEvent) error {
	done := make(chan error, 1)
	go func() {
		defer panics.Handle(ctx)
		done <- emit(ev)
	}()
	select {
	case err := <-done:
		return err
	case <-time.After(emitStallTimeout):
		return errs.ErrLiveConsumerStalled
	}
}

type loopLimits struct {
	maxDisplayRate  int32
	maxPayloadBytes int32
	detect          bool
	exclude         []string
}

func (l loopLimits) excludes(subject string) bool {
	return slices.ContainsFunc(l.exclude, func(pattern string) bool { return natsutil.MatchSubject(pattern, subject) })
}

// runLoop is the core event loop; it rate-limits, decodes, and batches
// messages. ctx cancellation triggers a final flush before returning.
func (s *Service) runLoop(
	ctx context.Context,
	sess *sessionState,
	msgChan <-chan *entities.NatsMessage,
	limits loopLimits,
	emit func(*entities.LiveEvent) error,
) error {
	maxDisplayRate, maxPayloadBytes := limits.maxDisplayRate, limits.maxPayloadBytes
	batchTicker := time.NewTicker(batchInterval)
	defer batchTicker.Stop()
	statsTicker := time.NewTicker(statsInterval)
	defer statsTicker.Stop()

	batch := make([]*entities.LiveMessage, 0, maxBatchSize)

	var lastMsgCount, muted int64
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

	decoder := s.protoService.NewLiveDecoder(limits.detect)
	subjectCounts := make(map[string]int64)

	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		ev := &entities.LiveEvent{Batch: &entities.LiveBatch{Messages: batch}}
		batch = make([]*entities.LiveMessage, 0, maxBatchSize)
		return emitWithDeadline(ctx, emit, ev)
	}

	// resetDecoderIfDirty reloads the decoder after a proto reload and notifies the client.
	// It runs per message and per stats tick.
	resetDecoderIfDirty := func() error {
		if !sess.decoderDirty.CompareAndSwap(1, 0) {
			return nil
		}
		decoder.Reset()
		return emitWithDeadline(ctx, emit, &entities.LiveEvent{ProtoReload: &entities.LiveProtoReload{}})
	}

	for {
		select {
		case <-ctx.Done():
			_ = flush() //nolint:errcheck // best-effort flush during shutdown; peer disconnect is expected
			return nil

		case <-sess.lost:
			_ = flush() //nolint:errcheck // best-effort flush
			return errs.ErrLiveConnectionLost

		case denied := <-sess.denials:
			if err := emitWithDeadline(ctx, emit, &entities.LiveEvent{Error: denied}); err != nil {
				return err
			}

		case <-statsTicker.C:
			if err := resetDecoderIfDirty(); err != nil {
				return err
			}

			now := time.Now()
			current := sess.totalMessages.Load() - muted

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
			if err := emitWithDeadline(ctx, emit, ev); err != nil {
				return err
			}

		case <-batchTicker.C:
			if err := flush(); err != nil {
				return err
			}

		case msg := <-msgChan:
			sess.bufferedBytes.Add(-int64(len(msg.Data)))
			if _, tracked := subjectCounts[msg.Subject]; tracked || len(subjectCounts) < maxSubjectCardinality {
				subjectCounts[msg.Subject]++
			}
			if limits.excludes(msg.Subject) {
				muted++
				continue
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

			if err := resetDecoderIfDirty(); err != nil {
				return err
			}
			if !decoder.Ready() {
				decoder.Init(ctx)
			}
			// Decode runs on the full payload (protobuf can't be partial-decoded),
			// then both the byte view and decoded JSON are truncated together.
			switch r := decoder.Decode(ctx, msg.Data, msg.Subject); {
			case r == nil:
			case r.Success:
				ds := string(r.Decoded)
				lm.Decoded, lm.DecodedType, lm.DecodedAuto = &ds, &r.MessageType, r.Auto
				if r.Auto {
					lm.DecodedSourceID = &r.SourceID
				}
			case r.Error != "":
				lm.DecodeError = &r.Error
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
