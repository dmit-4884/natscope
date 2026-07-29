// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsutil

import (
	"testing"
	"time"

	"github.com/dmit-4884/natscope/internal/entities"
)

func mapping(id, pattern, msgType, sourceID string, createdAt int64) *entities.SubjectMapping {
	m := entities.SubjectMappingNew(func(m *entities.SubjectMapping) {
		m.Pattern = pattern
		m.MessageType = msgType
		m.SourceID = sourceID
	})
	m.Id = id
	m.CreatedAt = time.UnixMilli(createdAt).UTC()
	return m
}

func TestResolver_ExactBeatsWildcard(t *testing.T) {
	exact := mapping("e", "orders.created", "T1", "src", 100)
	wild := mapping("w", "orders.*", "T2", "src", 50)

	r := NewMappingResolver(entities.SubjectMappings{wild, exact})
	got := r.Resolve("orders.created")
	if got == nil || got.Id != "e" {
		t.Fatalf("expected exact mapping to win, got %#v", got)
	}
}

func TestResolver_HigherSpecificityWins(t *testing.T) {
	star := mapping("a", "orders.*", "T1", "src", 100)
	greedy := mapping("b", "orders.>", "T2", "src", 50)

	r := NewMappingResolver(entities.SubjectMappings{greedy, star})
	got := r.Resolve("orders.created")
	if got == nil || got.Id != "a" {
		t.Fatalf("expected orders.* to beat orders.>, got %#v", got)
	}
}

func TestResolver_LongerLiteralChainWins(t *testing.T) {
	short := mapping("a", "events.>", "T1", "src", 100)
	long := mapping("b", "events.audit.>", "T2", "src", 100)

	r := NewMappingResolver(entities.SubjectMappings{short, long})
	got := r.Resolve("events.audit.created")
	if got == nil || got.Id != "b" {
		t.Fatalf("expected events.audit.> to win, got %#v", got)
	}
}

func TestResolver_LexTieBreak(t *testing.T) {
	a := mapping("a", "alpha.*", "T1", "src", 100)
	b := mapping("b", "beta.*", "T2", "src", 100)

	r := NewMappingResolver(entities.SubjectMappings{b, a})
	// neither matches "alpha.x" except a
	got := r.Resolve("alpha.x")
	if got == nil || got.Id != "a" {
		t.Fatalf("expected alpha.* to match, got %#v", got)
	}
}

func TestResolver_CreatedAtTieBreakSamePattern(t *testing.T) {
	older := mapping("old", "orders.*", "T1", "src1", 100)
	newer := mapping("new", "orders.*", "T2", "src2", 200)

	// Both wildcard with identical pattern; same specificity, same lex order.
	// Tie-break by CreatedAt: older should win.
	r := NewMappingResolver(entities.SubjectMappings{newer, older})
	got := r.Resolve("orders.created")
	if got == nil || got.Id != "old" {
		t.Fatalf("expected older mapping to win on tie-break, got %#v", got)
	}
}

func TestResolver_NilMappingsAreIgnored(t *testing.T) {
	a := mapping("a", "orders.*", "T1", "src", 100)
	r := NewMappingResolver(entities.SubjectMappings{nil, a, nil})
	got := r.Resolve("orders.created")
	if got == nil || got.Id != "a" {
		t.Fatalf("expected resolver to skip nils and resolve a, got %#v", got)
	}
}

func TestResolver_NoMatch(t *testing.T) {
	r := NewMappingResolver(entities.SubjectMappings{
		mapping("a", "orders.*", "T1", "src", 100),
	})
	if got := r.Resolve("users.created"); got != nil {
		t.Fatalf("expected no match, got %#v", got)
	}
}

func TestResolver_NilSafe(t *testing.T) {
	var r *MappingResolver
	if got := r.Resolve("anything"); got != nil {
		t.Fatalf("expected nil result on nil resolver")
	}
}

func TestSpecificity(t *testing.T) {
	cases := []struct {
		pattern string
		want    int
	}{
		{"orders.created", 20},
		{"orders.*", 11},
		{"orders.>", 5},
		{"events.audit.created", 30},
		{"events.audit.*", 21},
		{"events.>", 5},
		{">", -5},
		{"*", 1},
	}
	for _, c := range cases {
		if got := specificity(c.pattern); got != c.want {
			t.Errorf("specificity(%q) = %d, want %d", c.pattern, got, c.want)
		}
	}
}
