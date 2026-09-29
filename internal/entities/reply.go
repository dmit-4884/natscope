// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

import "time"

// Reply is the first message received in answer to a core NATS request.
type Reply struct {
	// Subject is the reply inbox the message was delivered on.
	Subject string

	// Data is the reply payload.
	Data []byte

	// Headers carries the responder's headers; nil when it sent none.
	Headers map[string]string

	// Duration is the time from sending the request to receiving the reply.
	Duration time.Duration
}
