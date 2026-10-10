// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package entities

import "encoding/json"

// DecodeResult is the result of decoding a protobuf message; a failed decode keeps Decoded for the first ValidBytes.
type DecodeResult struct {
	Success       bool
	Decoded       json.RawMessage
	FormattedJSON string
	Error         string
	MessageType   string
	UnknownFields []UnknownField
	ValidBytes    int
	// SourceID and Auto are set when the type came from auto-detection rather than a mapping.
	SourceID string
	Auto     bool
}

// TypeCandidate is a message type a payload decodes as; Score runs from 0 to 100.
type TypeCandidate struct {
	SourceID       string
	SourceRevision string
	MessageType    string
	Score          int
	UnknownBytes   int
	Decoded        json.RawMessage
}
