// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"fmt"
	"strings"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/dmit-4884/natscope/internal/errs"
)

// Length caps keep user input in a JetStream API subject or PUB line under nats-server's max_control_line.
const (
	// MaxNATSNameLength is NATS's own stream/consumer name limit.
	MaxNATSNameLength = 255
	// MaxNATSSubjectLength bounds subjects, filters, KV keys and object names.
	MaxNATSSubjectLength = 1024
)

// validateNATSNameLength rejects a stream or consumer name longer than MaxNATSNameLength.
func validateNATSNameLength(kind, value string) error {
	return validateLength(kind, value, MaxNATSNameLength)
}

// validateNATSSubjectLength rejects a subject, filter, KV key or object name longer than MaxNATSSubjectLength.
func validateNATSSubjectLength(kind, value string) error {
	return validateLength(kind, value, MaxNATSSubjectLength)
}

func validateLength(kind, value string, limit int) error {
	if len(value) > limit {
		return &errs.NATSValidationError{
			Description: fmt.Sprintf("%s exceeds the maximum length of %d characters", kind, limit),
		}
	}
	return nil
}

// validateConsumerRequestLengths checks the name and filter subject lengths of a consumer request.
func validateConsumerRequestLengths(streamName, consumerName, filterSubject string, filterSubjects []string) error {
	if err := validateNATSNameLength("stream name", streamName); err != nil {
		return err
	}
	if err := validateNATSNameLength("consumer name", consumerName); err != nil {
		return err
	}
	if err := validateNATSSubjectLength("filter subject", filterSubject); err != nil {
		return err
	}
	for _, s := range filterSubjects {
		if err := validateNATSSubjectLength("filter subject", s); err != nil {
			return err
		}
	}
	return nil
}

// invalidConsumerNameChars mirrors the jetstream SDK's consumer-name denylist.
const invalidConsumerNameChars = ">*. /\\\t\r\n"

// validateConsumerNameChars rejects characters JetStream disallows in a consumer name. The error wraps
// jetstream.ErrInvalidConsumerName so wrapErr classifies it like the SDK's own check.
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
