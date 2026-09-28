// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

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

	truncated := len(data) > detectScanLimit
	scan := data
	if truncated {
		scan = scan[:detectScanLimit]
	}

	// Try JSON only when first non-whitespace byte is { or [, so binary blobs
	// short-circuit before json.Valid walks the payload.
	for _, b := range scan {
		if b == ' ' || b == '\t' || b == '\n' || b == '\r' {
			continue
		}
		if b != '{' && b != '[' {
			break
		}
		// A complete scan can be validated outright. A truncated scan can
		// never be valid on its own — json.Valid always reports it as
		// invalid — so check instead whether it is a well-formed JSON
		// *prefix*: token-by-token parsing ran out of (truncated) bytes
		// mid-structure rather than hitting a real syntax error. Bounded to
		// detectScanLimit, so still cheap on multi-MB payloads (QA-082).
		if !truncated {
			if json.Valid(scan) {
				return ContentTypeJSON
			}
		} else if looksLikeJSONPrefix(scan) {
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

// looksLikeJSONPrefix reports whether scan is the start of a well-formed JSON
// document that simply ran out of (truncated) bytes, as opposed to bytes that
// happen to start with '{'/'[' but are not JSON. It tokenizes scan and
// accepts running out of input mid-token/mid-structure (io.EOF /
// io.ErrUnexpectedEOF) as "still plausibly JSON"; any other error means the
// prefix is genuinely malformed.
func looksLikeJSONPrefix(scan []byte) bool {
	dec := json.NewDecoder(bytes.NewReader(scan))
	for {
		if _, err := dec.Token(); err != nil {
			return errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF)
		}
	}
}
