// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"github.com/nats-io/nats.go/jetstream"
)

// buildSequenceList walks from startSeq in the given direction up to maxCount
// entries; caller bounds startSeq within [FirstSeq, LastSeq].
func buildSequenceList(startSeq uint64, info *jetstream.StreamInfo, direction string, maxCount int) []uint64 {
	var seqs []uint64

	if direction == DefaultDirection {
		for range maxCount {
			if startSeq < info.State.FirstSeq {
				break
			}
			seqs = append(seqs, startSeq)
			if startSeq == 0 {
				break
			}
			startSeq--
		}
	} else {
		for range maxCount {
			if startSeq > info.State.LastSeq {
				break
			}
			seqs = append(seqs, startSeq)
			startSeq++
		}
	}

	return seqs
}

// hasMoreMessages reports whether more messages exist past lastSeq in the given
// direction.
func hasMoreMessages(direction string, lastSeq uint64, info *jetstream.StreamInfo) bool {
	if direction == DefaultDirection {
		return lastSeq > info.State.FirstSeq
	}
	return lastSeq < info.State.LastSeq
}

// calculateNextSeq returns the next-page cursor sequence, or 0 when there's no
// more data in this direction.
func calculateNextSeq(hasMore bool, direction string, lastSeq uint64, info *jetstream.StreamInfo) uint64 {
	if !hasMore {
		return 0
	}

	if direction == DefaultDirection {
		if lastSeq > info.State.FirstSeq {
			return lastSeq - 1
		}
	} else {
		if lastSeq < info.State.LastSeq {
			return lastSeq + 1
		}
	}

	return 0
}
