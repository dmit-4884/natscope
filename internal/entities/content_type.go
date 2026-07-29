// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

import "encoding/json"

// ContentType is the encoding classification of a message payload.
type ContentType string

// Content type constants.
const (
	ContentTypeBinary ContentType = "binary"
	ContentTypeJSON   ContentType = "json"
	ContentTypeText   ContentType = "text"
)

// IsValid checks if the content type is a known value.
func (ct ContentType) IsValid() bool {
	switch ct {
	case ContentTypeBinary, ContentTypeJSON, ContentTypeText:
		return true
	default:
		return false
	}
}

// detectScanLimit caps classification scan; 4KB spots JSON
// prefix/non-printables, full-payload scan dominated CPU on multi-MB messages.
const detectScanLimit = 4 * 1024

// DetectContentType classifies data as JSON/text/binary by scanning the first
// detectScanLimit bytes (heuristic, not a validator).
func DetectContentType(data []byte) ContentType {
	if len(data) == 0 {
		return ContentTypeBinary
	}

	scan := data
	if len(scan) > detectScanLimit {
		scan = scan[:detectScanLimit]
	}

	// Try JSON only when first non-whitespace byte is { or [, so binary blobs
	// short-circuit before json.Valid walks the payload.
	for _, b := range scan {
		if b == ' ' || b == '\t' || b == '\n' || b == '\r' {
			continue
		}
		if (b == '{' || b == '[') && json.Valid(scan) {
			return ContentTypeJSON
		}
		break
	}

	// Check readable text, capped to detectScanLimit to bound branch cost on large
	// payloads.
	for _, b := range scan {
		if b < 32 && b != '\n' && b != '\r' && b != '\t' {
			return ContentTypeBinary
		}
		if b == 0 {
			return ContentTypeBinary
		}
	}
	return ContentTypeText
}
