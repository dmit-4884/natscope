// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContentType_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ct   ContentType
		want bool
	}{
		{name: "Binary", ct: ContentTypeBinary, want: true},
		{name: "JSON", ct: ContentTypeJSON, want: true},
		{name: "Text", ct: ContentTypeText, want: true},
		{name: "Empty", ct: "", want: false},
		{name: "Unknown", ct: "application/xml", want: false},
		{name: "CaseSensitive_BINARY", ct: "BINARY", want: false},
		{name: "CaseSensitive_Json", ct: "Json", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.ct.IsValid())
		})
	}
}

func TestDetectContentType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data []byte
		want ContentType
	}{
		{name: "EmptyData", data: []byte{}, want: ContentTypeText},
		{name: "NilData", data: nil, want: ContentTypeText},
		{name: "ValidJSONObject", data: []byte(`{"key": "value"}`), want: ContentTypeJSON},
		{name: "ValidJSONArray", data: []byte(`[1, 2, 3]`), want: ContentTypeJSON},
		{name: "JSONWithLeadingWhitespace", data: []byte(`  { "a": 1 }`), want: ContentTypeJSON},
		{name: "JSONWithTabs", data: []byte("\t{\"a\":1}"), want: ContentTypeJSON},
		{name: "JSONWithNewlines", data: []byte("\n\n{\"a\":1}"), want: ContentTypeJSON},
		{name: "InvalidJSONStartsWithBrace", data: []byte(`{not json`), want: ContentTypeText},
		{name: "InvalidJSONStartsWithBracket", data: []byte(`[not json`), want: ContentTypeText},
		{name: "PlainText", data: []byte("hello world"), want: ContentTypeText},
		{name: "TextWithNewlines", data: []byte("line1\nline2\r\nline3"), want: ContentTypeText},
		{name: "TextWithTabs", data: []byte("col1\tcol2"), want: ContentTypeText},
		{name: "BinaryWithNullByte", data: []byte{0x00}, want: ContentTypeBinary},
		{name: "BinaryWithControlChars", data: []byte{0x01, 0x02, 0x03}, want: ContentTypeBinary},
		{name: "BinaryMixed", data: []byte("hello\x00world"), want: ContentTypeBinary},
		{name: "BinaryHighBytes", data: []byte{0xFF, 0xFE, 0x01}, want: ContentTypeBinary},
		{name: "NumbersAsText", data: []byte("12345"), want: ContentTypeText},
		{name: "NestedJSON", data: []byte(`{"a": {"b": [1, 2]}}`), want: ContentTypeJSON},
		{name: "OnlyWhitespaceThenBrace", data: []byte(" \t\n\r{\"x\":1}"), want: ContentTypeJSON},
		{name: "TextStartsWithLetter", data: []byte("abc"), want: ContentTypeText},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := DetectContentType(tt.data)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestDetectContentType_LargeJSON (QA-082) is the exact repro: a valid JSON
// document longer than detectScanLimit must still classify as JSON, not text
// — json.Valid on a truncated prefix is always false.
func TestDetectContentType_LargeJSON(t *testing.T) {
	t.Parallel()

	data := []byte(`{"k":"` + strings.Repeat("v", 5000) + `"}`)
	assert.Greater(t, len(data), detectScanLimit)

	got := DetectContentType(data)
	assert.Equal(t, ContentTypeJSON, got)
}

// TestDetectContentType_LargeJSONArray covers the "[" branch and a value
// (not just a string) straddling the scan boundary.
func TestDetectContentType_LargeJSONArray(t *testing.T) {
	t.Parallel()

	items := make([]int, 2000)
	for i := range items {
		items[i] = i
	}
	data, err := json.Marshal(items)
	assert.NoError(t, err)
	assert.Greater(t, len(data), detectScanLimit)

	assert.Equal(t, ContentTypeJSON, DetectContentType(data))
}

// TestDetectContentType_LargeGarbageNotMisclassifiedAsJSON ensures the
// truncated-prefix heuristic doesn't turn into "anything starting with { is
// JSON": a real syntax error inside the scanned prefix must still fail.
func TestDetectContentType_LargeGarbageNotMisclassifiedAsJSON(t *testing.T) {
	t.Parallel()

	data := []byte(`{"k": not-a-valid-token-at-all, ` + strings.Repeat("x", 5000))
	assert.Greater(t, len(data), detectScanLimit)

	assert.Equal(t, ContentTypeText, DetectContentType(data))
}
