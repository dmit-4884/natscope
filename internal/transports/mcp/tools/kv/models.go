// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package kv

import (
	"encoding/json"
	"time"

	mcptransport "github.com/dmit-4884/natscope/internal/transports/mcp"
)

type bucketView struct {
	Bucket       string            `json:"bucket"`
	Description  string            `json:"description,omitempty"`
	Values       uint64            `json:"values"`
	Bytes        uint64            `json:"bytes"`
	History      uint8             `json:"history"`
	TTL          string            `json:"ttl,omitempty"`
	Storage      string            `json:"storage"`
	Replicas     int               `json:"replicas"`
	IsCompressed bool              `json:"compressed,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

type listBucketsOutput struct {
	Buckets []bucketView `json:"buckets"`
}

type bucketInput struct {
	mcptransport.ConnectionArg
	Bucket string `json:"bucket" jsonschema:"bucket name"`
}

type listKeysInput struct {
	bucketInput
	Pattern string `json:"pattern,omitempty" jsonschema:"key pattern; NATS wildcards * and > are allowed"`
	Limit   int    `json:"limit,omitempty" jsonschema:"maximum keys to return, 1-5000 (default 200)"`
}

type listKeysOutput struct {
	Keys      []string `json:"keys"`
	Truncated bool     `json:"truncated" jsonschema:"more keys match than the limit allowed; narrow the pattern or raise the limit"`
}

type entryView struct {
	Key         string             `json:"key"`
	Revision    uint64             `json:"revision"`
	Created     time.Time          `json:"created"`
	Operation   string             `json:"operation" jsonschema:"put, delete or purge"`
	TTL         string             `json:"ttl,omitempty" jsonschema:"how long this revision lives after created; absent when it does not expire on its own"`
	DecodedType string             `json:"decodedType,omitempty" jsonschema:"Protobuf message type the value was decoded as"`
	DecodedAuto bool               `json:"decodedAuto,omitempty" jsonschema:"no mapping matched; natscope detected decodedType"`
	Decoded     json.RawMessage    `json:"decoded,omitempty" jsonschema:"value decoded from Protobuf to JSON"`
	DecodeError string             `json:"decodeError,omitempty"`
	Value       *mcptransport.Body `json:"value,omitempty" jsonschema:"raw value, present when there is no decoded form"`
	Truncated   bool               `json:"truncated,omitempty"`
}

type keyInput struct {
	bucketInput
	Key           string `json:"key" jsonschema:"key name"`
	MaxValueBytes int    `json:"maxValueBytes,omitempty" jsonschema:"value budget in bytes (default 16384, max 1048576)"`
}

type historyInput struct {
	keyInput
	Limit int `json:"limit,omitempty" jsonschema:"revisions to return, newest first, 1-100 (default 20)"`
}

type historyOutput struct {
	Entries []entryView `json:"entries"`
	Total   int         `json:"total" jsonschema:"stored revisions"`
}
