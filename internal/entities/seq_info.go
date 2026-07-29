// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

// SequenceInfo is a delivery+stream sequence pair reported by JetStream.
type SequenceInfo struct {
	Consumer uint64
	Stream   uint64
	Last     *int64
}
