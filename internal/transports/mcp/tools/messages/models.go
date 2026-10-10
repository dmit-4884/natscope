// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package messages

import (
	"encoding/json"
	"time"

	mcptransport "github.com/dmit-4884/natscope/internal/transports/mcp"
)

type messageView struct {
	Sequence    uint64             `json:"seq"`
	Subject     string             `json:"subject"`
	Timestamp   time.Time          `json:"time"`
	Headers     map[string]string  `json:"headers,omitempty"`
	DataSize    int                `json:"size" jsonschema:"payload size in bytes"`
	DecodedType string             `json:"decodedType,omitempty" jsonschema:"Protobuf message type the payload was decoded as"`
	DecodedAuto bool               `json:"decodedAuto,omitempty" jsonschema:"no mapping matched the subject; natscope detected decodedType"`
	Decoded     json.RawMessage    `json:"decoded,omitempty" jsonschema:"payload decoded from Protobuf to JSON"`
	DecodeError string             `json:"decodeError,omitempty"`
	Body        *mcptransport.Body `json:"body,omitempty" jsonschema:"raw payload, present when there is no decoded form"`
	Truncated   bool               `json:"truncated,omitempty" jsonschema:"payload was clipped to the byte budget; get_message returns more"`
}

type findMessagesInput struct {
	mcptransport.ConnectionArg
	Stream          string `json:"stream" jsonschema:"stream name" normalize:"trim"`
	Subject         string `json:"subject,omitempty" jsonschema:"subject filter; NATS wildcards * and > are allowed" normalize:"trim"`
	StartSeq        uint64 `json:"startSeq,omitempty" jsonschema:"sequence to start from"`
	Since           string `json:"since,omitempty" jsonschema:"start at the first message published at or after this time: RFC 3339 or a duration ago like 15m"`
	Direction       string `json:"direction,omitempty" jsonschema:"backward (newest first) or forward, default with startSeq/since" normalize:"trim,lowercase"`
	Limit           int    `json:"limit,omitempty" jsonschema:"messages per page, 1-100 (default 20)"`
	Contains        string `json:"contains,omitempty" jsonschema:"case-insensitive payload text; searches the stream, 100000 messages per call" normalize:"trim"`
	Regex           bool   `json:"regex,omitempty" jsonschema:"treat contains as an RE2 regular expression, matched as written"`
	Header          string `json:"header,omitempty" jsonschema:"header the message must carry, as name or name=value; searches like contains" normalize:"trim"`
	MaxPayloadBytes int    `json:"maxPayloadBytes,omitempty" jsonschema:"per-message payload budget in bytes (default 4096); a page carries at most 256 KiB"`
}

type findMessagesOutput struct {
	Messages []messageView `json:"messages"`
	HasMore  bool          `json:"hasMore"`
	NextSeq  uint64        `json:"nextSeq,omitempty" jsonschema:"pass as startSeq with the same direction for the next page"`
	Scanned  uint64        `json:"scanned,omitempty" jsonschema:"messages the server read for a contains or header search"`
}

type getMessageInput struct {
	mcptransport.ConnectionArg
	Stream          string `json:"stream" jsonschema:"stream name" normalize:"trim"`
	Seq             uint64 `json:"seq" jsonschema:"stream sequence number"`
	MaxPayloadBytes int    `json:"maxPayloadBytes,omitempty" jsonschema:"payload budget in bytes (default 65536, max 262144)"`
}

type tailInput struct {
	mcptransport.ConnectionArg
	Subject         string `json:"subject" jsonschema:"subject to listen on; NATS wildcards * and > are allowed" normalize:"trim"`
	Stream          string `json:"stream,omitempty" jsonschema:"read new messages through this JetStream stream, not a core NATS subscription" normalize:"trim"`
	Seconds         int    `json:"seconds,omitempty" jsonschema:"how long to listen, 1-30 (default 10)"`
	MaxMessages     int    `json:"maxMessages,omitempty" jsonschema:"stop after this many messages, 1-200 (default 20)"`
	MaxPayloadBytes int    `json:"maxPayloadBytes,omitempty" jsonschema:"per-message payload budget in bytes (default 4096); a tail carries at most 256 KiB"`
}

type liveMessageView struct {
	Subject     string              `json:"subject"`
	Stream      string              `json:"stream,omitempty"`
	Sequence    *uint64             `json:"seq,omitempty"`
	Timestamp   *time.Time          `json:"time,omitempty"`
	Headers     map[string][]string `json:"headers,omitempty"`
	Size        int                 `json:"size" jsonschema:"payload size in bytes"`
	DecodedType string              `json:"decodedType,omitempty"`
	DecodedAuto bool                `json:"decodedAuto,omitempty"`
	Decoded     json.RawMessage     `json:"decoded,omitempty"`
	DecodeError string              `json:"decodeError,omitempty"`
	Body        *mcptransport.Body  `json:"body,omitempty"`
	Truncated   bool                `json:"truncated,omitempty" jsonschema:"payload was clipped; with stream and seq set, get_message returns it"`
}

type tailOutput struct {
	Messages  []liveMessageView `json:"messages"`
	StoppedBy string            `json:"stoppedBy" jsonschema:"timeout or maxMessages"`
	Dropped   int64             `json:"dropped,omitempty" jsonschema:"messages the server-side rate limit skipped"`
	Errors    []string          `json:"errors,omitempty"`
}
