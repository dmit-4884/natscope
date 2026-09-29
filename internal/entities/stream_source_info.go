// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

import "time"

// StreamSourceInfo is the live state of a mirror or source link, as reported by the downstream stream.
type StreamSourceInfo struct {
	// Name is the upstream stream name.
	Name string

	// External is set when the upstream lives in another account or JetStream domain.
	External *ExternalStreamRef

	// Lag is how many messages the link is behind the upstream; accurate only while the link is active.
	Lag uint64

	// Active is the time since the upstream was last heard from; negative when it never was.
	Active time.Duration

	// FilterSubject is the subject filter of the link.
	FilterSubject string

	// SubjectTransforms are the filters and transforms of the link; an empty destination means untransformed.
	SubjectTransforms []SubjectTransformConfig

	// Error is the last error the server hit setting the link up; empty while the link is healthy.
	Error string
}
