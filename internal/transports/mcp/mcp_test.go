// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package mcptransport

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"

	connectionssvc "github.com/dmit-4884/natscope/internal/services/connections"
)

func TestNewBody(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		data          []byte
		limit         int
		truncated     bool
		want          *Body
		wantTruncated bool
	}{
		{name: "empty", data: nil, limit: 10},
		{name: "json object", data: []byte(` {"a":1} `), limit: 0, want: &Body{JSON: []byte(`{"a":1}`)}},
		{name: "json array", data: []byte(`[1,2]`), limit: 10, want: &Body{JSON: []byte(`[1,2]`)}},
		{name: "json scalar stays text", data: []byte(`42`), limit: 10, want: &Body{Text: "42"}},
		{name: "whitespace only", data: []byte("  "), limit: 10, want: &Body{Text: "  "}},
		{name: "text", data: []byte("hello"), limit: 10, want: &Body{Text: "hello"}},
		{name: "clipped json becomes text", data: []byte(`{"a":"long"}`), limit: 5, want: &Body{Text: `{"a":`}, wantTruncated: true},
		{name: "clip drops a split rune", data: []byte("héllo"), limit: 2, want: &Body{Text: "h"}, wantTruncated: true},
		{name: "upstream truncation drops a split rune", data: []byte("h\xc3"), limit: 10, truncated: true, want: &Body{Text: "h"}, wantTruncated: true},
		{name: "binary", data: []byte{0x00, 0xff, 0xfe}, limit: 10, want: &Body{Base64: base64.StdEncoding.EncodeToString([]byte{0x00, 0xff, 0xfe})}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, truncated := NewBody(tt.data, tt.limit, tt.truncated)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantTruncated, truncated)
		})
	}
}

func TestNewBodyFromBase64(t *testing.T) {
	t.Parallel()

	got, truncated := NewBodyFromBase64(base64.StdEncoding.EncodeToString([]byte("hi")), 10, false)
	assert.Equal(t, &Body{Text: "hi"}, got)
	assert.False(t, truncated)

	got, _ = NewBodyFromBase64("not base64!", 10, false)
	assert.Equal(t, &Body{Base64: "not base64!"}, got)
}

func TestSince(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)

	got, err := Since("", now)
	require.NoError(t, err)
	assert.Nil(t, got)

	got, err = Since("15m", now)
	require.NoError(t, err)
	assert.Equal(t, now.Add(-15*time.Minute), *got)

	got, err = Since("2026-09-28T10:00:00Z", now)
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC), got.UTC())

	for _, bad := range []string{"yesterday", "-5m", "0s"} {
		_, err = Since(bad, now)
		var ae *agentError
		assert.ErrorAs(t, err, &ae, bad)
	}
}

func TestLimit(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 20, Limit(0, 20, 100))
	assert.Equal(t, 20, Limit(-3, 20, 100))
	assert.Equal(t, 7, Limit(7, 20, 100))
	assert.Equal(t, 100, Limit(500, 20, 100))
}

type fakeConnections struct {
	connectionssvc.Service
	items    entities.SavedConnections
	pageSize int
	err      error
}

func (f *fakeConnections) List(_ context.Context, in *entities.SavedConnectionsList) (*entities.List[entities.SavedConnections], error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.pageSize == 0 {
		return &entities.List[entities.SavedConnections]{Items: f.items}, nil
	}
	start := 0
	if in.Cursor != "" {
		fmt.Sscan(in.Cursor, &start) //nolint:errcheck
	}
	end := min(start+f.pageSize, len(f.items))
	page := &entities.List[entities.SavedConnections]{Items: f.items[start:end]}
	if end < len(f.items) {
		page.NextCursor = new(fmt.Sprint(end))
	}
	return page, nil
}

func savedConnection(id, name string) *entities.SavedConnection {
	return &entities.SavedConnection{Id: id, Name: name}
}

func TestConnectionsResolve(t *testing.T) {
	t.Parallel()
	two := entities.SavedConnections{savedConnection("id-1", "local"), savedConnection("id-2", "Staging")}
	twins := entities.SavedConnections{savedConnection("id-3", "Prod"), savedConnection("id-4", "prod")}

	tests := []struct {
		name    string
		items   entities.SavedConnections
		ref     string
		want    string
		wantErr string
	}{
		{name: "none saved", ref: "", wantErr: "no saved connections: add one in the natscope UI first"},
		{name: "single saved, ref omitted", items: two[:1], want: "id-1"},
		{name: "several saved, ref omitted", items: two, wantErr: "several connections are saved (local, Staging): pass `connection`"},
		{name: "by id", items: two, ref: "id-2", want: "id-2"},
		{name: "by name, case-insensitive", items: two, ref: " staging ", want: "id-2"},
		{name: "unknown", items: two, ref: "prod", wantErr: `connection "prod" not found; saved connections: local, Staging`},
		{name: "exact name beats a case-insensitive twin", items: twins, ref: "prod", want: "id-4"},
		{name: "case-insensitive twins are ambiguous", items: twins, ref: "PROD",
			wantErr: `connection "PROD" matches several names (Prod, prod): pass the exact name or id`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := NewConnections(&fakeConnections{items: tt.items}).Resolve(t.Context(), tt.ref)
			if tt.wantErr != "" {
				var ae *agentError
				require.ErrorAs(t, err, &ae)
				assert.Equal(t, tt.wantErr, ae.msg)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConnectionsListFollowsCursor(t *testing.T) {
	t.Parallel()
	items := entities.SavedConnections{savedConnection("a", "a"), savedConnection("b", "b"), savedConnection("c", "c")}
	all, err := NewConnections(&fakeConnections{items: items, pageSize: 2}).List(t.Context())
	require.NoError(t, err)
	assert.Equal(t, items, all)
}

func TestConnectionsResolve_PropagatesListError(t *testing.T) {
	t.Parallel()
	boom := errors.New("bbolt closed")
	_, err := NewConnections(&fakeConnections{err: boom}).Resolve(t.Context(), "")
	require.ErrorIs(t, err, boom)
}

func TestToolError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "agent error verbatim", err: Errorf("pass type or subject"), want: "pass type or subject"},
		{name: "wrapped agent error", err: fmt.Errorf("resolve: %w", Errorf("connection %q not found", "x")), want: `connection "x" not found`},
		{name: "domain sentinel", err: fmt.Errorf("get stream: %w", errs.ErrStreamNotFound), want: "stream not found [NATS_STREAM_NOT_FOUND]"},
		{name: "validation error keeps its description", err: &errs.NATSValidationError{Description: "subject must not be empty"},
			want: "subject must not be empty [NATS_INVALID_ARGUMENT]"},
		{name: "unmapped error is sanitized", err: errors.New("open /Users/secret/natscope.bolt: denied"), want: "internal error [INTERNAL]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.EqualError(t, toolError(t.Context(), tt.err), tt.want)
		})
	}
}

func TestInstructionsReflectWriteMode(t *testing.T) {
	t.Parallel()
	assert.Contains(t, instructions(true), "publish_message publishes")
	assert.Contains(t, instructions(true), "request_message sends")
	assert.Contains(t, instructions(false), "read-only")
}

func TestItems(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []int{}, Items[int](nil))
	assert.Equal(t, []int{1}, Items([]int{1}))
}

func TestAddToolNormalizesInput(t *testing.T) {
	t.Parallel()
	type echoInput struct {
		Subject   string `json:"subject" normalize:"trim"`
		Direction string `json:"direction" normalize:"trim,lowercase"`
	}
	type echoOutput struct {
		Subject   string `json:"subject"`
		Direction string `json:"direction"`
	}
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "v0"}, nil)
	AddTool(server, &mcp.Tool{Name: "echo"}, func(_ context.Context, _ *mcp.CallToolRequest, in echoInput) (*mcp.CallToolResult, echoOutput, error) {
		return nil, echoOutput(in), nil
	})
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	_, err := server.Connect(t.Context(), serverTransport, nil)
	require.NoError(t, err)
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "client", Version: "v0"}, nil).Connect(t.Context(), clientTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cs.Close() })

	res, err := cs.CallTool(t.Context(), &mcp.CallToolParams{
		Name:      "echo",
		Arguments: map[string]any{"subject": "  orders.>  ", "direction": " Backward "},
	})

	require.NoError(t, err)
	require.False(t, res.IsError)
	assert.Equal(t, map[string]any{"subject": "orders.>", "direction": "backward"}, res.StructuredContent)
}
