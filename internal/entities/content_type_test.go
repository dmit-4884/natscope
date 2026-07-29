// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

import (
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
		{name: "EmptyData", data: []byte{}, want: ContentTypeBinary},
		{name: "NilData", data: nil, want: ContentTypeBinary},
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
