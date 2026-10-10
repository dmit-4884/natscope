// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

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
		{name: "InboxPrefix", subject: "_INBOX.abc.def", want: true},
		{name: "InboxOnly", subject: "_INBOX", want: true},
		{name: "UnderscoreApplicationSubject", subject: "_audit.login", want: false},
		{name: "InboxLookalike", subject: "_INBOXES.x", want: false},
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

func TestValidateSubjectPattern(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		subject string
		wantErr bool
	}{
		{name: "Literal", subject: "orders.created", wantErr: false},
		{name: "StarToken", subject: "orders.*", wantErr: false},
		{name: "GreaterLast", subject: "orders.>", wantErr: false},
		{name: "GreaterAlone", subject: ">", wantErr: false},
		{name: "Empty", subject: "", wantErr: true},
		{name: "ConsecutiveDots", subject: "a..b", wantErr: true},
		{name: "LeadingDot", subject: ".a", wantErr: true},
		{name: "TrailingDot", subject: "a.", wantErr: true},
		{name: "GreaterNotLast", subject: "a.>.b", wantErr: true},
		{name: "GreaterNotLastShort", subject: ">.a", wantErr: true},
		{name: "PartialStarToken", subject: "a.b*", wantErr: true},
		{name: "PartialGreaterToken", subject: "a.>x", wantErr: true},
		{name: "Space", subject: "qa ui space", wantErr: true},
		{name: "ControlChar", subject: "orders.ctl\u0001", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateSubjectPattern(tt.subject)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateSubjectPattern(%q) error = %v, wantErr %v", tt.subject, err, tt.wantErr)
			}
		})
	}
}

func TestValidateLiteralSubject(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		subject string
		wantErr bool
	}{
		{name: "Literal", subject: "orders.created", wantErr: false},
		{name: "Star", subject: "orders.*", wantErr: true},
		{name: "Greater", subject: "orders.>", wantErr: true},
		{name: "Empty", subject: "", wantErr: true},
		{name: "WhitespaceOnly", subject: "   ", wantErr: true},
		{name: "ConsecutiveDots", subject: "orders..created", wantErr: true},
		{name: "ControlChar", subject: "orders.ctl\u0001", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateLiteralSubject(tt.subject)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateLiteralSubject(%q) error = %v, wantErr %v", tt.subject, err, tt.wantErr)
			}
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

func TestSubjectsCollide(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a, b string
		want bool
	}{
		{name: "Equal", a: "orders.new", b: "orders.new", want: true},
		{name: "DifferentLiterals", a: "orders.new", b: "orders.old", want: false},
		{name: "StarMatchesToken", a: "orders.*", b: "orders.new", want: true},
		{name: "StarNeedsSameLength", a: "orders.*", b: "orders.new.eu", want: false},
		{name: "TailMatchesRest", a: "orders.>", b: "orders.new.eu", want: true},
		{name: "TailNeedsOneToken", a: "orders.>", b: "orders", want: false},
		{name: "TailAgainstStar", a: "orders.>", b: "*.new", want: true},
		{name: "StarAgainstStar", a: "*.new", b: "orders.*", want: true},
		{name: "FullWildcard", a: ">", b: "anything.at.all", want: true},
		{name: "PrefixMismatch", a: "repub.>", b: "orders.>", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, SubjectsCollide(tt.a, tt.b))
			assert.Equal(t, tt.want, SubjectsCollide(tt.b, tt.a))
		})
	}
}

func TestTransformDestinationPattern(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		dest string
		want string
	}{
		{name: "Literal", dest: "audit.orders", want: "audit.orders"},
		{name: "Tail", dest: "repub.>", want: "repub.>"},
		{name: "DollarWildcard", dest: "repub.$1.done", want: "repub.*.done"},
		{name: "WildcardFunction", dest: "repub.{{wildcard(1)}}.done", want: "repub.*.done"},
		{name: "PartitionFunction", dest: "part.{{ partition(3,1) }}.x", want: "part.*.x"},
		{name: "SplitFunction", dest: "a.{{SplitFromLeft(1,2)}}.b", want: "a.>"},
		{name: "SliceFunction", dest: "{{sliceFromRight(1,2)}}", want: ">"},
		{name: "DollarWithoutDigits", dest: "$KV.bucket.>", want: "$KV.bucket.>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, TransformDestinationPattern(tt.dest))
		})
	}
}
