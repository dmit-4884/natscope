// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

// BatchDecodeItem is a single item for batch decoding.
type BatchDecodeItem struct {
	// Data is the raw protobuf bytes.
	Data []byte

	// MessageType is the fully qualified protobuf message type.
	MessageType string
}
