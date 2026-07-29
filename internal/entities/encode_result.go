// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

// EncodeResult is the result of encoding a JSON message to protobuf.
type EncodeResult struct {
	Success    bool
	DataBase64 string
	DataSize   int
	Error      string
}
