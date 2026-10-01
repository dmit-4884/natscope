// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

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
}
