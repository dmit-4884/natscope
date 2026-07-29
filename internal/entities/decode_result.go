// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

import "encoding/json"

// DecodeResult is the result of decoding a protobuf message.
type DecodeResult struct {
	Success       bool
	Decoded       json.RawMessage
	FormattedJSON string
	Error         string
	MessageType   string
}
