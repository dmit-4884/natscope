// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

// MessagesResponse is a page of messages fetched from a stream.
type MessagesResponse struct {
	// Messages is the list of messages.
	Messages []*Message

	// HasMore indicates if there are more messages available.
	HasMore bool

	// NextSeq is the next sequence number for pagination.
	NextSeq uint64
}
