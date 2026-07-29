// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package messages

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dmit-4884/natscope/internal/entities"
)

// --- Tests ---

func TestFilterByContent(t *testing.T) {
	t.Parallel()

	jsonData := `{"order_id": "abc123", "amount": 42.5}`
	jsonBase64 := base64.StdEncoding.EncodeToString([]byte(jsonData))

	textData := "Hello World this is a test message"
	textBase64 := base64.StdEncoding.EncodeToString([]byte(textData))

	decodedJSON := json.RawMessage(`{"user_name": "JohnDoe", "email": "john@example.com"}`)

	tests := []struct {
		name     string
		messages []*entities.Message
		filter   string
		wantLen  int
	}{
		{
			name:     "EmptyFilter_ReturnsAll",
			messages: []*entities.Message{{DataBase64: jsonBase64}, {DataBase64: textBase64}},
			filter:   "",
			wantLen:  2,
		},
		{
			name: "MatchInDecodedJSON",
			messages: []*entities.Message{
				{Decoded: decodedJSON, DataBase64: textBase64},
				{DataBase64: textBase64},
			},
			filter:  "johndoe",
			wantLen: 1,
		},
		{
			name: "MatchInRawJSON",
			messages: []*entities.Message{
				{DataBase64: jsonBase64},
				{DataBase64: textBase64},
			},
			filter:  "abc123",
			wantLen: 1,
		},
		{
			name: "MatchInRawText",
			messages: []*entities.Message{
				{DataBase64: jsonBase64},
				{DataBase64: textBase64},
			},
			filter:  "hello world",
			wantLen: 1,
		},
		{
			name: "CaseInsensitive",
			messages: []*entities.Message{
				{DataBase64: jsonBase64},
			},
			filter:  "ABC123",
			wantLen: 1,
		},
		{
			name: "NoMatch",
			messages: []*entities.Message{
				{DataBase64: jsonBase64},
				{DataBase64: textBase64},
			},
			filter:  "nonexistent_value_xyz",
			wantLen: 0,
		},
		{
			name:     "NilMessages_ReturnsNil",
			messages: nil,
			filter:   "test",
			wantLen:  0,
		},
		{
			name: "DecodedTakesPriority",
			messages: []*entities.Message{
				{Decoded: json.RawMessage(`{"secret": "found_in_decoded"}`), DataBase64: textBase64},
			},
			filter:  "found_in_decoded",
			wantLen: 1,
		},
		{
			name: "MatchMultiple",
			messages: []*entities.Message{
				{DataBase64: base64.StdEncoding.EncodeToString([]byte("test message one"))},
				{DataBase64: base64.StdEncoding.EncodeToString([]byte("another thing"))},
				{DataBase64: base64.StdEncoding.EncodeToString([]byte("test message two"))},
			},
			filter:  "test message",
			wantLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := filterByContent(tt.messages, tt.filter)
			assert.Len(t, result, tt.wantLen)
		})
	}
}

func TestMatchesContent(t *testing.T) {
	t.Parallel()

	t.Run("EmptyDataBase64", func(t *testing.T) {
		t.Parallel()
		msg := &entities.Message{DataBase64: ""}
		assert.False(t, matchesContent(msg, "test"))
	})

	t.Run("InvalidBase64", func(t *testing.T) {
		t.Parallel()
		msg := &entities.Message{DataBase64: "not-valid-base64!!!"}
		assert.False(t, matchesContent(msg, "test"))
	})

	t.Run("BinaryData_NoMatch", func(t *testing.T) {
		t.Parallel()
		binaryData := []byte{0x00, 0x01, 0x02, 0xFF}
		msg := &entities.Message{DataBase64: base64.StdEncoding.EncodeToString(binaryData)}
		assert.False(t, matchesContent(msg, "hello"))
	})

	t.Run("MatchInDecodedOnly", func(t *testing.T) {
		t.Parallel()
		msg := &entities.Message{
			Decoded:    json.RawMessage(`{"key": "uniquevalue42"}`),
			DataBase64: base64.StdEncoding.EncodeToString([]byte("unrelated")),
		}
		assert.True(t, matchesContent(msg, "uniquevalue42"))
	})

	t.Run("MatchInRawOnly", func(t *testing.T) {
		t.Parallel()
		msg := &entities.Message{
			DataBase64: base64.StdEncoding.EncodeToString([]byte(`{"raw_key": "raw_value"}`)),
		}
		assert.True(t, matchesContent(msg, "raw_value"))
	})
}
