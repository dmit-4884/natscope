// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package bbolt

import "github.com/dmit-4884/natscope/internal/pkg/bbstore"

// historyDoc is the persistence model for a publish-history entry.
type historyDoc struct {
	bbstore.Base

	ConnectionID     *string `json:"connectionId,omitempty"`
	ConnectionURL    string  `json:"connectionUrl"`
	Stream           string  `json:"stream,omitempty"`
	Subject          string  `json:"subject"`
	SubjectPattern   *string `json:"subjectPattern,omitempty"`
	EncodingType     string  `json:"encodingType,omitempty"`
	MessageType      string  `json:"messageType,omitempty"`
	PayloadJSON      string  `json:"payloadJson,omitempty"`
	PayloadTruncated bool    `json:"payloadTruncated,omitempty"`
	PayloadSize      int     `json:"payloadSize,omitempty"`
	Sequence         *uint64 `json:"sequence,omitempty"`
	Success          bool    `json:"success,omitempty"`
	Error            *string `json:"error,omitempty"`
}
