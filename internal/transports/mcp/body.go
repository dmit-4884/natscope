// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package mcptransport

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"unicode/utf8"
)

// NewBody renders data clipped to limit bytes (0 = unlimited); truncated marks data already clipped upstream.
// It reports whether the rendered body is incomplete.
func NewBody(data []byte, limit int, truncated bool) (*Body, bool) {
	if limit > 0 && len(data) > limit {
		data, truncated = data[:limit], true
	}
	if len(data) == 0 {
		return nil, truncated
	}

	trimmed := bytes.TrimSpace(data)
	isContainer := len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[')
	if !truncated && isContainer && json.Valid(trimmed) {
		return &Body{JSON: trimmed}, false
	}

	text := data
	if truncated {
		text = trimPartialRune(data)
	}
	if utf8.Valid(text) {
		return &Body{Text: string(text)}, truncated
	}
	return &Body{Base64: base64.StdEncoding.EncodeToString(data)}, truncated
}

// NewBodyFromBase64 renders a base64-encoded payload like NewBody; undecodable input is passed through as base64.
func NewBodyFromBase64(encoded string, limit int, truncated bool) (*Body, bool) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return &Body{Base64: encoded}, truncated
	}
	return NewBody(data, limit, truncated)
}

func trimPartialRune(b []byte) []byte {
	for i := 0; i < utf8.UTFMax-1 && len(b) > 0 && !utf8.Valid(b); i++ {
		b = b[:len(b)-1]
	}
	return b
}
