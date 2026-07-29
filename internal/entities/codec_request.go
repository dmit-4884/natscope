// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

// CodecRequest is the unified input for proto Decode/Encode/Validate.
type CodecRequest struct {
	// Data carries protobuf wire bytes for Decode / Validate.
	Data []byte
	// JSON carries protojson bytes for Encode / EncodeWithValidation.
	JSON []byte
	// SourceID identifies the proto source whose snapshot to use. Required.
	SourceID string
	// Tag selects a version; empty falls back to the source's active selection (or
	// LocalTag for local sources).
	Tag string
	// MessageType is the fully-qualified name of the protobuf message type.
	MessageType string
}
