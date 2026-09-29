// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

import "time"

// RequestMessage is the request service input: a publish payload sent as a
// core NATS request, answered by the first reply within Timeout.
type RequestMessage struct {
	PublishRequest

	// Timeout bounds the wait for the reply; zero falls back to the service
	// default.
	Timeout time.Duration
}
