// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package mcptransport

import (
	"reflect"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/pkg/convcodecs"
)

const noneName = "none"

// ViewCodecs renders durations and NATS enums as readable strings when converting entities to tool views.
var ViewCodecs = converter.WithCodecs(
	convcodecs.DurationString,
	convcodecs.EnumNames(map[reflect.Type][]string{
		reflect.TypeFor[entities.RetentionPolicy]():  {"limits", "interest", "workqueue"},
		reflect.TypeFor[entities.StorageType]():      {"file", "memory"},
		reflect.TypeFor[entities.DiscardPolicy]():    {"old", "new"},
		reflect.TypeFor[entities.StoreCompression](): {noneName, "s2"},
		reflect.TypeFor[entities.DeliverPolicy]():    {"all", "last", "new", "by_start_sequence", "by_start_time", "last_per_subject"},
		reflect.TypeFor[entities.AckPolicy]():        {"explicit", "all", noneName},
		reflect.TypeFor[entities.ReplayPolicy]():     {"instant", "original"},
		reflect.TypeFor[entities.AuthMethod]():       {noneName, "user_pass", "token", "nkey", "credentials"},
	}),
)

// Limit returns v bounded to [1, maxValue], or def when v is not positive.
func Limit(v, def, maxValue int) int {
	if v <= 0 {
		return def
	}
	return min(v, maxValue)
}

// Since parses an RFC 3339 timestamp or a duration counted back from now ("15m", "2h30m").
func Since(v string, now time.Time) (*time.Time, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil, nil //nolint:nilnil // absent value
	}
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return &t, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return nil, Errorf("since %q is neither an RFC 3339 timestamp nor a positive duration like 15m", v)
	}
	t := now.Add(-d)
	return &t, nil
}

// ReadOnly annotates a tool that only reads.
func ReadOnly(title string) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{Title: title, ReadOnlyHint: true}
}

// Items returns s, or an empty slice for nil, so an empty list reaches the agent as [] rather than null.
func Items[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}
