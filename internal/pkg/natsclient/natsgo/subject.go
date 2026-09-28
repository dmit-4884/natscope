// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"fmt"
	"strings"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/dmit-4884/natscope/internal/errs"
)

// MaxNATSNameLength bounds any stream/consumer name, KV key, object name,
// filter subject, or publish/live subject natscope embeds into a JetStream
// API subject or a core NATS PUB control line. nats-server caps a single
// protocol line at max_control_line (4096 bytes by default); a name anywhere
// near that blows the limit once wrapped in "PUB <subject> ..." or
// "$JS.API...<name>", and the server responds by closing the whole shared
// connection ("maximum control line exceeded") — taking every other request
// and live subscription on that connection down with it. 255 matches NATS's
// own documented stream/consumer name limit and leaves a wide safety margin
// for longer subjects/filters.
const MaxNATSNameLength = 255

// validateNATSNameLength rejects a name/subject/filter longer than
// MaxNATSNameLength before it is embedded in a JetStream API subject or PUB
// line. kind identifies the offending field in the returned error.
func validateNATSNameLength(kind, value string) error {
	if len(value) > MaxNATSNameLength {
		return &errs.NATSValidationError{
			Description: fmt.Sprintf("%s exceeds the maximum length of %d characters", kind, MaxNATSNameLength),
		}
	}
	return nil
}

// validateConsumerRequestLengths checks the stream name, consumer name, and
// filter subject(s) of a consumer create/update request against
// MaxNATSNameLength (QA-014).
func validateConsumerRequestLengths(streamName, consumerName, filterSubject string, filterSubjects []string) error {
	if err := validateNATSNameLength("stream name", streamName); err != nil {
		return err
	}
	if err := validateNATSNameLength("consumer name", consumerName); err != nil {
		return err
	}
	if err := validateNATSNameLength("filter subject", filterSubject); err != nil {
		return err
	}
	for _, s := range filterSubjects {
		if err := validateNATSNameLength("filter subject", s); err != nil {
			return err
		}
	}
	return nil
}

// invalidConsumerNameChars mirrors the character denylist jetstream SDK's
// (unexported) validateConsumerName applies before building a subject from a
// consumer name — kept in sync so both reject the same names.
const invalidConsumerNameChars = ">*. /\\\t\r\n"

// validateConsumerNameChars rejects a consumer name containing a character
// JetStream itself disallows there, before it is embedded by hand in a
// "$JS.API.CONSUMER...<name>" subject (QA-050). The error wraps
// jetstream.ErrInvalidConsumerName so wrapErr classifies it exactly like the
// SDK's own client-side rejection.
func validateConsumerNameChars(name string) error {
	if strings.ContainsAny(name, invalidConsumerNameChars) {
		return fmt.Errorf("%w: %q", jetstream.ErrInvalidConsumerName, name)
	}
	return nil
}

// containsWildcard checks if a subject pattern contains NATS wildcards.
func containsWildcard(pattern string) bool {
	for _, c := range pattern {
		if c == '*' || c == '>' {
			return true
		}
	}
	return false
}
