// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

import "time"

// ScanOptions selects the stored messages a scan reads, oldest first.
type ScanOptions struct {
	// SubjectFilter narrows the scan to matching subjects; wildcards allowed, empty reads every subject.
	SubjectFilter string

	// FromSeq and ToSeq bound the scan, both inclusive.
	FromSeq uint64
	ToSeq   uint64

	// FetchMethod is "consumer" (a short-lived consumer) or "direct" (single-message reads).
	FetchMethod string
}

// MessageSearchRequest is the input for a budgeted search through a stream.
type MessageSearchRequest struct {
	ConnectionID  string
	StreamName    string
	SubjectFilter string

	// Direction is "forward" or "backward"; empty uses the user's default.
	Direction string

	// FromSeq, ToSeq, FromTime and ToTime bound the search; ToTime is inclusive.
	FromSeq  *uint64
	ToSeq    *uint64
	FromTime *time.Time
	ToTime   *time.Time

	// CursorSeq resumes a search where a previous run stopped.
	CursorSeq *uint64

	// Text is what the payload must contain, case-insensitive; with Regex it is an RE2 expression.
	Text  string
	Regex bool

	// HeaderName is a header the message must carry; HeaderValue, when set, is the value it must have.
	HeaderName  string
	HeaderValue string

	// MaxPayloadBytes caps each returned payload; nil uses the user's list setting.
	MaxPayloadBytes *int32

	// MaxMatches ends a run after this many matches; zero or more than one run allows uses the run limit.
	MaxMatches int
}

// SearchStopReason says why a search run ended.
type SearchStopReason int

const (
	SearchStopUnspecified SearchStopReason = iota
	SearchStopComplete
	SearchStopScanLimit
	SearchStopTimeLimit
	SearchStopMatchLimit
)

// MessageSearchProgress reports how far a running search got.
type MessageSearchProgress struct {
	Scanned    uint64
	Matched    uint64
	CurrentSeq uint64
	RangeFirst uint64
	RangeLast  uint64

	// ResumeSeq is where a search stopped now continues without skipping or repeating a match; 0 = nowhere left.
	ResumeSeq uint64
}

// MessageSearchDone summarizes a finished search run.
type MessageSearchDone struct {
	Scanned    uint64
	Matched    uint64
	Reason     SearchStopReason
	RangeFirst uint64
	RangeLast  uint64

	// NextSeq is where to continue; zero when the whole range was searched.
	NextSeq uint64
}

// MessageSearchEvent is one step of a search: progress, a batch of matches, or the summary.
type MessageSearchEvent struct {
	Progress *MessageSearchProgress
	Matches  []*Message
	Done     *MessageSearchDone
}
