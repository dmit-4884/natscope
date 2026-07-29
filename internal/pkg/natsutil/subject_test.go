// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsInternalSubject(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		subject string
		want    bool
	}{
		{name: "EmptyString", subject: "", want: true},
		{name: "DollarPrefix", subject: "$SYS.monitor", want: true},
		{name: "DollarOnly", subject: "$", want: true},
		{name: "UnderscorePrefix", subject: "_INBOX.abc.def", want: true},
		{name: "UnderscoreOnly", subject: "_", want: true},
		// "handler." is an application prefix, not a NATS namespace: it must be
		// delivered like any other user subject.
		{name: "HandlerDotPrefix", subject: "handler.something", want: false},
		{name: "HandlerDotExact", subject: "handler.", want: false},
		{name: "HandlerDotLong", subject: "handler.a.b.c", want: false},
		{name: "NormalSubject", subject: "orders.created", want: false},
		{name: "SingleChar", subject: "a", want: false},
		{name: "HandlerWithoutDot", subject: "handlers", want: false},
		{name: "HandlerShort", subject: "handle", want: false},
		{name: "HandlerExactNoTrail", subject: "handler", want: false},
		{name: "NumberPrefix", subject: "123.test", want: false},
		{name: "DotPrefix", subject: ".hidden", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := IsInternalSubject(tt.subject)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMatchSubject(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		pattern string
		subject string
		want    bool
	}{
		{name: "ExactMatch", pattern: "orders.created", subject: "orders.created", want: true},
		{name: "ExactMismatch", pattern: "orders.created", subject: "orders.updated", want: false},
		{name: "WildcardStarSingle", pattern: "orders.*", subject: "orders.created", want: true},
		{name: "WildcardStarMultiSegment", pattern: "orders.*", subject: "orders.eu.created", want: false},
		{name: "WildcardStarMiddle", pattern: "orders.*.created", subject: "orders.eu.created", want: true},
		{name: "WildcardStarMiddleMismatch", pattern: "orders.*.created", subject: "orders.eu.updated", want: false},
		{name: "WildcardGreaterThan", pattern: "orders.>", subject: "orders.eu.created", want: true},
		{name: "WildcardGreaterThanSingle", pattern: "orders.>", subject: "orders.created", want: true},
		{name: "GreaterThanMatchesAll", pattern: ">", subject: "anything.here.works", want: true},
		{name: "GreaterThanSingleToken", pattern: ">", subject: "single", want: true},
		{name: "StarSingleToken", pattern: "*", subject: "single", want: true},
		{name: "StarMultiToken", pattern: "*", subject: "a.b", want: false},
		{name: "PatternLongerThanSubject", pattern: "a.b.c", subject: "a.b", want: false},
		{name: "SubjectLongerThanPattern", pattern: "a.b", subject: "a.b.c", want: false},
		{name: "BothEmpty", pattern: "", subject: "", want: true},
		{name: "MultipleStars", pattern: "*.*.created", subject: "eu.orders.created", want: true},
		{name: "MultipleStarsMismatch", pattern: "*.*.created", subject: "eu.orders.updated", want: false},
		{name: "MixedWildcards", pattern: "*.orders.>", subject: "eu.orders.created.v2", want: true},
		{name: "EmptyPatternNonEmptySubject", pattern: "", subject: "a", want: false},
		{name: "NonEmptyPatternEmptySubject", pattern: "a", subject: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := MatchSubject(tt.pattern, tt.subject)
			assert.Equal(t, tt.want, got)
		})
	}
}
